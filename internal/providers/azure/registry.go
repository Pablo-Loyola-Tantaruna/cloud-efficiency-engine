package azure

import (
	"fmt"

	"cloud-efficiency-engine/internal/analysis/capacity"
	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/pricing"
	providerregistry "cloud-efficiency-engine/internal/providers"
)

func Register(
	registry *providerregistry.Registry,
	metricsClient MetricsClient,
	pricingClient PricingClient,
	billingClient BillingClient,
	capacityClient CapacityClient,
) error {
	if registry == nil {
		return fmt.Errorf("provider registry must not be nil")
	}
	if metricsClient == nil {
		return fmt.Errorf("Azure metrics client must not be nil")
	}
	if pricingClient == nil {
		return fmt.Errorf("Azure pricing client must not be nil")
	}
	if billingClient == nil {
		return fmt.Errorf("Azure billing client must not be nil")
	}
	if capacityClient == nil {
		return fmt.Errorf("Azure capacity client must not be nil")
	}

	registry.RegisterMetricsProvider(
		domain.CloudProviderAzure,
		func(analysisContext domain.AnalysisContext) (
			metrics.Provider,
			metrics.HistoricalProvider,
			error,
		) {
			provider, err := NewProvider(
				metricsClient,
				pricingClient,
				billingClient,
				capacityClient,
			)
			if err != nil {
				return nil, nil, err
			}
			return provider, provider, nil
		},
	)

	registry.RegisterPricingProvider(
		domain.CloudProviderAzure,
		func(analysisContext domain.AnalysisContext) (pricing.Provider, error) {
			provider, err := NewProvider(
				metricsClient,
				pricingClient,
				billingClient,
				capacityClient,
			)
			if err != nil {
				return nil, err
			}
			return provider, nil
		},
	)

	registry.RegisterBillingProvider(
		domain.CloudProviderAzure,
		func(analysisContext domain.AnalysisContext) (billing.Provider, error) {
			provider, err := NewProvider(
				metricsClient,
				pricingClient,
				billingClient,
				capacityClient,
			)
			if err != nil {
				return nil, err
			}
			return provider, nil
		},
	)

	registry.RegisterCapacityProvider(
		domain.CloudProviderAzure,
		func(analysisContext domain.AnalysisContext) (capacity.Provider, error) {
			provider, err := NewProvider(
				metricsClient,
				pricingClient,
				billingClient,
				capacityClient,
			)
			if err != nil {
				return nil, err
			}
			return provider, nil
		},
	)

	return nil
}
