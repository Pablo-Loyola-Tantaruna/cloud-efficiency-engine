package aws

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/pricing"
	providerregistry "cloud-efficiency-engine/internal/providers"
)

type awsMetricsProviderMock struct{}

func (m *awsMetricsProviderMock) GetWorkloads(
	ctx context.Context,
	namespace string,
) ([]domain.WorkloadMetrics, error) {
	return nil, nil
}

func (m *awsMetricsProviderMock) GetWorkloadHistory(
	ctx context.Context,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {
	return nil, nil
}

type awsPricingProviderMock struct{}

func (m *awsPricingProviderMock) GetPricing(
	ctx context.Context,
) (pricing.ResourcePricing, error) {
	return pricing.ResourcePricing{}, nil
}

func TestRegisterCapacityProvider_ShouldRegisterAWSProvider(
	t *testing.T,
) {

	registry :=
		providerregistry.NewRegistry()

	err :=
		RegisterCapacityProvider(
			registry,
			&capacityEC2Mock{},
		)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	registry.RegisterMetricsProvider(
		domain.CloudProviderAWS,
		func(analysisContext domain.AnalysisContext) (
			metrics.Provider,
			metrics.HistoricalProvider,
			error,
		) {
			provider := &awsMetricsProviderMock{}
			return provider, provider, nil
		},
	)

	registry.RegisterPricingProvider(
		domain.CloudProviderAWS,
		func(analysisContext domain.AnalysisContext) (pricing.Provider, error) {
			return &awsPricingProviderMock{}, nil
		},
	)

	bundle, err :=
		registry.Resolve(
			context.Background(),
			domain.AnalysisContext{
				Provider: domain.CloudProviderAWS,
			},
		)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if bundle == nil {
		t.Fatal("expected bundle")
	}

	if bundle.CapacityProvider == nil {
		t.Fatal(
			"expected AWS capacity provider",
		)
	}
}
