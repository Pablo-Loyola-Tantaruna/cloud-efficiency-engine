package azure

import (
	"context"
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

type fakeAKSNodePoolClient struct {
	current int64
	updated int64
	err     error
}

func (f *fakeAKSNodePoolClient) GetNodePool(context.Context, string, string, string) (int64, error) {
	return f.current, f.err
}

func (f *fakeAKSNodePoolClient) SetNodePoolSize(_ context.Context, _ string, _ string, _ string, desired int64) error {
	if f.err != nil {
		return f.err
	}
	f.updated = desired
	return nil
}

func TestAKSExecutor_ShouldExecuteReduceNodeGroup(t *testing.T) {
	client := &fakeAKSNodePoolClient{current: 8}
	executor := NewAKSExecutor(client, "rg-finops")
	action := domain.Action{
		ID:           "action-azure-1",
		Type:         domain.ActionReduceNodeGroup,
		Provider:     domain.CloudProviderAzure,
		Cluster:      "aks-prod",
		NodeGroup:    "system",
		CurrentValue: 8,
		DesiredValue: 6,
	}
	execution := domain.ExecutionRecord{
		ID:             "exec-azure-1",
		IdempotencyKey: "plan-1:action-azure-1",
		PlanID:         "plan-1",
		ActionID:       action.ID,
		Provider:       domain.CloudProviderAzure,
		Cluster:        action.Cluster,
		Status:         domain.ExecutionStatusRunning,
		Attempt:        1,
		CurrentValue:   8,
		DesiredValue:   6,
	}

	result, err := executor.Execute(context.Background(), action, execution)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Status != domain.ExecutionResultSucceeded {
		t.Fatalf("expected SUCCEEDED, got %s", result.Status)
	}
	if client.updated != 6 {
		t.Fatalf("expected desired size 6, got %d", client.updated)
	}
}

func TestAKSExecutor_ShouldRejectDriftBeforeMutation(t *testing.T) {
	client := &fakeAKSNodePoolClient{current: 7}
	executor := NewAKSExecutor(client, "rg-finops")
	action := domain.Action{
		ID:           "action-azure-drift",
		Type:         domain.ActionReduceNodeGroup,
		Provider:     domain.CloudProviderAzure,
		Cluster:      "aks-prod",
		NodeGroup:    "system",
		CurrentValue: 8,
		DesiredValue: 6,
	}
	execution := domain.ExecutionRecord{
		ID:             "exec-azure-drift",
		IdempotencyKey: "plan-1:action-azure-drift",
		PlanID:         "plan-1",
		ActionID:       action.ID,
		Provider:       domain.CloudProviderAzure,
		Cluster:        action.Cluster,
		Status:         domain.ExecutionStatusRunning,
		Attempt:        1,
		CurrentValue:   8,
		DesiredValue:   6,
	}

	if _, err := executor.Execute(context.Background(), action, execution); err == nil {
		t.Fatal("expected drift error")
	}
	if client.updated != 0 {
		t.Fatalf("expected no mutation, got %d", client.updated)
	}
}
