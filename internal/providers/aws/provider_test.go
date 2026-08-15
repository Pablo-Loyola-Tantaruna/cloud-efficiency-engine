package aws

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type metricsClientMock struct {
	contextReceived domain.AnalysisContext
}

func (
	m *metricsClientMock,
) GetWorkloads(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) ([]WorkloadResource, error) {

	m.contextReceived =
		analysisContext

	return []WorkloadResource{
		{
			Namespace: "payments",

			Name: "payments-api",

			Type: "Deployment",

			Replicas: 3,

			CPURequestMillicores: 1500,

			CPUUsageMillicores: 600,

			MemoryRequestBytes: 3 *
				1024 *
				1024 *
				1024,

			MemoryUsageBytes: 1 *
				1024 *
				1024 *
				1024,
		},
	}, nil
}

func (
	m *metricsClientMock,
) GetWorkloadHistory(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (map[string][]WorkloadSample, error) {

	m.contextReceived =
		analysisContext

	return map[string][]WorkloadSample{
		"payments-api": {
			{
				Timestamp: time.Now(),

				CPUUsageMillicores: 600,

				MemoryUsageBytes: 1 *
					1024 *
					1024 *
					1024,
			},
		},
	}, nil
}

type pricingClientMock struct {
	contextReceived domain.AnalysisContext
}

func (
	m *pricingClientMock,
) GetPricing(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (ResourcePrice, error) {

	m.contextReceived =
		analysisContext

	return ResourcePrice{
		CPUPerCoreHour: 0.05,

		MemoryPerGBHour: 0.006,
	}, nil
}

func TestNewProvider_ShouldCreateGenericProvider(
	t *testing.T,
) {

	provider, err :=
		NewProvider(
			&metricsClientMock{},
			&pricingClientMock{},
		)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if provider == nil {
		t.Fatal(
			"expected provider",
		)
	}
}

func TestNewProvider_ShouldRejectNilMetricsClient(
	t *testing.T,
) {

	_, err :=
		NewProvider(
			nil,
			&pricingClientMock{},
		)

	if err == nil {
		t.Fatal(
			"expected metrics client error",
		)
	}
}

func TestNewProvider_ShouldRejectNilPricingClient(
	t *testing.T,
) {

	_, err :=
		NewProvider(
			&metricsClientMock{},
			nil,
		)

	if err == nil {
		t.Fatal(
			"expected pricing client error",
		)
	}
}

func TestMetricsSource_ShouldMapWorkloads(
	t *testing.T,
) {

	client :=
		&metricsClientMock{}

	source :=
		NewMetricsSource(
			client,
		)

	workloads, err :=
		source.GetWorkloads(
			context.Background(),
			domain.AnalysisContext{
				Provider: domain.CloudProviderAWS,

				Environment: "production",

				AccountID: "123456789",

				Region: "us-east-1",
			},
			"payments",
		)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if len(workloads) != 1 {
		t.Fatalf(
			"expected 1 workload, got %d",
			len(workloads),
		)
	}

	if workloads[0].Name !=
		"payments-api" {

		t.Fatalf(
			"expected payments-api, got %s",
			workloads[0].Name,
		)
	}

	if client.contextReceived.Region !=
		"us-east-1" {

		t.Fatalf(
			"expected us-east-1, got %s",
			client.contextReceived.Region,
		)
	}
}

func TestPricingSource_ShouldMapPricing(
	t *testing.T,
) {

	client :=
		&pricingClientMock{}

	source :=
		NewPricingSource(
			client,
		)

	result, err :=
		source.GetPricing(
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
}
