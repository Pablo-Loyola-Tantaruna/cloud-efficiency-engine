package actions

import (
	"strings"
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func TestBuildExecutionPreview_ShouldCombineProviderChangesAndDiff(t *testing.T) {
	plan := domain.ActionPlan{
		ID: "plan-1", Provider: domain.CloudProviderAWS, Cluster: "production",
		Status:                 domain.ActionPlanReadyToApply,
		TotalMonthlySavingsUSD: 300, TotalAnnualizedSavingsUSD: 3600,
		RequiresApproval: true,
		Actions: []domain.Action{
			{ID: "b", Type: domain.ActionReduceNodeGroup, Provider: domain.CloudProviderAWS, Cluster: "production", NodeGroup: "workers", CurrentValue: 8, DesiredValue: 6, MonthlySavingsUSD: 200, AnnualizedSavingsUSD: 2400, Risk: domain.ActionRiskHigh, RequiresApproval: true},
			{ID: "a", Type: domain.ActionRightsizeWorkloadCPU, Provider: domain.CloudProviderAWS, Cluster: "production", Workload: "payments/api", CurrentValue: 2000, DesiredValue: 750, MonthlySavingsUSD: 100, AnnualizedSavingsUSD: 1200, Risk: domain.ActionRiskLow, RequiresApproval: true},
		},
	}

	preview, err := BuildExecutionPreview(plan)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if preview.Mode != ExecutionModeDryRun {
		t.Fatalf("expected dry-run mode, got %q", preview.Mode)
	}
	if len(preview.Changes) != 2 {
		t.Fatalf("expected two changes, got %d", len(preview.Changes))
	}
	if preview.Changes[0].ActionID != "a" || preview.Changes[1].ActionID != "b" {
		t.Fatalf("expected deterministic change ordering")
	}
	if preview.HighestRisk != domain.ActionRiskHigh {
		t.Fatalf("expected highest risk HIGH, got %q", preview.HighestRisk)
	}
	if !preview.ApprovalRequired {
		t.Fatal("expected approval to be required")
	}
	if !strings.Contains(preview.Diff, "UPDATE_EKS_NODE_GROUP_DESIRED_SIZE") {
		t.Fatal("expected node group operation in diff")
	}
	if !strings.Contains(preview.Diff, "- value: 8") || !strings.Contains(preview.Diff, "+ value: 6") {
		t.Fatal("expected value diff")
	}
}

func TestBuildExecutionPreview_ShouldRejectPlanBeforeReadyToApply(t *testing.T) {
	plan := domain.ActionPlan{ID: "plan-1", Provider: domain.CloudProviderAWS, Cluster: "production", Status: domain.ActionPlanApproved, TotalMonthlySavingsUSD: 100, TotalAnnualizedSavingsUSD: 1200, RequiresApproval: true, Actions: []domain.Action{{ID: "a", Type: domain.ActionReduceNodeGroup, Provider: domain.CloudProviderAWS, Cluster: "production", NodeGroup: "workers", CurrentValue: 4, DesiredValue: 2, MonthlySavingsUSD: 100, AnnualizedSavingsUSD: 1200, Risk: domain.ActionRiskMedium, RequiresApproval: true}}}
	if _, err := BuildExecutionPreview(plan); err == nil {
		t.Fatal("expected non-ready plan to be rejected")
	}
}
