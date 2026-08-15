package aws

import (
	"fmt"

	providerregistry "cloud-efficiency-engine/internal/providers"
)

func NewProvider(
	metricsClient MetricsClient,
	pricingClient PricingClient,
) (*providerregistry.GenericProvider, error) {

	if metricsClient == nil {
		return nil,
			fmt.Errorf(
				"AWS metrics client must not be nil",
			)
	}

	if pricingClient == nil {
		return nil,
			fmt.Errorf(
				"AWS pricing client must not be nil",
			)
	}

	return providerregistry.NewGenericProvider(
		NewMetricsSource(
			metricsClient,
		),
		NewPricingSource(
			pricingClient,
		),
	), nil
}

func NewProviderWithSources(
	metricsSource providerregistry.MetricsSource,
	pricingClient PricingClient,
) (*providerregistry.GenericProvider, error) {

	if metricsSource == nil {
		return nil,
			fmt.Errorf(
				"AWS metrics source must not be nil",
			)
	}

	if pricingClient == nil {
		return nil,
			fmt.Errorf(
				"AWS pricing client must not be nil",
			)
	}

	return providerregistry.NewGenericProvider(
		metricsSource,
		NewPricingSource(
			pricingClient,
		),
	), nil
}
