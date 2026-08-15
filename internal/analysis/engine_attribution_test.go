package analysis

import (
	"context"
	"math"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/analysis/capacity"
	"cloud-efficiency-engine/internal/analysis/optimizer"
	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/pricing"
)

type attributionMetricsProviderMock struct{}

func (m *attributionMetricsProviderMock) GetWorkloads(
	ctx context.Context,
	namespace string,
) ([]domain.WorkloadMetrics, error) {
	return []domain.WorkloadMetrics{{
		Namespace:            namespace,
		Name:                 "payments-api",
		Type:                 domain.WorkloadDeployment,
		CPURequestMillicores: 1000,
		MemoryRequestBytes:   4 * 1024 * 1024 * 1024,
	}}, nil
}

func (m *attributionMetricsProviderMock) GetWorkloadHistory(
	ctx context.Context,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {
	return nil, nil
}

type attributionPricingProviderMock struct{}

func (p *attributionPricingProviderMock) GetPricing(
	ctx context.Context,
) (pricing.ResourcePricing, error) {
	return pricing.ResourcePricing{
		CPUPerCoreHour:  0.04,
		MemoryPerGBHour: 0.005,
	}, nil
}

type attributionBillingProviderMock struct{}

func (p *attributionBillingProviderMock) GetCost(
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

type attributionCapacityProviderMock struct{}

func (p *attributionCapacityProviderMock) GetCapacity(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (cost.ClusterCapacity, error) {
	return cost.ClusterCapacity{
		CPUCapacityMillicores: 4000,
		MemoryCapacityBytes:   16 * 1024 * 1024 * 1024,
	}, nil
}

var _ metrics.Provider = (*attributionMetricsProviderMock)(nil)
var _ metrics.HistoricalProvider = (*attributionMetricsProviderMock)(nil)
var _ pricing.Provider = (*attributionPricingProviderMock)(nil)
var _ billing.Provider = (*attributionBillingProviderMock)(nil)
var _ capacity.Provider = (*attributionCapacityProviderMock)(nil)

func TestEngine_ShouldBuildCostAttribution(
	t *testing.T,
) {

	engine :=
		NewEngine(
			&attributionMetricsProviderMock{},
			nil,
			&attributionPricingProviderMock{},
			nil,
			optimizer.DefaultOptimizationPolicy(),
			nil,
			nil,
		)

	engine.costAttributor =
		cost.NewCostAttributor()

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

	end := start.Add(
		24 * time.Hour,
	)

	report := &AnalysisReport{
		GeneratedAt: start,
		Context: domain.AnalysisContext{
			Provider: domain.CloudProviderAWS,
		},
		Billing: &billing.CostReport{
			Provider: domain.CloudProviderAWS,
			Start:    start,
			End:      end,
			Currency: "USD",
			TotalUSD: 400,
		},
	}

	workloads, err :=
		engine.getWorkloads(
			context.Background(),
			report.Context,
			AnalysisOptions{Namespace: "payments"},
			&attributionMetricsProviderMock{},
		)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	attribution, err :=
		engine.calculateAttribution(
			context.Background(),
			report.Context,
			workloads,
			report.Billing,
			&attributionCapacityProviderMock{},
		)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedMonthlyCost := 400.0 * 730.0 / 24.0

	if math.Abs(
		attribution.Cluster.MonthlyCostUSD-
			expectedMonthlyCost,
	) > 0.000001 {
		t.Fatalf(
			"expected monthly cost %f, got %f",
			expectedMonthlyCost,
			attribution.Cluster.MonthlyCostUSD,
		)
	}

	if len(attribution.Workloads) != 1 {
		t.Fatalf(
			"expected 1 workload allocation, got %d",
			len(attribution.Workloads),
		)
	}

	allocation := attribution.Workloads[0]

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

	expectedAllocatedCost := expectedMonthlyCost * 0.25

	if math.Abs(
		allocation.AllocatedCostUSD-
			expectedAllocatedCost,
	) > 0.01 {
		t.Fatalf(
			"expected allocated cost %f, got %f",
			expectedAllocatedCost,
			allocation.AllocatedCostUSD,
		)
	}

	expectedUnallocatedCost :=
		expectedMonthlyCost - expectedAllocatedCost

	if math.Abs(
		attribution.UnallocatedCostUSD-
			expectedUnallocatedCost,
	) > 0.01 {
		t.Fatalf(
			"expected unallocated cost %f, got %f",
			expectedUnallocatedCost,
			attribution.UnallocatedCostUSD,
		)
	}
}
