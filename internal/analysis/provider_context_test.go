package analysis

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/analysis/optimizer"
	"cloud-efficiency-engine/internal/analysis/resolver"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/pricing"
)

type contextAwareMetricsProviderMock struct {
	workloads []domain.WorkloadMetrics

	contextReceived domain.AnalysisContext
}

func (
	p *contextAwareMetricsProviderMock,
) GetWorkloads(
	ctx context.Context,
	namespace string,
) ([]domain.WorkloadMetrics, error) {

	return p.workloads, nil
}

func (
	p *contextAwareMetricsProviderMock,
) GetWorkloadsWithContext(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
) ([]domain.WorkloadMetrics, error) {

	p.contextReceived =
		analysisContext

	return p.workloads, nil
}

type contextAwareHistoricalProviderMock struct {
	contextReceived domain.AnalysisContext
}

func (
	p *contextAwareHistoricalProviderMock,
) GetWorkloadHistory(
	ctx context.Context,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {

	return nil, nil
}

func (
	p *contextAwareHistoricalProviderMock,
) GetWorkloadHistoryWithContext(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {

	p.contextReceived =
		analysisContext

	return nil, nil
}

type contextAwarePricingProviderMock struct {
	contextReceived domain.AnalysisContext
}

func (
	p *contextAwarePricingProviderMock,
) GetPricing(
	ctx context.Context,
) (pricing.ResourcePricing, error) {

	return pricing.ResourcePricing{
		CPUPerCoreHour: 0.04,

		MemoryPerGBHour: 0.005,
	}, nil
}

func (
	p *contextAwarePricingProviderMock,
) GetPricingWithContext(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (pricing.ResourcePricing, error) {

	p.contextReceived =
		analysisContext

	return pricing.ResourcePricing{
		CPUPerCoreHour: 0.04,

		MemoryPerGBHour: 0.005,
	}, nil
}

func TestEngineAnalyze_ShouldPassAnalysisContextToProviders(
	t *testing.T,
) {

	metricsProvider :=
		&contextAwareMetricsProviderMock{
			workloads: []domain.WorkloadMetrics{},
		}

	historicalProvider :=
		&contextAwareHistoricalProviderMock{}

	pricingProvider :=
		&contextAwarePricingProviderMock{}

	engine :=
		NewEngine(
			metricsProvider,
			historicalProvider,
			pricingProvider,
			nil,
			optimizer.DefaultOptimizationPolicy(),
			resolver.NewResolver(),
			cost.NewCalculator(
				730,
			),
		)

	expectedContext :=
		domain.AnalysisContext{
			Provider: domain.CloudProviderAWS,

			Environment: "production",

			AccountID: "123456789",

			Region: "us-east-1",

			ClusterName: "prod-eks",
		}

	now :=
		time.Now().UTC()

	result, err :=
		engine.Analyze(
			context.Background(),
			AnalysisOptions{
				Namespace: "payments",

				Start: now.Add(
					-24 * time.Hour,
				),

				End: now,

				Step: 5 * time.Minute,

				Context: expectedContext,
			},
		)

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if result == nil {

		t.Fatal(
			"expected report",
		)
	}

	if metricsProvider.
		contextReceived.Provider !=
		domain.CloudProviderAWS {

		t.Fatalf(
			"expected metrics provider to receive AWS context, got %s",
			metricsProvider.contextReceived.Provider,
		)
	}

	if historicalProvider.
		contextReceived.Region !=
		"us-east-1" {

		t.Fatalf(
			"expected historical provider region us-east-1, got %s",
			historicalProvider.contextReceived.Region,
		)
	}

	if pricingProvider.
		contextReceived.AccountID !=
		"123456789" {

		t.Fatalf(
			"expected pricing provider account 123456789, got %s",
			pricingProvider.contextReceived.AccountID,
		)
	}
}

var _ metrics.Provider = (*contextAwareMetricsProviderMock)(nil)

var _ metrics.ContextAwareProvider = (*contextAwareMetricsProviderMock)(nil)

var _ pricing.Provider = (*contextAwarePricingProviderMock)(nil)

var _ pricing.ContextAwareProvider = (*contextAwarePricingProviderMock)(nil)
