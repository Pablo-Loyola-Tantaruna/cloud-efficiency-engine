package kubernetes

import (
	"context"
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

type capacityPrometheusMock struct{}

func (
	m *capacityPrometheusMock,
) QueryInstant(
	ctx context.Context,
	query string,
) ([]map[string]string, []float64, error) {

	switch query {

	case `sum(kube_node_status_allocatable{resource="cpu"})`:

		return nil,
			[]float64{
				8,
			},
			nil

	case `sum(kube_node_status_allocatable{resource="memory"})`:

		return nil,
			[]float64{
				16 * 1024 * 1024 * 1024,
			},
			nil

	default:

		return nil,
			nil,
			nil
	}
}

func TestCapacitySource_ShouldResolveKubernetesCapacity(
	t *testing.T,
) {

	source :=
		NewCapacitySource(
			&capacityPrometheusMock{},
		)

	cpu, memory, err :=
		source.GetCapacity(
			context.Background(),
			domain.AnalysisContext{
				Provider: domain.CloudProviderKubernetes,
			},
		)

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if cpu != 8000 {

		t.Fatalf(
			"expected 8000 millicores, got %d",
			cpu,
		)
	}

	expectedMemory :=
		int64(
			16 *
				1024 *
				1024 *
				1024,
		)

	if memory != expectedMemory {

		t.Fatalf(
			"expected memory %d, got %d",
			expectedMemory,
			memory,
		)
	}
}

func TestCapacitySource_ShouldRejectMissingMetrics(
	t *testing.T,
) {

	source :=
		NewCapacitySource(
			&emptyCapacityPrometheusMock{},
		)

	_, _, err :=
		source.GetCapacity(
			context.Background(),
			domain.AnalysisContext{
				Provider: domain.CloudProviderKubernetes,
			},
		)

	if err == nil {

		t.Fatal(
			"expected error",
		)
	}
}

type emptyCapacityPrometheusMock struct{}

func (
	m *emptyCapacityPrometheusMock,
) QueryInstant(
	ctx context.Context,
	query string,
) ([]map[string]string, []float64, error) {

	return nil, nil, nil
}
