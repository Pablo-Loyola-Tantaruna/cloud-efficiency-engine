package providers

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/analysis/capacity"
	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/pricing"
)

type metricsProviderMock struct{}

func (m *metricsProviderMock) GetWorkloads(
	ctx context.Context,
	namespace string,
) ([]domain.WorkloadMetrics, error) {
	return nil, nil
}

func (m *metricsProviderMock) GetWorkloadHistory(
	ctx context.Context,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {
	return nil, nil
}

type historicalProviderMock struct{}

func (m *historicalProviderMock) GetWorkloadHistory(
	ctx context.Context,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {
	return nil, nil
}

type pricingProviderMock struct{}

func (p *pricingProviderMock) GetPricing(
	ctx context.Context,
) (pricing.ResourcePricing, error) {
	return pricing.ResourcePricing{}, nil
}

func TestRegistryResolve_ShouldResolveRegisteredProviders(
	t *testing.T,
) {
	registry := NewRegistry()
	expectedMetrics := &metricsProviderMock{}
	expectedPricing := &pricingProviderMock{}

	registry.RegisterMetricsProvider(
		domain.CloudProviderAWS,
		func(analysisContext domain.AnalysisContext) (metrics.Provider, metrics.HistoricalProvider, error) {
			return expectedMetrics, expectedMetrics, nil
		},
	)

	registry.RegisterPricingProvider(
		domain.CloudProviderAWS,
		func(analysisContext domain.AnalysisContext) (pricing.Provider, error) {
			return expectedPricing, nil
		},
	)

	result, err := registry.Resolve(
		context.Background(),
		domain.AnalysisContext{
			Provider:    domain.CloudProviderAWS,
			Environment: "production",
		},
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected provider bundle")
	}
	if result.MetricsProvider != expectedMetrics {
		t.Fatal("expected registered metrics provider")
	}
	if result.PricingProvider != expectedPricing {
		t.Fatal("expected registered pricing provider")
	}
}

func TestRegistryResolve_ShouldRejectUnsupportedProvider(
	t *testing.T,
) {
	registry := NewRegistry()

	_, err := registry.Resolve(
		context.Background(),
		domain.AnalysisContext{
			Provider: domain.CloudProviderAWS,
		},
	)

	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}

func TestRegistry_ShouldResolveKubernetesWhenExplicitlyRegistered(
	t *testing.T,
) {
	registry := NewRegistry()

	metricsProvider := &metricsProviderMock{}
	pricingProvider := &pricingProviderMock{}
	capacityProvider := &capacityProviderMock{}

	registry.RegisterMetricsProvider(
		domain.CloudProviderKubernetes,
		func(analysisContext domain.AnalysisContext) (metrics.Provider, metrics.HistoricalProvider, error) {
			return metricsProvider, metricsProvider, nil
		},
	)

	registry.RegisterPricingProvider(
		domain.CloudProviderKubernetes,
		func(analysisContext domain.AnalysisContext) (pricing.Provider, error) {
			return pricingProvider, nil
		},
	)

	registry.RegisterCapacityProvider(
		domain.CloudProviderKubernetes,
		func(analysisContext domain.AnalysisContext) (capacity.Provider, error) {
			return capacityProvider, nil
		},
	)

	result, err := registry.Resolve(
		context.Background(),
		domain.AnalysisContext{
			Provider:    domain.CloudProviderKubernetes,
			Environment: "production",
		},
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected provider bundle")
	}
	if result.MetricsProvider == nil {
		t.Fatal("expected metrics provider")
	}
	if result.HistoricalProvider == nil {
		t.Fatal("expected historical provider")
	}
	if result.PricingProvider == nil {
		t.Fatal("expected pricing provider")
	}
	if result.CapacityProvider == nil {
		t.Fatal("expected capacity provider")
	}
}

type billingProviderMock struct{}

func (b *billingProviderMock) GetCost(
	ctx context.Context,
	query billing.CostQuery,
) (billing.CostReport, error) {
	return billing.CostReport{
		Provider: domain.CloudProviderAWS,
		Currency: "USD",
		TotalUSD: 100,
	}, nil
}

func TestRegistry_ShouldResolveOptionalBillingProvider(
	t *testing.T,
) {
	registry := NewRegistry()

	registry.RegisterMetricsProvider(
		domain.CloudProviderAWS,
		func(analysisContext domain.AnalysisContext) (metrics.Provider, metrics.HistoricalProvider, error) {
			provider := &metricsProviderMock{}
			return provider, provider, nil
		},
	)

	registry.RegisterPricingProvider(
		domain.CloudProviderAWS,
		func(analysisContext domain.AnalysisContext) (pricing.Provider, error) {
			return pricing.NewStaticProvider(
				pricing.ResourcePricing{
					CPUPerCoreHour:  0.04,
					MemoryPerGBHour: 0.005,
				},
			), nil
		},
	)

	billingProvider := &billingProviderMock{}

	registry.RegisterBillingProvider(
		domain.CloudProviderAWS,
		func(analysisContext domain.AnalysisContext) (billing.Provider, error) {
			return billingProvider, nil
		},
	)

	bundle, err := registry.Resolve(
		context.Background(),
		domain.AnalysisContext{Provider: domain.CloudProviderAWS},
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if bundle.BillingProvider == nil {
		t.Fatal("expected AWS billing provider")
	}
}

type capacityProviderMock struct{}

func (p *capacityProviderMock) GetCapacity(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (cost.ClusterCapacity, error) {
	return cost.ClusterCapacity{
		CPUCapacityMillicores: 4000,
		MemoryCapacityBytes:   16 * 1024 * 1024 * 1024,
		MonthlyCostUSD:        400,
	}, nil
}

func TestRegistry_ShouldResolveOptionalCapacityProvider(
	t *testing.T,
) {
	registry := NewRegistry()

	registry.RegisterMetricsProvider(
		domain.CloudProviderAWS,
		func(analysisContext domain.AnalysisContext) (metrics.Provider, metrics.HistoricalProvider, error) {
			provider := &metricsProviderMock{}
			return provider, provider, nil
		},
	)

	registry.RegisterPricingProvider(
		domain.CloudProviderAWS,
		func(analysisContext domain.AnalysisContext) (pricing.Provider, error) {
			return pricing.NewStaticProvider(
				pricing.ResourcePricing{
					CPUPerCoreHour:  0.04,
					MemoryPerGBHour: 0.005,
				},
			), nil
		},
	)

	registry.RegisterCapacityProvider(
		domain.CloudProviderAWS,
		func(analysisContext domain.AnalysisContext) (capacity.Provider, error) {
			return &capacityProviderMock{}, nil
		},
	)

	bundle, err := registry.Resolve(
		context.Background(),
		domain.AnalysisContext{Provider: domain.CloudProviderAWS},
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if bundle == nil {
		t.Fatal("expected bundle")
	}
	if bundle.CapacityProvider == nil {
		t.Fatal("expected capacity provider")
	}
}

func TestRegistry_ShouldResolveAllSupportedCloudContracts(t *testing.T) {
	cloudProviders := []domain.CloudProvider{
		domain.CloudProviderAWS,
		domain.CloudProviderAzure,
		domain.CloudProviderGCP,
	}

	for _, cloudProvider := range cloudProviders {
		t.Run(string(cloudProvider), func(t *testing.T) {
			registry := NewRegistry()
			metricsProvider := &metricsProviderMock{}
			pricingProvider := &pricingProviderMock{}
			billingProvider := &billingProviderMock{}
			capacityProvider := &capacityProviderMock{}

			registry.RegisterMetricsProvider(
				cloudProvider,
				func(analysisContext domain.AnalysisContext) (metrics.Provider, metrics.HistoricalProvider, error) {
					return metricsProvider, metricsProvider, nil
				},
			)
			registry.RegisterPricingProvider(
				cloudProvider,
				func(analysisContext domain.AnalysisContext) (pricing.Provider, error) {
					return pricingProvider, nil
				},
			)
			registry.RegisterBillingProvider(
				cloudProvider,
				func(analysisContext domain.AnalysisContext) (billing.Provider, error) {
					return billingProvider, nil
				},
			)
			registry.RegisterCapacityProvider(
				cloudProvider,
				func(analysisContext domain.AnalysisContext) (capacity.Provider, error) {
					return capacityProvider, nil
				},
			)

			bundle, err := registry.Resolve(
				context.Background(),
				domain.AnalysisContext{
					Provider:    cloudProvider,
					Environment: "production",
					Region:      "test-region",
				},
			)
			if err != nil {
				t.Fatalf("expected %s to resolve, got %v", cloudProvider, err)
			}

			validateProviderContract(t, providerContractBundle{
				metrics:  bundle.MetricsProvider,
				history:  bundle.HistoricalProvider,
				pricing:  bundle.PricingProvider,
				billing:  bundle.BillingProvider,
				capacity: bundle.CapacityProvider,
			})
		})
	}
}

var _ capacity.Provider = (*capacityProviderMock)(nil)
var _ metrics.Provider = (*metricsProviderMock)(nil)
var _ metrics.HistoricalProvider = (*metricsProviderMock)(nil)
var _ pricing.Provider = (*pricingProviderMock)(nil)
var _ billing.Provider = (*billingProviderMock)(nil)
