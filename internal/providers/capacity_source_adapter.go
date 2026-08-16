package providers

import (
	"context"

	"cloud-efficiency-engine/internal/analysis/capacity"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

type CapacitySourceAdapter struct {
	source CapacitySource
}

func NewCapacitySourceAdapter(source CapacitySource) *CapacitySourceAdapter {
	return &CapacitySourceAdapter{source: source}
}

func (a *CapacitySourceAdapter) GetCapacity(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (cost.ClusterCapacity, error) {
	if a == nil || a.source == nil {
		return cost.ClusterCapacity{}, nil
	}

	cpu, memory, err := a.source.GetCapacity(ctx, analysisContext)
	if err != nil {
		return cost.ClusterCapacity{}, err
	}

	cluster := cost.ClusterCapacity{
		CPUCapacityMillicores: cpu,
		MemoryCapacityBytes:   memory,
	}

	if nodeSource, ok := a.source.(NodeCountSource); ok {
		nodeCount, nodeErr := nodeSource.GetNodeCount(ctx, analysisContext)
		if nodeErr != nil {
			return cost.ClusterCapacity{}, nodeErr
		}
		cluster.NodeCount = nodeCount
	}

	if nodeGroupSource, ok := a.source.(interface {
		GetNodeGroups(context.Context, domain.AnalysisContext) ([]cost.NodeGroupCapacity, error)
	}); ok {
		nodeGroups, groupErr := nodeGroupSource.GetNodeGroups(ctx, analysisContext)
		if groupErr != nil {
			return cost.ClusterCapacity{}, groupErr
		}
		cluster.NodeGroups = nodeGroups
	}

	return cluster, nil
}

func (a *CapacitySourceAdapter) GetNodeGroups(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) ([]cost.NodeGroupCapacity, error) {
	if a == nil || a.source == nil {
		return nil, nil
	}

	nodeGroupSource, ok := a.source.(interface {
		GetNodeGroups(context.Context, domain.AnalysisContext) ([]cost.NodeGroupCapacity, error)
	})
	if !ok {
		return nil, nil
	}

	return nodeGroupSource.GetNodeGroups(ctx, analysisContext)
}

var _ capacity.Provider = (*CapacitySourceAdapter)(nil)
var _ capacity.NodeGroupProvider = (*CapacitySourceAdapter)(nil)
