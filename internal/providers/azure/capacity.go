package azure

import (
	"context"
	"fmt"

	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

type CapacitySource interface {
	GetCapacity(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
	) (int64, int64, error)
}

type CapacityProvider struct {
	source CapacitySource
}

func NewCapacityProvider(
	source CapacitySource,
) *CapacityProvider {

	return &CapacityProvider{
		source: source,
	}
}

func (
	p *CapacityProvider,
) GetCapacity(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (cost.ClusterCapacity, error) {

	if p == nil ||
		p.source == nil {

		return cost.ClusterCapacity{},
			fmt.Errorf(
				"Azure capacity provider is not configured",
			)
	}

	cpu, memory, err :=
		p.source.GetCapacity(
			ctx,
			analysisContext,
		)

	if err != nil {

		return cost.ClusterCapacity{},
			err
	}

	return cost.ClusterCapacity{
		CPUCapacityMillicores: cpu,

		MemoryCapacityBytes: memory,
	}, nil
}
