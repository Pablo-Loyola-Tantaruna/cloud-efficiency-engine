package metrics

import (
	"context"

	"cloud-efficiency-engine/internal/domain"
)

type Provider interface {
	GetWorkloads(ctx context.Context) ([]domain.WorkloadMetrics, error)
}
