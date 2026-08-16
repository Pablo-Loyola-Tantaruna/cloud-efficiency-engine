package aws

import (
	"context"
	"fmt"

	"cloud-efficiency-engine/internal/domain"
)

// EKSNodeGroupClient is the provider-facing contract for managed EKS node groups.
// Implementations own AWS SDK details and must not leak them to the action layer.
type EKSNodeGroupClient interface {
	DescribeDesiredSize(ctx context.Context, cluster, nodeGroup string) (int64, error)
	UpdateDesiredSize(ctx context.Context, cluster, nodeGroup string, desired int64, clientRequestToken string) (string, error)
	WaitForUpdate(ctx context.Context, cluster, nodeGroup, updateID string) error
}

type EKSExecutor struct {
	client EKSNodeGroupClient
}

func NewEKSExecutor(client EKSNodeGroupClient) *EKSExecutor {
	return &EKSExecutor{client: client}
}

func (e *EKSExecutor) Execute(ctx context.Context, action domain.Action, execution domain.ExecutionRecord) (domain.ExecutionResult, error) {
	if e == nil || e.client == nil {
		return domain.ExecutionResult{}, fmt.Errorf("aws eks executor client must not be nil")
	}
	if execution.Status != domain.ExecutionStatusRunning {
		return domain.ExecutionResult{}, fmt.Errorf("execution %q must be RUNNING", execution.ID)
	}
	if action.Type != domain.ActionReduceNodeGroup {
		return domain.ExecutionResult{}, fmt.Errorf("aws eks executor does not support action type %q", action.Type)
	}
	if action.Provider != domain.CloudProviderAWS || execution.Provider != domain.CloudProviderAWS {
		return domain.ExecutionResult{}, fmt.Errorf("aws eks executor requires AWS provider")
	}
	if action.Cluster != execution.Cluster || action.ID != execution.ActionID {
		return domain.ExecutionResult{}, fmt.Errorf("action %q does not match execution %q", action.ID, execution.ID)
	}
	if action.NodeGroup == "" {
		return domain.ExecutionResult{}, fmt.Errorf("node group must not be empty")
	}
	if action.CurrentValue != execution.CurrentValue || action.DesiredValue != execution.DesiredValue {
		return domain.ExecutionResult{}, fmt.Errorf("action values do not match execution values")
	}

	observedBefore, err := e.client.DescribeDesiredSize(ctx, action.Cluster, action.NodeGroup)
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("describe EKS node group before execution: %w", err)
	}
	if observedBefore != action.CurrentValue {
		return domain.ExecutionResult{}, fmt.Errorf("EKS node group %q drifted before execution: expected %d, observed %d", action.NodeGroup, action.CurrentValue, observedBefore)
	}

	updateID, err := e.client.UpdateDesiredSize(ctx, action.Cluster, action.NodeGroup, action.DesiredValue, execution.IdempotencyKey)
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("update EKS node group %q: %w", action.NodeGroup, err)
	}
	if updateID == "" {
		return domain.ExecutionResult{}, fmt.Errorf("EKS node group %q update returned an empty update id", action.NodeGroup)
	}
	if err := e.client.WaitForUpdate(ctx, action.Cluster, action.NodeGroup, updateID); err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("wait for EKS node group %q update %q: %w", action.NodeGroup, updateID, err)
	}

	result := domain.ExecutionResult{
		Status:       domain.ExecutionResultSucceeded,
		ExecutionID:  execution.ID,
		Provider:     domain.CloudProviderAWS,
		Cluster:      action.Cluster,
		ActionID:     action.ID,
		BeforeValue:  observedBefore,
		DesiredValue: action.DesiredValue,
		Message:      fmt.Sprintf("EKS node group %q desired size updated from %d to %d (update %s)", action.NodeGroup, observedBefore, action.DesiredValue, updateID),
	}
	if err := result.Validate(); err != nil {
		return domain.ExecutionResult{}, err
	}
	return result, nil
}
