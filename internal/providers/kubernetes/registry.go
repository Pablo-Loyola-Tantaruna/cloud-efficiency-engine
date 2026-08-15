package kubernetes

import (
	"fmt"

	"cloud-efficiency-engine/internal/analysis/capacity"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/pricing"
	providerregistry "cloud-efficiency-engine/internal/providers"
)

func Register(
	registry *providerregistry.Registry,
	prometheusURL string,
	cpuPerCoreHour float64,
	memoryPerGBHour float64,
	capacityProvider capacity.Provider,
) error {

	if registry == nil {

		return fmt.Errorf(
			"provider registry must not be nil",
		)
	}

	if capacityProvider == nil {

		return fmt.Errorf(
			"Kubernetes capacity provider must not be nil",
		)
	}

	provider :=
		NewProvider(
			prometheusURL,
			cpuPerCoreHour,
			memoryPerGBHour,
		)

	registry.RegisterMetricsProvider(
		domain.CloudProviderKubernetes,
		func(
			analysisContext domain.AnalysisContext,
		) (
			metrics.Provider,
			metrics.HistoricalProvider,
			error,
		) {

			return provider,
				provider,
				nil
		},
	)

	registry.RegisterPricingProvider(
		domain.CloudProviderKubernetes,
		func(
			analysisContext domain.AnalysisContext,
		) (
			pricing.Provider,
			error,
		) {

			return provider,
				nil
		},
	)

	registry.RegisterCapacityProvider(
		domain.CloudProviderKubernetes,
		func(
			analysisContext domain.AnalysisContext,
		) (
			capacity.Provider,
			error,
		) {

			return capacityProvider,
				nil
		},
	)

	return nil
}
