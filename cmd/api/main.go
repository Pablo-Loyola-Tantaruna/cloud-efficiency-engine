package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud-efficiency-engine/internal/analysis"
	"cloud-efficiency-engine/internal/analysis/actions"
	"cloud-efficiency-engine/internal/analysis/optimizer"
	"cloud-efficiency-engine/internal/analysis/resolver"
	"cloud-efficiency-engine/internal/analysis/rules"
	"cloud-efficiency-engine/internal/api"
	rediscache "cloud-efficiency-engine/internal/cache/redis"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
	kubeclient "cloud-efficiency-engine/internal/kubernetes"
	metricproviders "cloud-efficiency-engine/internal/metrics/providers"
	"cloud-efficiency-engine/internal/observability"
	postgrespersistence "cloud-efficiency-engine/internal/persistence/postgres"
	providerregistry "cloud-efficiency-engine/internal/providers"
	awsprovider "cloud-efficiency-engine/internal/providers/aws"
	azureprovider "cloud-efficiency-engine/internal/providers/azure"
	gcpprovider "cloud-efficiency-engine/internal/providers/gcp"
	kubernetesprovider "cloud-efficiency-engine/internal/providers/kubernetes"
	"cloud-efficiency-engine/internal/security"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultLookbackHours    = 7 * 24
	defaultHistoryStep      = 5 * time.Minute
	defaultHoursPerMonth    = 730.0
	defaultAnalysisInterval = 15 * time.Minute
	defaultPrometheusURL    = "http://localhost:9090"
)

func main() {
	logger := api.NewLogger()

	tracingShutdown, tracingErr := observability.InitTracing(context.Background(), "cloud-efficiency-engine")
	if tracingErr != nil {
		logger.Warn("tracing_initialization_failed", "error", tracingErr)
	}
	defer func() {
		if err := tracingShutdown(context.Background()); err != nil {
			logger.Warn("tracing_shutdown_failed", "error", err)
		}
	}()

	optimizationRules := []rules.Rule{
		rules.NewCPUOverprovisioningRule(),
		rules.NewMemoryOverprovisioningRule(),
	}

	calculator := cost.NewCalculator(defaultHoursPerMonth)
	recommendationResolver := resolver.NewResolver()
	analysisContext := loadAnalysisContext()
	registry := providerregistry.NewRegistry()

	persistencePool, err := initializePersistence()
	if err != nil {
		logger.Error("persistence_initialization_failed", "error", err)
		return
	}
	if persistencePool != nil {
		defer persistencePool.Close()
		logger.Info("persistence_ready")
	}

	redisClient := initializeRedis(logger)
	if redisClient != nil {
		defer redisClient.Close()
	}

	authMiddleware, authErr := security.MiddlewareFromEnv()
	if authErr != nil && !errors.Is(authErr, security.ErrAuthenticationDisabled) {
		logger.Error("security_initialization_failed", "error", authErr)
		return
	}
	if errors.Is(authErr, security.ErrAuthenticationDisabled) {
		logger.Warn("finops_authentication_disabled", "warning", "explicit FINOPS_AUTH_MODE=disabled")
	}

	prometheusProvider := metricproviders.NewPrometheusProvider(getPrometheusURL(), nil)
	kubernetesCapacitySource := kubernetesprovider.NewCapacitySource(prometheusProvider)
	kubernetesCapacityProvider := kubernetesprovider.NewCapacityProvider(kubernetesCapacitySource)

	if err := kubernetesprovider.Register(
		registry,
		getPrometheusURL(),
		0.04,
		0.005,
		kubernetesCapacityProvider,
	); err != nil {
		logger.Error("kubernetes_provider_registration_failed", "error", err)
		return
	}

	var gcpClients *gcpprovider.Clients

	switch analysisContext.Provider {
	case domain.CloudProviderAWS:
		if err := registerAWSProvider(registry, analysisContext, getPrometheusURL()); err != nil {
			logger.Error("aws_provider_registration_failed", "error", err)
			return
		}
	case domain.CloudProviderAzure:
		if err := registerAzureProvider(registry, analysisContext, getPrometheusURL()); err != nil {
			logger.Error("azure_provider_registration_failed", "error", err)
			return
		}
	case domain.CloudProviderGCP:
		var err error
		gcpClients, err = registerGCPProvider(registry, analysisContext, getPrometheusURL())
		if err != nil {
			logger.Error("gcp_provider_registration_failed", "error", err)
			return
		}
	}

	if gcpClients != nil {
		defer func() {
			if err := gcpClients.Close(); err != nil {
				logger.Error("gcp_clients_close_failed", "error", err)
			}
		}()
	}

	engine := analysis.NewEngineWithRegistry(
		registry,
		optimizationRules,
		optimizer.DefaultOptimizationPolicy(),
		recommendationResolver,
		calculator,
	)

	httpMetrics := api.NewHTTPMetrics()
	analysisMetrics := api.NewAnalysisMetrics()
	observabilityMetrics := observability.NewMetrics()

	handler := api.NewHandler(engine, analysisMetrics)
	workloadExecutor := initializeWorkloadExecutor(logger)
	finOpsHandler := initializeFinOpsHandler(persistencePool, redisClient, workloadExecutor)
	var router http.Handler
	if authMiddleware != nil {
		router = api.NewSecureFinOpsRouter(
			handler,
			logger,
			httpMetrics,
			analysisMetrics,
			finOpsHandler,
			authMiddleware,
			observabilityMetrics.Handler(),
		)
	} else {
		router = api.NewFinOpsRouter(
			handler,
			logger,
			httpMetrics,
			analysisMetrics,
			finOpsHandler,
			observabilityMetrics.Handler(),
		)
	}

	server := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	analysisInterval := parseDurationEnv("ANALYSIS_INTERVAL", defaultAnalysisInterval)
	scheduler := analysis.NewScheduler(
		engine,
		observabilityMetrics,
		logger,
		analysis.SchedulerConfig{
			Namespace:     analysisContextNamespace(),
			Context:       analysisContext,
			Interval:      analysisInterval,
			LookbackHours: defaultLookbackHours,
			Step:          defaultHistoryStep,
		},
	)

	go scheduler.Run(ctx)

	serverError := make(chan error, 1)
	go func() {
		logger.Info(
			"http_server_started",
			"addr", server.Addr,
			"provider", analysisContext.Provider,
			"environment", analysisContext.Environment,
		)
		serverError <- server.ListenAndServe()
	}()

	select {
	case err := <-serverError:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("http_server_failed", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		logger.Info("shutdown_started")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("http_server_shutdown_failed", "error", err)
		return
	}

	logger.Info("shutdown_completed")
}

func initializePersistence() (*pgxpool.Pool, error) {
	if os.Getenv("DATABASE_URL") == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cfg, err := postgrespersistence.ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	pool, err := postgrespersistence.NewPool(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := postgrespersistence.ApplyMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply persistence migrations: %w", err)
	}
	return pool, nil
}

func initializeRedis(logger interface{ Warn(string, ...any) }) *rediscache.Client {
	client, err := rediscache.NewFromEnv()
	if errors.Is(err, rediscache.ErrNotConfigured) {
		return nil
	}
	if err != nil {
		logger.Warn("redis_initialization_failed", "error", err)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		logger.Warn("redis_unavailable", "error", err)
		_ = client.Close()
		return nil
	}
	return client
}

func initializeWorkloadExecutor(logger interface{ Warn(string, ...any) }) *kubernetesprovider.WorkloadExecutor {
	client, err := kubeclient.NewClientFromEnv()
	if errors.Is(err, kubeclient.ErrNotConfigured) {
		logger.Warn("workload_execution_disabled", "warning", "FINOPS_KUBECONFIG or in-cluster Kubernetes configuration is not available")
		return nil
	}
	if err != nil {
		logger.Warn("kubernetes_client_initialization_failed", "error", err)
		return nil
	}
	return kubernetesprovider.NewWorkloadExecutor(client)
}

func initializeFinOpsHandler(pool *pgxpool.Pool, redisClient *rediscache.Client, workloadExecutor *kubernetesprovider.WorkloadExecutor) *api.FinOpsHandler {
	if pool == nil {
		return nil
	}
	repositories, err := postgrespersistence.NewRepositories(pool)
	if err != nil {
		return nil
	}

	var executionEngine *actions.ExecutionEngine
	if workloadExecutor != nil {
		executionService := actions.NewExecutionService(repositories.Execution)
		resolver := providerregistry.NewStaticExecutorResolver(map[domain.CloudProvider]domain.ProviderExecutor{
			domain.CloudProviderAWS:        workloadExecutor,
			domain.CloudProviderAzure:      workloadExecutor,
			domain.CloudProviderGCP:        workloadExecutor,
			domain.CloudProviderKubernetes: workloadExecutor,
		})
		verifier := actions.NewVerificationService(workloadExecutor)
		executionEngine = actions.NewExecutionEngine(executionService, resolver, verifier)
	}

	return api.NewFinOpsHandler(
		repositories.ActionPlan,
		repositories.Approval,
		repositories.Recovery,
		repositories.Execution,
		repositories.Audit,
		repositories.Verification,
		executionEngine,
		redisClient,
	)
}

func loadAnalysisContext() domain.AnalysisContext {
	provider := domain.CloudProvider(getEnv("CLOUD_PROVIDER", string(domain.CloudProviderKubernetes)))
	environment := getEnv("ANALYSIS_ENVIRONMENT", "development")

	return domain.NormalizeAnalysisContext(domain.AnalysisContext{
		Provider:    provider,
		Environment: environment,
		AccountID:   os.Getenv("CLOUD_ACCOUNT_ID"),
		Region:      os.Getenv("CLOUD_REGION"),
		ClusterName: os.Getenv("CLOUD_CLUSTER_NAME"),
	})
}

func analysisContextNamespace() string {
	return getEnv("ANALYSIS_NAMESPACE", "cloud-efficiency-engine")
}

func getPrometheusURL() string {
	return getEnv("PROMETHEUS_URL", defaultPrometheusURL)
}

func getEnv(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func parseDurationEnv(name string, fallback time.Duration) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func registerAWSProvider(registry *providerregistry.Registry, analysisContext domain.AnalysisContext, prometheusURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	clients, err := awsprovider.LoadSDKClients(ctx, analysisContext.Region)
	if err != nil {
		return err
	}

	prometheusProvider := metricproviders.NewPrometheusProvider(prometheusURL, nil)
	metricsSource := providerregistry.NewMetricsSourceAdapter(prometheusProvider, prometheusProvider)
	pricingClient := awsprovider.NewAWSPricingClient(clients.EC2, clients.Pricing)
	if err := awsprovider.RegisterWithSources(registry, metricsSource, pricingClient); err != nil {
		return err
	}

	billingClient := awsprovider.NewCostExplorerBillingClient(clients.CostExplorer)
	if err := awsprovider.RegisterBillingProvider(registry, billingClient); err != nil {
		return err
	}

	return awsprovider.RegisterCapacityProvider(registry, clients.EC2)
}

func registerAzureProvider(registry *providerregistry.Registry, analysisContext domain.AnalysisContext, prometheusURL string) error {
	if analysisContext.AccountID == "" {
		return fmt.Errorf("Azure subscription ID must be configured in CLOUD_ACCOUNT_ID")
	}

	clients, err := azureprovider.NewClients(analysisContext.AccountID)
	if err != nil {
		return err
	}

	return azureprovider.RegisterRuntime(registry, clients, prometheusURL, nil)
}

func registerGCPProvider(registry *providerregistry.Registry, analysisContext domain.AnalysisContext, prometheusURL string) (*gcpprovider.Clients, error) {
	if analysisContext.AccountID == "" {
		return nil, fmt.Errorf("GCP project ID must be configured in CLOUD_ACCOUNT_ID")
	}
	if analysisContext.Region == "" {
		return nil, fmt.Errorf("GCP GKE location must be configured in CLOUD_REGION")
	}
	if analysisContext.ClusterName == "" {
		return nil, fmt.Errorf("GCP GKE cluster name must be configured in CLOUD_CLUSTER_NAME")
	}

	return gcpprovider.RegisterRuntime(
		context.Background(),
		registry,
		gcpprovider.ProjectContext{
			ProjectID: analysisContext.AccountID,
			Region:    analysisContext.Region,
			Cluster:   analysisContext.ClusterName,
		},
		prometheusURL,
		nil,
	)
}
