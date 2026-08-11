package main

import (
	"log"
	"net/http"
	"os"
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
	"cloud-efficiency-engine/internal/pricing"
)

const (
	defaultLookbackHours = 7 * 24
	defaultHistoryStep   = 5 * time.Minute
	defaultHoursPerMonth = 730.0
)

func main() {

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

	handler :=
		api.NewHandler(
			engine,
		)

	router :=
		api.NewRouter(
			handler,
		)

	server :=
		&http.Server{
			Addr:    ":8080",
			Handler: router,
		}

	log.Println(
		"Cloud Efficiency Engine listening on :8080",
	)

	if err :=
		server.ListenAndServe(); err != nil {

		log.Fatal(err)
	}
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
		),
		providers.NewMockHistoricalProvider(
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

			MemoryRequestBytes: 2 * 1024 * 1024 * 1024,

			MemoryUsageBytes: 640 * 1024 * 1024,
		},
		{
			Namespace: "orders",
			Name:      "orders-api",

			Type: domain.WorkloadDeployment,

			Replicas: 2,

			CPURequestMillicores: 500,
			CPUUsageMillicores:   350,

			MemoryRequestBytes: 1024 * 1024 * 1024,

			MemoryUsageBytes: 805 * 1024 * 1024,
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
				640*1024*1024,
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
				805*1024*1024,
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
						) * defaultHistoryStep,
					),

					Value: base * multiplier,
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
						) * defaultHistoryStep,
					),

					Value: base * multiplier,
				},
			)
	}

	return result
}
