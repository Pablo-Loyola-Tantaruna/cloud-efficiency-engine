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

type ContextAwareProvider interface {
	GetWorkloadsWithContext(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
		namespace string,
	) ([]domain.WorkloadMetrics, error)
}
