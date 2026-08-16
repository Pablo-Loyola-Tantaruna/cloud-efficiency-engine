package azure

import (
	"context"
	"fmt"

	"cloud-efficiency-engine/internal/domain"
)

type AKSStateReader struct {
	client        AKSNodePoolClient
	resourceGroup string
}

func NewAKSStateReader(client AKSNodePoolClient, resourceGroup string) *AKSStateReader {
	return &AKSStateReader{client: client, resourceGroup: resourceGroup}
}

func (r *AKSStateReader) ReadState(ctx context.Context, action domain.Action) (domain.ObservedState, error) {
	if r == nil || r.client == nil {
		return domain.ObservedState{}, fmt.Errorf("Azure AKS state reader client must not be nil")
	}
	if r.resourceGroup == "" {
		return domain.ObservedState{}, fmt.Errorf("Azure AKS resource group must not be empty")
	}
	if action.Provider != domain.CloudProviderAzure {
		return domain.ObservedState{}, fmt.Errorf("Azure AKS state reader requires Azure provider")
	}
	if action.Cluster == "" || action.NodeGroup == "" {
		return domain.ObservedState{}, fmt.Errorf("Azure AKS cluster and node pool must not be empty")
	}

	value, err := r.client.GetNodePool(ctx, r.resourceGroup, action.Cluster, action.NodeGroup)
	if err != nil {
		return domain.ObservedState{}, fmt.Errorf("read Azure AKS node pool state: %w", err)
	}
	return domain.ObservedState{CurrentValue: value}, nil
}

var _ domain.StateReader = (*AKSStateReader)(nil)
