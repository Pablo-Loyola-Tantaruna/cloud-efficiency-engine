package providers

import (
	"context"

	"cloud-efficiency-engine/internal/domain"
)

type MockProvider struct {
	workloads []domain.WorkloadMetrics
}

func NewMockProvider(
	workloads []domain.WorkloadMetrics,
) *MockProvider {

	return &MockProvider{
		workloads: workloads,
	}
}

func (p *MockProvider) GetWorkloads(
	ctx context.Context,
	namespace string,
) ([]domain.WorkloadMetrics, error) {

	result :=
		make(
			[]domain.WorkloadMetrics,
			0,
			len(p.workloads),
		)

	for _, workload := range p.workloads {

		if workload.Namespace != namespace {
			continue
		}

		result =
			append(
				result,
				workload,
			)
	}

	return result, nil
}
