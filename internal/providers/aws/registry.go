package aws

import (
	"fmt"

	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/pricing"
	providerregistry "cloud-efficiency-engine/internal/providers"
)

func Register(
	registry *providerregistry.Registry,
	metricsClient MetricsClient,
	pricingClient PricingClient,
) error {

	if registry == nil {
		return fmt.Errorf(
			"provider registry must not be nil",
		)
	}

	if metricsClient == nil {
		return fmt.Errorf(
			"AWS metrics client must not be nil",
		)
	}

	if pricingClient == nil {
		return fmt.Errorf(
			"AWS pricing client must not be nil",
		)
	}

	registry.RegisterMetricsProvider(
		domain.CloudProviderAWS,
		func(
			analysisContext domain.AnalysisContext,
		) (
			metrics.Provider,
			metrics.HistoricalProvider,
			error,
		) {

			provider, err :=
				NewProvider(
					metricsClient,
					pricingClient,
				)

			if err != nil {
				return nil, nil, err
			}

			return provider, provider, nil
		},
	)

	registry.RegisterPricingProvider(
		domain.CloudProviderAWS,
		func(
			analysisContext domain.AnalysisContext,
		) (
			pricing.Provider,
			error,
		) {

			provider, err :=
				NewProvider(
					metricsClient,
					pricingClient,
				)

			if err != nil {
				return nil, err
			}

			return provider, nil
		},
	)

	return nil
}

func RegisterWithSources(
	registry *providerregistry.Registry,
	metricsSource providerregistry.MetricsSource,
	pricingClient PricingClient,
) error {

	if registry == nil {
		return fmt.Errorf(
			"provider registry must not be nil",
		)
	}

	if metricsSource == nil {
		return fmt.Errorf(
			"AWS metrics source must not be nil",
		)
	}

	if pricingClient == nil {
		return fmt.Errorf(
			"AWS pricing client must not be nil",
		)
	}

	registry.RegisterMetricsProvider(
		domain.CloudProviderAWS,
		func(
			analysisContext domain.AnalysisContext,
		) (
			metrics.Provider,
			metrics.HistoricalProvider,
			error,
		) {

			provider, err :=
				NewProviderWithSources(
					metricsSource,
					pricingClient,
				)

			if err != nil {
				return nil, nil, err
			}

			return provider, provider, nil
		},
	)

	registry.RegisterPricingProvider(
		domain.CloudProviderAWS,
		func(
			analysisContext domain.AnalysisContext,
		) (
			pricing.Provider,
			error,
		) {

			provider, err :=
				NewProviderWithSources(
					metricsSource,
					pricingClient,
				)

			if err != nil {
				return nil, err
			}

			return provider, nil
		},
	)

	return nil
}
