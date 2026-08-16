package gcp

import (
	"context"
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

type fakeGKENodePoolClient struct {
	current int64
	updated int64
	err     error
}

func (f *fakeGKENodePoolClient) GetNodePool(context.Context, string, string, string, string) (int64, error) {
	return f.current, f.err
}

func (f *fakeGKENodePoolClient) SetNodePoolSize(_ context.Context, _ string, _ string, _ string, _ string, desired int64) error {
	if f.err != nil {
		return f.err
	}
	f.updated = desired
	return nil
}

func TestGKEExecutor_ShouldExecuteReduceNodeGroup(t *testing.T) {
	client := &fakeGKENodePoolClient{current: 8}
	executor := NewGKEExecutor(client, "gcp-project", "us-central1")
	action := domain.Action{
		ID:           "action-gcp-1",
		Type:         domain.ActionReduceNodeGroup,
		Provider:     domain.CloudProviderGCP,
		Cluster:      "gke-prod",
		NodeGroup:    "default-pool",
		CurrentValue: 8,
		DesiredValue: 6,
	}
	execution := domain.ExecutionRecord{
		ID:             "exec-gcp-1",
		IdempotencyKey: "plan-1:action-gcp-1",
		PlanID:         "plan-1",
		ActionID:       action.ID,
		Provider:       domain.CloudProviderGCP,
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

func TestGKEExecutor_ShouldRejectDriftBeforeMutation(t *testing.T) {
	client := &fakeGKENodePoolClient{current: 7}
	executor := NewGKEExecutor(client, "gcp-project", "us-central1")
	action := domain.Action{
		ID:           "action-gcp-drift",
		Type:         domain.ActionReduceNodeGroup,
		Provider:     domain.CloudProviderGCP,
		Cluster:      "gke-prod",
		NodeGroup:    "default-pool",
		CurrentValue: 8,
		DesiredValue: 6,
	}
	execution := domain.ExecutionRecord{
		ID:             "exec-gcp-drift",
		IdempotencyKey: "plan-1:action-gcp-drift",
		PlanID:         "plan-1",
		ActionID:       action.ID,
		Provider:       domain.CloudProviderGCP,
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
