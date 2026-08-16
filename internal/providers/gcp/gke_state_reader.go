package gcp

import (
	"context"
	"fmt"

	"cloud-efficiency-engine/internal/domain"
)

type GKEStateReader struct {
	client    GKENodePoolClient
	projectID string
	location  string
}

func NewGKEStateReader(client GKENodePoolClient, projectID, location string) *GKEStateReader {
	return &GKEStateReader{client: client, projectID: projectID, location: location}
}

func (r *GKEStateReader) ReadState(ctx context.Context, action domain.Action) (domain.ObservedState, error) {
	if r == nil || r.client == nil {
		return domain.ObservedState{}, fmt.Errorf("GCP GKE state reader client must not be nil")
	}
	if r.projectID == "" || r.location == "" {
		return domain.ObservedState{}, fmt.Errorf("GCP project ID and location must not be empty")
	}
	if action.Provider != domain.CloudProviderGCP {
		return domain.ObservedState{}, fmt.Errorf("GCP GKE state reader requires GCP provider")
	}
	if action.Cluster == "" || action.NodeGroup == "" {
		return domain.ObservedState{}, fmt.Errorf("GCP GKE cluster and node pool must not be empty")
	}

	value, err := r.client.GetNodePool(ctx, r.projectID, r.location, action.Cluster, action.NodeGroup)
	if err != nil {
		return domain.ObservedState{}, fmt.Errorf("read GCP GKE node pool state: %w", err)
	}
	return domain.ObservedState{CurrentValue: value}, nil
}

var _ domain.StateReader = (*GKEStateReader)(nil)
