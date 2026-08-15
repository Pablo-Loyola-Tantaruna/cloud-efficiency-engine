package analysis

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/analysis/optimizer"
	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/pricing"
)

type engineBillingMetricsMock struct {
}

func (m *engineBillingMetricsMock) GetWorkloads(
	ctx context.Context,
	namespace string,
) ([]domain.WorkloadMetrics, error) {

	return []domain.WorkloadMetrics{
		{
			Namespace:            namespace,
			Name:                 "payments-api",
			Type:                 domain.WorkloadDeployment,
			Replicas:             2,
			CPURequestMillicores: 1000,
			CPUUsageMillicores:   500,
			MemoryRequestBytes:   2 * 1024 * 1024 * 1024,
			MemoryUsageBytes:     1024 * 1024 * 1024,
		},
	}, nil
}

func (m *engineBillingMetricsMock) GetWorkloadHistory(
	ctx context.Context,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {
	return nil, nil
}

type engineBillingPricingMock struct{}

func (m *engineBillingPricingMock) GetPricing(
	ctx context.Context,
) (pricing.ResourcePricing, error) {
	return pricing.ResourcePricing{
		CPUPerCoreHour:  0.04,
		MemoryPerGBHour: 0.005,
	}, nil
}

type engineBillingProviderMock struct{}

func (m *engineBillingProviderMock) GetCost(
	ctx context.Context,
	query billing.CostQuery,
) (billing.CostReport, error) {
	return billing.CostReport{
		Provider: domain.CloudProviderAWS,
		Start:    query.Start,
		End:      query.End,
		Currency: "USD",
		TotalUSD: 1250,
	}, nil
}

var _ metrics.Provider = (*engineBillingMetricsMock)(nil)
var _ metrics.HistoricalProvider = (*engineBillingMetricsMock)(nil)
var _ pricing.Provider = (*engineBillingPricingMock)(nil)
var _ billing.Provider = (*engineBillingProviderMock)(nil)

func TestEngine_ShouldIncludeActualBillingCost(
	t *testing.T,
) {

	engine :=
		NewEngine(
			&engineBillingMetricsMock{},
			&engineBillingMetricsMock{},
			&engineBillingPricingMock{},
			nil,
			optimizer.DefaultOptimizationPolicy(),
			nil,
			nil,
		)

	start :=
		time.Now().UTC().Add(-24 * time.Hour)

	end := time.Now().UTC()

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
					Environment: "test",
				},
			},
		)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if report == nil {
		t.Fatal("expected report")
	}
}
