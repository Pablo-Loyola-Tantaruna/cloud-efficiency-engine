package kubernetes

import (
	"cloud-efficiency-engine/internal/cost"
	"context"
	"testing"

	"cloud-efficiency-engine/internal/domain"
	providerregistry "cloud-efficiency-engine/internal/providers"
)

type registryCapacityMock struct{}

func (
	m *registryCapacityMock,
) GetCapacity(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (cost.ClusterCapacity, error) {

	return cost.ClusterCapacity{
		CPUCapacityMillicores: 8000,
		MemoryCapacityBytes:   16 * 1024 * 1024 * 1024,
	}, nil
}

func TestRegister_ShouldRegisterKubernetesProviders(
	t *testing.T,
) {

	registry :=
		providerregistry.NewRegistry()

	capacityProvider :=
		&registryCapacityMock{}

	err :=
		Register(
			registry,
			"http://localhost:9090",
			0.04,
			0.005,
			capacityProvider,
		)

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	bundle, err :=
		registry.Resolve(
			context.Background(),
			domain.AnalysisContext{
				Provider: domain.CloudProviderKubernetes,
			},
		)

	if err != nil {

		t.Fatalf(
			"expected registry resolution to succeed, got %v",
			err,
		)
	}

	if bundle == nil {

		t.Fatal(
			"expected bundle",
		)
	}

	if bundle.MetricsProvider == nil {

		t.Fatal(
			"expected metrics provider",
		)
	}

	if bundle.PricingProvider == nil {

		t.Fatal(
			"expected pricing provider",
		)
	}

	if bundle.CapacityProvider == nil {

		t.Fatal(
			"expected capacity provider",
		)
	}
}
