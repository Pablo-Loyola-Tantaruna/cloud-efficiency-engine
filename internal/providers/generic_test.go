package providers

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/pricing"
)

type metricsSourceMock struct {
	workloads []domain.WorkloadMetrics
	history   []domain.WorkloadHistory

	contextReceived domain.AnalysisContext
}

func (
	m *metricsSourceMock,
) GetWorkloads(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
) ([]domain.WorkloadMetrics, error) {

	m.contextReceived =
		analysisContext

	return m.workloads, nil
}

func (
	m *metricsSourceMock,
) GetWorkloadHistory(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {

	m.contextReceived =
		analysisContext

	return m.history, nil
}

type pricingSourceMock struct {
	pricing pricing.ResourcePricing

	contextReceived domain.AnalysisContext
}

func (
	m *pricingSourceMock,
) GetPricing(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (pricing.ResourcePricing, error) {

	m.contextReceived =
		analysisContext

	return m.pricing, nil
}

func TestGenericProvider_ShouldDelegateMetrics(
	t *testing.T,
) {

	source :=
		&metricsSourceMock{
			workloads: []domain.WorkloadMetrics{
				{
					Namespace: "payments",

					Name: "payments-api",

					Replicas: 3,
				},
			},
		}

	provider :=
		NewGenericProvider(
			source,
			nil,
		)

	analysisContext :=
		domain.AnalysisContext{
			Provider: domain.CloudProviderAWS,

			Environment: "production",

			AccountID: "123456789",

			Region: "us-east-1",
		}

	result, err :=
		provider.GetWorkloadsWithContext(
			context.Background(),
			analysisContext,
			"payments",
		)

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if len(result) != 1 {

		t.Fatalf(
			"expected 1 workload, got %d",
			len(result),
		)
	}

	if result[0].Name !=
		"payments-api" {

		t.Fatalf(
			"expected payments-api, got %s",
			result[0].Name,
		)
	}

	if source.
		contextReceived.
		Provider !=
		domain.CloudProviderAWS {

		t.Fatalf(
			"expected AWS provider, got %s",
			source.contextReceived.Provider,
		)
	}

	if source.
		contextReceived.
		Region !=
		"us-east-1" {

		t.Fatalf(
			"expected us-east-1, got %s",
			source.contextReceived.Region,
		)
	}
}

func TestGenericProvider_ShouldDelegateHistory(
	t *testing.T,
) {

	source :=
		&metricsSourceMock{
			history: []domain.WorkloadHistory{
				{
					Namespace: "payments",

					Name: "payments-api",
				},
			},
		}

	provider :=
		NewGenericProvider(
			source,
			nil,
		)

	result, err :=
		provider.GetWorkloadHistoryWithContext(
			context.Background(),
			domain.AnalysisContext{
				Provider: domain.CloudProviderGCP,
			},
			"payments",
			time.Now().Add(-1*time.Hour),
			time.Now(),
			5*time.Minute,
		)

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if len(result) != 1 {

		t.Fatalf(
			"expected 1 history, got %d",
			len(result),
		)
	}

	if result[0].Name !=
		"payments-api" {

		t.Fatalf(
			"expected payments-api, got %s",
			result[0].Name,
		)
	}
}

func TestGenericProvider_ShouldDelegatePricing(
	t *testing.T,
) {

	source :=
		&pricingSourceMock{
			pricing: pricing.ResourcePricing{
				CPUPerCoreHour: 0.05,

				MemoryPerGBHour: 0.006,
			},
		}

	provider :=
		NewGenericProvider(
			nil,
			source,
		)

	result, err :=
		provider.GetPricingWithContext(
			context.Background(),
			domain.AnalysisContext{
				Provider: domain.CloudProviderAzure,
			},
		)

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if result.CPUPerCoreHour !=
		0.05 {

		t.Fatalf(
			"expected CPU price 0.05, got %.4f",
			result.CPUPerCoreHour,
		)
	}

	if result.MemoryPerGBHour !=
		0.006 {

		t.Fatalf(
			"expected memory price 0.006, got %.4f",
			result.MemoryPerGBHour,
		)
	}

	if source.
		contextReceived.
		Provider !=
		domain.CloudProviderAzure {

		t.Fatalf(
			"expected Azure provider, got %s",
			source.contextReceived.Provider,
		)
	}
}

func TestGenericProvider_ShouldRejectMissingMetricsSource(
	t *testing.T,
) {

	provider :=
		NewGenericProvider(
			nil,
			nil,
		)

	_, err :=
		provider.GetWorkloadsWithContext(
			context.Background(),
			domain.AnalysisContext{
				Provider: domain.CloudProviderAWS,
			},
			"payments",
		)

	if err == nil {

		t.Fatal(
			"expected metrics source error",
		)
	}
}

func TestGenericProvider_ShouldRejectMissingPricingSource(
	t *testing.T,
) {

	provider :=
		NewGenericProvider(
			nil,
			nil,
		)

	_, err :=
		provider.GetPricingWithContext(
			context.Background(),
			domain.AnalysisContext{
				Provider: domain.CloudProviderGCP,
			},
		)

	if err == nil {

		t.Fatal(
			"expected pricing source error",
		)
	}
}
