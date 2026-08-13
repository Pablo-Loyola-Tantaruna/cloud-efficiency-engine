package metrics

import (
	"context"

	"cloud-efficiency-engine/internal/domain"
)

type Provider interface {
	GetWorkloads(
		ctx context.Context,
		namespace string,
	) ([]domain.WorkloadMetrics, error)
}
