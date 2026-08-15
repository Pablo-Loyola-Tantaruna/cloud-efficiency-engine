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

func NewCapacitySourceAdapter(
	source CapacitySource,
) *CapacitySourceAdapter {

	return &CapacitySourceAdapter{
		source: source,
	}
}

func (
	a *CapacitySourceAdapter,
) GetCapacity(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (cost.ClusterCapacity, error) {

	cpu, memory, err :=
		a.source.GetCapacity(
			ctx,
			analysisContext,
		)

	if err != nil {
		return cost.ClusterCapacity{}, err
	}

	return cost.ClusterCapacity{
		CPUCapacityMillicores: cpu,

		MemoryCapacityBytes: memory,
	}, nil
}

var _ capacity.Provider = (*CapacitySourceAdapter)(nil)
