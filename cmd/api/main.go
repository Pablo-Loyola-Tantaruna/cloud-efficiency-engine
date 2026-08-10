package main

import (
	"log"
	"net/http"
	"os"

	"cloud-efficiency-engine/internal/analysis"
	"cloud-efficiency-engine/internal/analysis/rules"
	"cloud-efficiency-engine/internal/api"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/metrics/providers"
)

func main() {

	optimizationRules := []rules.Rule{
		rules.NewCPUOverprovisioningRule(),
		rules.NewMemoryOverprovisioningRule(),
	}

	pricing := cost.Pricing{
		CPUPerCoreHour:  0.04,
		MemoryPerGBHour: 0.005,
		HoursPerMonth:   730,
	}

	calculator := cost.NewCalculator(pricing)

	provider := createMetricsProvider()

	analyzer := analysis.NewAnalyzer(
		provider,
		optimizationRules,
		calculator,
	)

	handler := api.NewHandler(analyzer)
	router := api.NewRouter(handler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	log.Println("Cloud Efficiency Engine listening on :8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func createMetricsProvider() metrics.Provider {

	providerType := os.Getenv("METRICS_PROVIDER")

	if providerType == "prometheus" {

		prometheusURL := os.Getenv("PROMETHEUS_URL")

		if prometheusURL == "" {
			prometheusURL = "http://localhost:9090"
		}

		return providers.NewPrometheusProvider(
			prometheusURL,
			nil,
		)
	}

	return providers.NewMockProvider(
		mockWorkloads(),
	)
}

func mockWorkloads() []domain.WorkloadMetrics {

	return []domain.WorkloadMetrics{

		{
			Namespace:            "payments",
			Name:                 "payments-api",
			Replicas:             3,
			CPURequestMillicores: 1000,
			CPUUsageMillicores:   180,
			MemoryRequestBytes:   2 * 1024 * 1024 * 1024,
			MemoryUsageBytes:     640 * 1024 * 1024,
		},

		{
			Namespace:            "orders",
			Name:                 "orders-api",
			Replicas:             2,
			CPURequestMillicores: 500,
			CPUUsageMillicores:   350,
			MemoryRequestBytes:   1024 * 1024 * 1024,
			MemoryUsageBytes:     805 * 1024 * 1024,
		},
	}
}
