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
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/metrics/providers"
	"cloud-efficiency-engine/internal/observability"
	"cloud-efficiency-engine/internal/pricing"
)

const (
	defaultLookbackHours = 7 * 24
	defaultHistoryStep   = 5 * time.Minute
	defaultHoursPerMonth = 730.0

	defaultAnalysisInterval = 15 * time.Minute
)

func main() {

	logger :=
		api.NewLogger()

	optimizationRules :=
		[]rules.Rule{
			rules.NewCPUOverprovisioningRule(),
			rules.NewMemoryOverprovisioningRule(),
		}

	pricingProvider :=
		pricing.NewStaticProvider(
			pricing.ResourcePricing{
				CPUPerCoreHour:  0.04,
				MemoryPerGBHour: 0.005,
			},
		)

	calculator :=
		cost.NewCalculator(
			defaultHoursPerMonth,
		)

	recommendationResolver :=
		resolver.NewResolver()

	provider,
		historicalProvider :=
		createMetricsProviders()

	engine :=
		analysis.NewEngine(
			provider,
			historicalProvider,
			pricingProvider,
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

	namespace :=
		os.Getenv(
			"ANALYSIS_NAMESPACE",
		)

	if namespace == "" {

		namespace =
			"cloud-efficiency-engine"
	}

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
				Namespace: namespace,

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
		time.ParseDuration(value)

	if err != nil {

		return fallback
	}

	if parsed <= 0 {

		return fallback
	}

	return parsed
}

func createMetricsProviders() (
	metrics.Provider,
	metrics.HistoricalProvider,
) {

	providerType :=
		os.Getenv(
			"METRICS_PROVIDER",
		)

	if providerType == "prometheus" {

		prometheusURL :=
			os.Getenv(
				"PROMETHEUS_URL",
			)

		if prometheusURL == "" {

			prometheusURL =
				"http://localhost:9090"
		}

		provider :=
			providers.NewPrometheusProvider(
				prometheusURL,
				nil,
			)

		return provider, provider
	}

	return providers.NewMockProvider(
			mockWorkloads(),
		), providers.NewMockHistoricalProvider(
			mockHistoricalWorkloads(),
		)
}

func mockWorkloads() []domain.WorkloadMetrics {

	return []domain.WorkloadMetrics{
		{
			Namespace: "payments",
			Name:      "payments-api",

			Type: domain.WorkloadDeployment,

			Replicas: 3,

			CPURequestMillicores: 1000,
			CPUUsageMillicores:   180,

			MemoryRequestBytes: 2 *
				1024 *
				1024 *
				1024,

			MemoryUsageBytes: 640 *
				1024 *
				1024,
		},
		{
			Namespace: "orders",
			Name:      "orders-api",

			Type: domain.WorkloadDeployment,

			Replicas: 2,

			CPURequestMillicores: 500,
			CPUUsageMillicores:   350,

			MemoryRequestBytes: 1024 *
				1024 *
				1024,

			MemoryUsageBytes: 805 *
				1024 *
				1024,
		},
	}
}

func mockHistoricalWorkloads() []domain.WorkloadHistory {

	now :=
		time.Now().UTC()

	samples :=
		int(
			(defaultLookbackHours *
				time.Hour) /
				defaultHistoryStep,
		)

	return []domain.WorkloadHistory{
		{
			Namespace: "payments",
			Name:      "payments-api",

			CPUUsageMillicores: generateCPUHistory(
				now,
				180,
				samples,
			),

			MemoryUsageBytes: generateMemoryHistory(
				now,
				640*
					1024*
					1024,
				samples,
			),
		},
		{
			Namespace: "orders",
			Name:      "orders-api",

			CPUUsageMillicores: generateCPUHistory(
				now,
				350,
				samples,
			),

			MemoryUsageBytes: generateMemoryHistory(
				now,
				805*
					1024*
					1024,
				samples,
			),
		},
	}
}

func generateCPUHistory(
	now time.Time,
	base float64,
	samples int,
) []domain.MetricSample {

	result :=
		make(
			[]domain.MetricSample,
			0,
			samples,
		)

	values :=
		[]float64{
			0.80,
			0.90,
			1.00,
			1.10,
			1.20,
		}

	for i := 0; i < samples; i++ {

		multiplier :=
			values[i%len(values)]

		result =
			append(
				result,
				domain.MetricSample{
					Timestamp: now.Add(
						-time.Duration(
							samples-i,
						) *
							defaultHistoryStep,
					),

					Value: base *
						multiplier,
				},
			)
	}

	return result
}

func generateMemoryHistory(
	now time.Time,
	base float64,
	samples int,
) []domain.MetricSample {

	result :=
		make(
			[]domain.MetricSample,
			0,
			samples,
		)

	values :=
		[]float64{
			0.80,
			0.90,
			1.00,
			1.05,
			1.10,
		}

	for i := 0; i < samples; i++ {

		multiplier :=
			values[i%len(values)]

		result =
			append(
				result,
				domain.MetricSample{
					Timestamp: now.Add(
						-time.Duration(
							samples-i,
						) *
							defaultHistoryStep,
					),

					Value: base *
						multiplier,
				},
			)
	}

	return result
}
