package aws

import (
	"context"
	"fmt"

	"cloud-efficiency-engine/internal/domain"
)

type EKSStateReader struct {
	client EKSNodeGroupClient
}

func NewEKSStateReader(client EKSNodeGroupClient) *EKSStateReader {
	return &EKSStateReader{client: client}
}

func (r *EKSStateReader) ReadState(ctx context.Context, action domain.Action) (domain.ObservedState, error) {
	if r == nil || r.client == nil {
		return domain.ObservedState{}, fmt.Errorf("EKS state reader client must not be nil")
	}
	if action.Provider != domain.CloudProviderAWS {
		return domain.ObservedState{}, fmt.Errorf("EKS state reader requires AWS provider")
	}
	if action.Type != domain.ActionReduceNodeGroup {
		return domain.ObservedState{}, fmt.Errorf("EKS state reader does not support action type %q", action.Type)
	}
	if action.Cluster == "" || action.NodeGroup == "" {
		return domain.ObservedState{}, fmt.Errorf("EKS state reader requires cluster and node group")
	}
	value, err := r.client.DescribeDesiredSize(ctx, action.Cluster, action.NodeGroup)
	if err != nil {
		return domain.ObservedState{}, err
	}
	if value <= 0 {
		return domain.ObservedState{}, fmt.Errorf("EKS node group %q returned invalid desired size %d", action.NodeGroup, value)
	}
	return domain.ObservedState{CurrentValue: value}, nil
}
