package kubernetes

import (
	"context"
	"fmt"

	"cloud-efficiency-engine/internal/domain"
)

type PrometheusQueryClient interface {
	QueryInstant(
		ctx context.Context,
		query string,
	) (
		[]map[string]string,
		[]float64,
		error,
	)
}

type CapacitySource struct {
	client PrometheusQueryClient
}

func NewCapacitySource(
	client PrometheusQueryClient,
) *CapacitySource {

	return &CapacitySource{
		client: client,
	}
}

func (
	s *CapacitySource,
) GetCapacity(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (int64, int64, error) {

	if s == nil ||
		s.client == nil {

		return 0,
			0,
			fmt.Errorf(
				"Kubernetes capacity source is not configured",
			)
	}

	_, cpuValues, err :=
		s.client.QueryInstant(
			ctx,
			`sum(kube_node_status_allocatable{resource="cpu"})`,
		)

	if err != nil {
		return 0, 0, err
	}

	_, memoryValues, err :=
		s.client.QueryInstant(
			ctx,
			`sum(kube_node_status_allocatable{resource="memory"})`,
		)

	if err != nil {
		return 0, 0, err
	}

	if len(cpuValues) == 0 ||
		len(memoryValues) == 0 {

		return 0,
			0,
			fmt.Errorf(
				"Kubernetes capacity metrics returned no data",
			)
	}

	cpuCores :=
		cpuValues[0]

	memoryBytes :=
		memoryValues[0]

	if cpuCores <= 0 {

		return 0,
			0,
			fmt.Errorf(
				"Kubernetes CPU capacity must be greater than zero",
			)
	}

	if memoryBytes <= 0 {

		return 0,
			0,
			fmt.Errorf(
				"Kubernetes memory capacity must be greater than zero",
			)
	}

	return int64(cpuCores * 1000),
		int64(memoryBytes),
		nil
}
