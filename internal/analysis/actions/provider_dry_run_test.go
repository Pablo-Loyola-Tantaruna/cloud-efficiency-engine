package actions

import (
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func TestRenderProviderDryRun_ShouldRenderAWSNodeGroupChange(t *testing.T) {
	operation := ExecutionOperation{
		ActionID:     "action-1",
		Type:         domain.ActionReduceNodeGroup,
		Provider:     domain.CloudProviderAWS,
		Cluster:      "prod",
		NodeGroup:    "workers",
		CurrentValue: 8,
		DesiredValue: 6,
	}

	change, err := RenderProviderDryRun(operation)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !change.DryRun {
		t.Fatal("expected dry-run change")
	}
	if change.Operation != "UPDATE_EKS_NODE_GROUP_DESIRED_SIZE" {
		t.Fatalf("unexpected operation %q", change.Operation)
	}
	if change.Target != "prod/workers" {
		t.Fatalf("unexpected target %q", change.Target)
	}
}

func TestRenderProviderDryRun_ShouldRenderAzureNodePoolChange(t *testing.T) {
	operation := ExecutionOperation{
		ActionID:     "action-1",
		Type:         domain.ActionReduceNodeGroup,
		Provider:     domain.CloudProviderAzure,
		Cluster:      "prod",
		NodeGroup:    "userpool",
		CurrentValue: 8,
		DesiredValue: 6,
	}

	change, err := RenderProviderDryRun(operation)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if change.Operation != "UPDATE_AKS_NODE_POOL_COUNT" {
		t.Fatalf("unexpected operation %q", change.Operation)
	}
}

func TestRenderProviderDryRun_ShouldRenderGCPNodePoolChange(t *testing.T) {
	operation := ExecutionOperation{
		ActionID:     "action-1",
		Type:         domain.ActionReduceNodeGroup,
		Provider:     domain.CloudProviderGCP,
		Cluster:      "prod",
		NodeGroup:    "default-pool",
		CurrentValue: 10,
		DesiredValue: 7,
	}

	change, err := RenderProviderDryRun(operation)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if change.Operation != "RESIZE_GKE_NODE_POOL" {
		t.Fatalf("unexpected operation %q", change.Operation)
	}
}

func TestRenderExecutionProviderChanges_ShouldRejectNonDryRunPlan(t *testing.T) {
	plan := ExecutionPlan{Mode: "APPLY"}
	if _, err := RenderExecutionProviderChanges(plan); err == nil {
		t.Fatal("expected error for non dry-run plan")
	}
}

func TestRenderProviderDryRun_ShouldRejectMissingNodeGroup(t *testing.T) {
	operation := ExecutionOperation{
		ActionID:     "action-1",
		Type:         domain.ActionReduceNodeGroup,
		Provider:     domain.CloudProviderAWS,
		Cluster:      "prod",
		CurrentValue: 8,
		DesiredValue: 6,
	}
	if _, err := RenderProviderDryRun(operation); err == nil {
		t.Fatal("expected missing node group error")
	}
}
