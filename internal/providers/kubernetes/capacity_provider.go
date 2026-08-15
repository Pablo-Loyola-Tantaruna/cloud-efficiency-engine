package kubernetes

import (
	"context"

	"cloud-efficiency-engine/internal/analysis/capacity"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

type CapacityProvider struct {
	source *CapacitySource
}

func NewCapacityProvider(
	source *CapacitySource,
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

	if p == nil || p.source == nil {
		return cost.ClusterCapacity{},
			&CapacityProviderError{
				message: "Kubernetes capacity provider is not configured",
			}
	}

	cpu, memory, err :=
		p.source.GetCapacity(
			ctx,
			analysisContext,
		)

	if err != nil {
		return cost.ClusterCapacity{}, err
	}

	return cost.ClusterCapacity{
		CPUCapacityMillicores: cpu,
		MemoryCapacityBytes:   memory,
	}, nil
}

type CapacityProviderError struct {
	message string
}

func (e *CapacityProviderError) Error() string {
	return e.message
}

var _ capacity.Provider = (*CapacityProvider)(nil)
