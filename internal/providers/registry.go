package providers

import (
	"context"
	"fmt"

	"cloud-efficiency-engine/internal/analysis/capacity"
	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/pricing"
)

type MetricsProviderFactory func(
	analysisContext domain.AnalysisContext,
) (
	metrics.Provider,
	metrics.HistoricalProvider,
	error,
)

type PricingProviderFactory func(
	analysisContext domain.AnalysisContext,
) (
	pricing.Provider,
	error,
)

type BillingProviderFactory func(
	analysisContext domain.AnalysisContext,
) (
	billing.Provider,
	error,
)

type CapacityProviderFactory func(
	analysisContext domain.AnalysisContext,
) (
	capacity.Provider,
	error,
)

type Bundle struct {
	MetricsProvider metrics.Provider

	HistoricalProvider metrics.HistoricalProvider

	PricingProvider pricing.Provider

	BillingProvider billing.Provider

	CapacityProvider capacity.Provider
}

type Registry struct {
	metricsFactories map[domain.CloudProvider]MetricsProviderFactory

	pricingFactories map[domain.CloudProvider]PricingProviderFactory

	billingFactories map[domain.CloudProvider]BillingProviderFactory

	capacityFactories map[domain.CloudProvider]CapacityProviderFactory
}

func NewRegistry() *Registry {

	return &Registry{
		metricsFactories: make(
			map[domain.CloudProvider]MetricsProviderFactory,
		),

		pricingFactories: make(
			map[domain.CloudProvider]PricingProviderFactory,
		),

		billingFactories: make(
			map[domain.CloudProvider]BillingProviderFactory,
		),

		capacityFactories: make(
			map[domain.CloudProvider]CapacityProviderFactory,
		),
	}
}

func (r *Registry) RegisterMetricsProvider(
	provider domain.CloudProvider,
	factory MetricsProviderFactory,
) {

	r.metricsFactories[provider] =
		factory
}

func (r *Registry) RegisterPricingProvider(
	provider domain.CloudProvider,
	factory PricingProviderFactory,
) {

	r.pricingFactories[provider] =
		factory
}

func (r *Registry) RegisterBillingProvider(
	provider domain.CloudProvider,
	factory BillingProviderFactory,
) {

	r.billingFactories[provider] =
		factory
}

func (r *Registry) RegisterCapacityProvider(
	provider domain.CloudProvider,
	factory CapacityProviderFactory,
) {

	r.capacityFactories[provider] =
		factory
}

func (r *Registry) Resolve(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (*Bundle, error) {

	normalizedContext :=
		domain.NormalizeAnalysisContext(
			analysisContext,
		)

	metricsFactory, ok :=
		r.metricsFactories[normalizedContext.Provider]

	if !ok {

		return nil,
			fmt.Errorf(
				"metrics provider is not registered for cloud provider %q",
				normalizedContext.Provider,
			)
	}

	pricingFactory, ok :=
		r.pricingFactories[normalizedContext.Provider]

	if !ok {

		return nil,
			fmt.Errorf(
				"pricing provider is not registered for cloud provider %q",
				normalizedContext.Provider,
			)
	}

	metricsProvider,
		historicalProvider,
		err :=
		metricsFactory(
			normalizedContext,
		)

	if err != nil {

		return nil,
			fmt.Errorf(
				"create metrics providers for %q: %w",
				normalizedContext.Provider,
				err,
			)
	}

	pricingProvider,
		err :=
		pricingFactory(
			normalizedContext,
		)

	if err != nil {

		return nil,
			fmt.Errorf(
				"create pricing provider for %q: %w",
				normalizedContext.Provider,
				err,
			)
	}

	var billingProvider billing.Provider

	billingFactory,
		hasBilling :=
		r.billingFactories[normalizedContext.Provider]

	if hasBilling {

		billingProvider,
			err =
			billingFactory(
				normalizedContext,
			)

		if err != nil {

			return nil,
				fmt.Errorf(
					"create billing provider for %q: %w",
					normalizedContext.Provider,
					err,
				)
		}
	}

	var capacityProvider capacity.Provider

	capacityFactory,
		hasCapacity :=
		r.capacityFactories[normalizedContext.Provider]

	if hasCapacity {

		capacityProvider,
			err =
			capacityFactory(
				normalizedContext,
			)

		if err != nil {

			return nil,
				fmt.Errorf(
					"create capacity provider for %q: %w",
					normalizedContext.Provider,
					err,
				)
		}
	}

	return &Bundle{
		MetricsProvider: metricsProvider,

		HistoricalProvider: historicalProvider,

		PricingProvider: pricingProvider,

		BillingProvider: billingProvider,

		CapacityProvider: capacityProvider,
	}, nil
}
