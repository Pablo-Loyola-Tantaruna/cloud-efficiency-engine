package analysis

import (
	"context"
	"math"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/analysis/capacity"
	"cloud-efficiency-engine/internal/analysis/optimizer"
	"cloud-efficiency-engine/internal/analysis/resolver"
	"cloud-efficiency-engine/internal/analysis/rules"
	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/pricing"
	providerregistry "cloud-efficiency-engine/internal/providers"
)

type finopsMetricsProvider struct{}

func (p *finopsMetricsProvider) GetWorkloads(
	ctx context.Context,
	namespace string,
) ([]domain.WorkloadMetrics, error) {

	return []domain.WorkloadMetrics{
		{
			Namespace:            namespace,
			Name:                 "payments-api",
			Type:                 domain.WorkloadDeployment,
			CPURequestMillicores: 1000,
			MemoryRequestBytes:   4 * 1024 * 1024 * 1024,
		},
	}, nil
}

func (p *finopsMetricsProvider) GetWorkloadHistory(
	ctx context.Context,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {

	return []domain.WorkloadHistory{}, nil
}

type finopsPricingProvider struct{}

func (p *finopsPricingProvider) GetPricing(
	ctx context.Context,
) (pricing.ResourcePricing, error) {

	return pricing.ResourcePricing{
		CPUPerCoreHour:  0.04,
		MemoryPerGBHour: 0.005,
	}, nil
}

type finopsBillingProvider struct{}

func (p *finopsBillingProvider) GetCost(
	ctx context.Context,
	query billing.CostQuery,
) (billing.CostReport, error) {

	return billing.CostReport{
		Provider: domain.CloudProviderAWS,
		Start:    query.Start,
		End:      query.End,
		Currency: "USD",
		TotalUSD: 400,
	}, nil
}

type finopsCapacityProvider struct{}

func (p *finopsCapacityProvider) GetCapacity(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (cost.ClusterCapacity, error) {

	return cost.ClusterCapacity{
		CPUCapacityMillicores: 4000,
		MemoryCapacityBytes:   16 * 1024 * 1024 * 1024,
	}, nil
}

var _ metrics.Provider = (*finopsMetricsProvider)(nil)
var _ metrics.HistoricalProvider = (*finopsMetricsProvider)(nil)
var _ pricing.Provider = (*finopsPricingProvider)(nil)
var _ billing.Provider = (*finopsBillingProvider)(nil)
var _ capacity.Provider = (*finopsCapacityProvider)(nil)

func TestEngineWithRegistry_ShouldBuildFinOpsAttribution(
	t *testing.T,
) {

	registry := providerregistry.NewRegistry()

	registry.RegisterMetricsProvider(
		domain.CloudProviderAWS,
		func(
			analysisContext domain.AnalysisContext,
		) (
			metrics.Provider,
			metrics.HistoricalProvider,
			error,
		) {
			provider := &finopsMetricsProvider{}
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
			return &finopsPricingProvider{}, nil
		},
	)

	registry.RegisterBillingProvider(
		domain.CloudProviderAWS,
		func(
			analysisContext domain.AnalysisContext,
		) (
			billing.Provider,
			error,
		) {
			return &finopsBillingProvider{}, nil
		},
	)

	registry.RegisterCapacityProvider(
		domain.CloudProviderAWS,
		func(
			analysisContext domain.AnalysisContext,
		) (
			capacity.Provider,
			error,
		) {
			return &finopsCapacityProvider{}, nil
		},
	)

	engine :=
		NewEngineWithRegistry(
			registry,
			[]rules.Rule{},
			optimizer.DefaultOptimizationPolicy(),
			resolver.NewResolver(),
			cost.NewCalculator(730),
		)

	start := time.Date(
		2026,
		8,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	end := start.Add(24 * time.Hour)

	report, err :=
		engine.Analyze(
			context.Background(),
			AnalysisOptions{
				Namespace: "payments",
				Start:     start,
				End:       end,
				Step:      5 * time.Minute,
				Context: domain.AnalysisContext{
					Provider:    domain.CloudProviderAWS,
					Environment: "production",
					Region:      "us-east-1",
					ClusterName: "production",
				},
			},
		)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if report == nil {
		t.Fatal("expected report")
	}

	if report.Billing == nil {
		t.Fatal("expected billing report")
	}

	if report.Billing.TotalUSD != 400 {
		t.Fatalf(
			"expected billing cost 400, got %f",
			report.Billing.TotalUSD,
		)
	}

	if report.Attribution == nil {
		t.Fatal("expected attribution report")
	}

	if len(report.Attribution.Workloads) != 1 {
		t.Fatalf(
			"expected one workload attribution, got %d",
			len(report.Attribution.Workloads),
		)
	}

	allocation := report.Attribution.Workloads[0]

	if allocation.CPUAllocationPercentage != 25 {
		t.Fatalf(
			"expected CPU allocation 25%%, got %f",
			allocation.CPUAllocationPercentage,
		)
	}

	if allocation.MemoryAllocationPercentage != 25 {
		t.Fatalf(
			"expected memory allocation 25%%, got %f",
			allocation.MemoryAllocationPercentage,
		)
	}

	expectedMonthlyCost := 400.0 * 730.0 / 24.0
	expectedAllocatedCost := expectedMonthlyCost * 0.25
	expectedUnallocatedCost := expectedMonthlyCost - expectedAllocatedCost

	if math.Abs(
		allocation.AllocatedCostUSD-expectedAllocatedCost,
	) > 0.01 {
		t.Fatalf(
			"expected allocated cost %f, got %f",
			expectedAllocatedCost,
			allocation.AllocatedCostUSD,
		)
	}

	if math.Abs(
		report.Attribution.AllocatedCostUSD-expectedAllocatedCost,
	) > 0.01 {
		t.Fatalf(
			"expected total allocated cost %f, got %f",
			expectedAllocatedCost,
			report.Attribution.AllocatedCostUSD,
		)
	}

	if math.Abs(
		report.Attribution.UnallocatedCostUSD-expectedUnallocatedCost,
	) > 0.01 {
		t.Fatalf(
			"expected unallocated cost %f, got %f",
			expectedUnallocatedCost,
			report.Attribution.UnallocatedCostUSD,
		)
	}
}
