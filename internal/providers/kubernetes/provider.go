package kubernetes

import (
	"cloud-efficiency-engine/internal/metrics/providers"
	providerregistry "cloud-efficiency-engine/internal/providers"
)

func NewProvider(
	prometheusURL string,
	cpuPerCoreHour float64,
	memoryPerGBHour float64,
) *providerregistry.GenericProvider {

	return providerregistry.NewGenericProvider(
		NewMetricsSource(
			prometheusURL,
		),
		NewPricingSource(
			cpuPerCoreHour,
			memoryPerGBHour,
		),
	)
}

func NewCapacityProviderFromPrometheus(
	prometheusURL string,
) *CapacityProvider {

	prometheusClient :=
		providers.NewPrometheusProvider(
			prometheusURL,
			nil,
		)

	source :=
		NewCapacitySource(
			prometheusClient,
		)

	return NewCapacityProvider(
		source,
	)
}
