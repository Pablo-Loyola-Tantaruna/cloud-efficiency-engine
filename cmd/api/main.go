package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud-efficiency-engine/internal/analysis"
	"cloud-efficiency-engine/internal/analysis/optimizer"
	"cloud-efficiency-engine/internal/analysis/resolver"
	"cloud-efficiency-engine/internal/analysis/rules"
	"cloud-efficiency-engine/internal/api"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/observability"
	providerregistry "cloud-efficiency-engine/internal/providers"

	metricproviders "cloud-efficiency-engine/internal/metrics/providers"
	awsprovider "cloud-efficiency-engine/internal/providers/aws"

	kubernetesprovider "cloud-efficiency-engine/internal/providers/kubernetes"
)

const (
	defaultLookbackHours = 7 * 24

	defaultHistoryStep = 5 * time.Minute

	defaultHoursPerMonth = 730.0

	defaultAnalysisInterval = 15 * time.Minute

	defaultPrometheusURL = "http://localhost:9090"
)

func main() {

	logger :=
		api.NewLogger()

	optimizationRules :=
		[]rules.Rule{
			rules.NewCPUOverprovisioningRule(),

			rules.NewMemoryOverprovisioningRule(),
		}

	calculator :=
		cost.NewCalculator(
			defaultHoursPerMonth,
		)

	recommendationResolver :=
		resolver.NewResolver()

	analysisContext :=
		loadAnalysisContext()

	registry :=
		providerregistry.NewRegistry()

	prometheusProvider :=
		metricproviders.NewPrometheusProvider(
			getPrometheusURL(),
			nil,
		)

	kubernetesCapacitySource :=
		kubernetesprovider.NewCapacitySource(
			prometheusProvider,
		)

	kubernetesCapacityProvider :=
		kubernetesprovider.NewCapacityProvider(
			kubernetesCapacitySource,
		)

	if err :=
		kubernetesprovider.Register(
			registry,
			getPrometheusURL(),
			0.04,
			0.005,
			kubernetesCapacityProvider,
		); err != nil {

		logger.Error(
			"kubernetes_provider_registration_failed",
			"error",
			err,
		)

		return
	}

	if analysisContext.Provider ==
		domain.CloudProviderAWS {

		if err :=
			registerAWSProvider(
				registry,
				analysisContext,
				getPrometheusURL(),
			); err != nil {

			logger.Error(
				"aws_provider_registration_failed",
				"error",
				err,
			)

			return
		}
	}

	engine :=
		analysis.NewEngineWithRegistry(
			registry,
			optimizationRules,
			optimizer.DefaultOptimizationPolicy(),
			recommendationResolver,
			calculator,
		)

	httpMetrics :=
		api.NewHTTPMetrics()

	analysisMetrics :=
		api.NewAnalysisMetrics()

	observabilityMetrics :=
		observability.NewMetrics()

	handler :=
		api.NewHandler(
			engine,
			analysisMetrics,
		)

	router :=
		api.NewRouter(
			handler,
			logger,
			httpMetrics,
			analysisMetrics,
			observabilityMetrics.Handler(),
		)

	server :=
		&http.Server{
			Addr: ":8080",

			Handler: router,

			ReadHeaderTimeout: 5 * time.Second,

			ReadTimeout: 15 * time.Second,

			WriteTimeout: 30 * time.Second,

			IdleTimeout: 60 * time.Second,
		}

	ctx, stop :=
		signal.NotifyContext(
			context.Background(),
			os.Interrupt,
			syscall.SIGTERM,
		)

	defer stop()

	analysisInterval :=
		parseDurationEnv(
			"ANALYSIS_INTERVAL",
			defaultAnalysisInterval,
		)

	scheduler :=
		analysis.NewScheduler(
			engine,
			observabilityMetrics,
			logger,
			analysis.SchedulerConfig{
				Namespace: analysisContextNamespace(),

				Context: analysisContext,

				Interval: analysisInterval,

				LookbackHours: defaultLookbackHours,

				Step: defaultHistoryStep,
			},
		)

	go scheduler.Run(
		ctx,
	)

	serverError :=
		make(
			chan error,
			1,
		)

	go func() {

		logger.Info(
			"http_server_started",
			"addr",
			server.Addr,
			"provider",
			analysisContext.Provider,
			"environment",
			analysisContext.Environment,
		)

		serverError <- server.ListenAndServe()

	}()

	select {

	case err :=
		<-serverError:

		if err != nil &&
			err != http.ErrServerClosed {

			logger.Error(
				"http_server_failed",
				"error",
				err,
			)

			os.Exit(1)
		}

	case <-ctx.Done():

		logger.Info(
			"shutdown_started",
		)
	}

	shutdownContext, cancel :=
		context.WithTimeout(
			context.Background(),
			10*time.Second,
		)

	defer cancel()

	if err :=
		server.Shutdown(
			shutdownContext,
		); err != nil {

		logger.Error(
			"http_server_shutdown_failed",
			"error",
			err,
		)

		return
	}

	logger.Info(
		"shutdown_completed",
	)
}

func loadAnalysisContext() domain.AnalysisContext {

	provider :=
		domain.CloudProvider(
			getEnv(
				"CLOUD_PROVIDER",
				string(
					domain.CloudProviderKubernetes,
				),
			),
		)

	environment :=
		getEnv(
			"ANALYSIS_ENVIRONMENT",
			"development",
		)

	return domain.NormalizeAnalysisContext(
		domain.AnalysisContext{
			Provider: provider,

			Environment: environment,

			AccountID: os.Getenv(
				"CLOUD_ACCOUNT_ID",
			),

			Region: os.Getenv(
				"CLOUD_REGION",
			),

			ClusterName: os.Getenv(
				"CLOUD_CLUSTER_NAME",
			),
		},
	)
}

func analysisContextNamespace() string {

	return getEnv(
		"ANALYSIS_NAMESPACE",
		"cloud-efficiency-engine",
	)
}

func getPrometheusURL() string {

	return getEnv(
		"PROMETHEUS_URL",
		defaultPrometheusURL,
	)
}

func getEnv(
	name string,
	fallback string,
) string {

	value :=
		os.Getenv(
			name,
		)

	if value == "" {
		return fallback
	}

	return value
}

func parseDurationEnv(
	name string,
	fallback time.Duration,
) time.Duration {

	value :=
		os.Getenv(name)

	if value == "" {
		return fallback
	}

	parsed, err :=
		time.ParseDuration(
			value,
		)

	if err != nil {
		return fallback
	}

	if parsed <= 0 {
		return fallback
	}

	return parsed
}

func registerAWSProvider(
	registry *providerregistry.Registry,
	analysisContext domain.AnalysisContext,
	prometheusURL string,
) error {

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			15*time.Second,
		)

	defer cancel()

	clients, err :=
		awsprovider.LoadSDKClients(
			ctx,
			analysisContext.Region,
		)

	if err != nil {
		return err
	}

	prometheusProvider :=
		metricproviders.NewPrometheusProvider(
			prometheusURL,
			nil,
		)

	metricsSource :=
		providerregistry.NewMetricsSourceAdapter(
			prometheusProvider,
			prometheusProvider,
		)

	pricingClient :=
		awsprovider.NewAWSPricingClient(
			clients.EC2,
			clients.Pricing,
		)

	if err :=
		awsprovider.RegisterWithSources(
			registry,
			metricsSource,
			pricingClient,
		); err != nil {

		return err
	}

	billingClient :=
		awsprovider.NewCostExplorerBillingClient(
			clients.CostExplorer,
		)

	if err :=
		awsprovider.RegisterBillingProvider(
			registry,
			billingClient,
		); err != nil {

		return err
	}

	return awsprovider.RegisterCapacityProvider(
		registry,
		clients.EC2,
	)
}
