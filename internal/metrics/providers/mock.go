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
) ([]domain.WorkloadMetrics, error) {
	return p.workloads, nil
}
