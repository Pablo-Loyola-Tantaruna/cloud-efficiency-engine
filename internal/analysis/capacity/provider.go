package capacity

import (
	"context"

	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

type Provider interface {
	GetCapacity(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
	) (cost.ClusterCapacity, error)
}
