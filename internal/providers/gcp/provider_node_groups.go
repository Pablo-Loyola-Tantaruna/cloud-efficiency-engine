package gcp

import (
	"context"
	"fmt"

	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

func (p *Provider) GetNodeGroups(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) ([]cost.NodeGroupCapacity, error) {
	if p == nil || p.capacityClient == nil {
		return nil, fmt.Errorf("GCP capacity provider is not configured")
	}

	nodeGroupClient, ok := p.capacityClient.(interface {
		GetNodeGroups(context.Context, domain.AnalysisContext) ([]cost.NodeGroupCapacity, error)
	})
	if !ok {
		return nil, nil
	}

	return nodeGroupClient.GetNodeGroups(ctx, analysisContext)
}
