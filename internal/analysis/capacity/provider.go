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

type NodeGroupProvider interface {
	GetNodeGroups(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
	) ([]cost.NodeGroupCapacity, error)
}
