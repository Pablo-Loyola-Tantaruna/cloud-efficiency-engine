package providers_test

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
	providers "cloud-efficiency-engine/internal/providers"
)

type contractMetricsProvider struct{}

func (*contractMetricsProvider) GetWorkloads(
	ctx context.Context,
	namespace string,
) ([]domain.WorkloadMetrics, error) {
	return []domain.WorkloadMetrics{{
		Namespace: namespace,
		Name:      "workload",
	}}, nil
}

func (*contractMetricsProvider) GetWorkloadHistory(
	ctx context.Context,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {
	return nil, nil
}

type contractPricingProvider struct{}

func (*contractPricingProvider) GetPricing(
	ctx context.Context,
) (pricing.ResourcePricing, error) {
	return pricing.ResourcePricing{}, nil
}

type contractBillingProvider struct {
	provider domain.CloudProvider
}

func (p *contractBillingProvider) GetCost(
	ctx context.Context,
	query billing.CostQuery,
) (billing.CostReport, error) {
	return billing.CostReport{
		Provider: p.provider,
		Start:    query.Start,
		End:      query.End,
		Currency: "USD",
	}, nil
}

type contractCapacityProvider struct{}

func (*contractCapacityProvider) GetCapacity(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (cost.ClusterCapacity, error) {
	return cost.ClusterCapacity{
		CPUCapacityMillicores: 1000,
		MemoryCapacityBytes:   1024 * 1024 * 1024,
	}, nil
}

func TestRegistry_ShouldSupportAllCloudProviders(
	t *testing.T,
) {
	providersToTest := []struct {
		name           string
		provider       domain.CloudProvider
		requireBilling bool
	}{
		{
			name:           "aws",
			provider:       domain.CloudProviderAWS,
			requireBilling: true,
		},
		{
			name:           "azure",
			provider:       domain.CloudProviderAzure,
			requireBilling: true,
		},
		{
			name:           "gcp",
			provider:       domain.CloudProviderGCP,
			requireBilling: true,
		},
		{
			name:           "kubernetes",
			provider:       domain.CloudProviderKubernetes,
			requireBilling: false,
		},
	}

	for _, tc := range providersToTest {
		t.Run(tc.name, func(t *testing.T) {
			registry := providers.NewRegistry()

			registry.RegisterMetricsProvider(
				tc.provider,
				func(analysisContext domain.AnalysisContext) (metrics.Provider, metrics.HistoricalProvider, error) {
					provider := &contractMetricsProvider{}
					return provider, provider, nil
				},
			)

			registry.RegisterPricingProvider(
				tc.provider,
				func(analysisContext domain.AnalysisContext) (pricing.Provider, error) {
					return &contractPricingProvider{}, nil
				},
			)

			if tc.requireBilling {
				registry.RegisterBillingProvider(
					tc.provider,
					func(analysisContext domain.AnalysisContext) (billing.Provider, error) {
						return &contractBillingProvider{
							provider: tc.provider,
						}, nil
					},
				)
			}

			registry.RegisterCapacityProvider(
				tc.provider,
				func(analysisContext domain.AnalysisContext) (capacity.Provider, error) {
					return &contractCapacityProvider{}, nil
				},
			)

			bundle, err := registry.Resolve(
				context.Background(),
				domain.AnalysisContext{
					Provider:    tc.provider,
					Environment: "production",
				},
			)

			if err != nil {
				t.Fatalf("expected resolve to succeed: %v", err)
			}

			if bundle.MetricsProvider == nil {
				t.Fatal("expected metrics provider")
			}
			if bundle.HistoricalProvider == nil {
				t.Fatal("expected historical provider")
			}
			if bundle.PricingProvider == nil {
				t.Fatal("expected pricing provider")
			}
			if bundle.CapacityProvider == nil {
				t.Fatal("expected capacity provider")
			}

			if tc.requireBilling && bundle.BillingProvider == nil {
				t.Fatal("expected billing provider")
			}

			if !tc.requireBilling && bundle.BillingProvider != nil {
				t.Fatal("expected billing provider to remain optional")
			}
		})
	}
}
