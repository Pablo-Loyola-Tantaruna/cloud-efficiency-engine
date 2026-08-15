package gcp

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/pricing"
	"cloud-efficiency-engine/internal/providers"
)

type metricsClientMock struct{}

func (m *metricsClientMock) GetWorkloads(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
) ([]domain.WorkloadMetrics, error) {
	return []domain.WorkloadMetrics{
		{Namespace: namespace, Name: "api"},
	}, nil
}

func (m *metricsClientMock) GetWorkloadHistory(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {
	return []domain.WorkloadHistory{}, nil
}

type pricingClientMock struct{}

func (m *pricingClientMock) GetPricing(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (pricing.ResourcePricing, error) {
	return pricing.ResourcePricing{
		CPUPerCoreHour:  0.04,
		MemoryPerGBHour: 0.005,
	}, nil
}

type billingClientMock struct{}

func (m *billingClientMock) GetCost(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	query billing.CostQuery,
) (billing.CostReport, error) {
	return billing.CostReport{
		Provider: domain.CloudProviderGCP,
		Start:    query.Start,
		End:      query.End,
		Currency: "USD",
		TotalUSD: 100,
	}, nil
}

type capacityClientMock struct{}

func (m *capacityClientMock) GetCapacity(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (int64, int64, error) {
	return 4000, 16 * 1024 * 1024 * 1024, nil
}

func TestRegister_ShouldResolveAllGCPCapabilities(
	t *testing.T,
) {
	registry := providers.NewRegistry()

	err := Register(
		registry,
		&metricsClientMock{},
		&pricingClientMock{},
		&billingClientMock{},
		&capacityClientMock{},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	bundle, err := registry.Resolve(
		context.Background(),
		domain.AnalysisContext{
			Provider:    domain.CloudProviderGCP,
			Environment: "production",
			Region:      "us-central1",
		},
	)
	if err != nil {
		t.Fatalf("expected resolve to succeed, got %v", err)
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
	if bundle.BillingProvider == nil {
		t.Fatal("expected billing provider")
	}
	if bundle.CapacityProvider == nil {
		t.Fatal("expected capacity provider")
	}
}
