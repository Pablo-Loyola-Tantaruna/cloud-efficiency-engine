package providers

import (
	"context"
	"testing"

	"cloud-efficiency-engine/internal/analysis/capacity"
	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/pricing"
)

type providerContractBundle struct {
	metrics metrics.Provider

	history metrics.HistoricalProvider

	pricing pricing.Provider

	billing billing.Provider

	capacity capacity.Provider
}

func validateProviderContract(
	t *testing.T,
	bundle providerContractBundle,
) {

	t.Helper()

	if bundle.metrics == nil {
		t.Fatal("metrics provider must not be nil")
	}

	if bundle.history == nil {
		t.Fatal("historical provider must not be nil")
	}

	if bundle.pricing == nil {
		t.Fatal("pricing provider must not be nil")
	}

	if bundle.billing == nil {
		t.Fatal("billing provider must not be nil")
	}

	if bundle.capacity == nil {
		t.Fatal("capacity provider must not be nil")
	}
}

func resolveProviderContract(
	t *testing.T,
	registry *Registry,
	provider domain.CloudProvider,
) providerContractBundle {

	t.Helper()

	bundle, err :=
		registry.Resolve(
			context.Background(),
			domain.AnalysisContext{
				Provider: provider,

				Environment: "production",
			},
		)

	if err != nil {

		t.Fatalf(
			"resolve %q: %v",
			provider,
			err,
		)
	}

	return providerContractBundle{
		metrics: bundle.MetricsProvider,

		history: bundle.HistoricalProvider,

		pricing: bundle.PricingProvider,

		billing: bundle.BillingProvider,

		capacity: bundle.CapacityProvider,
	}
}
