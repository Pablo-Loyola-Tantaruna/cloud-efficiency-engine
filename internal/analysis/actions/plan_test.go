package actions

import (
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func TestBuildActionPlan_ShouldBuildPreview(t *testing.T) {
	actions := []domain.Action{
		{
			ID: "action-1", Type: domain.ActionReduceNodeGroup,
			Provider: domain.CloudProviderAWS, Cluster: "prod",
			NodeGroup: "workers", CurrentValue: 8, DesiredValue: 6,
			MonthlySavingsUSD: 200, AnnualizedSavingsUSD: 2400,
			Risk: domain.ActionRiskMedium, RequiresApproval: true,
		},
	}

	plan, err := BuildActionPlan(domain.CloudProviderAWS, "prod", actions)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != domain.ActionPlanPreview {
		t.Fatalf("expected preview status, got %q", plan.Status)
	}
	if plan.ID == "" {
		t.Fatal("expected plan id")
	}
	if plan.TotalMonthlySavingsUSD != 200 || plan.TotalAnnualizedSavingsUSD != 2400 {
		t.Fatalf("unexpected totals: %.2f / %.2f", plan.TotalMonthlySavingsUSD, plan.TotalAnnualizedSavingsUSD)
	}
	if !plan.RequiresApproval {
		t.Fatal("expected approval to be required")
	}
}

func TestBuildActionPlan_ShouldBeDeterministic(t *testing.T) {
	action := domain.Action{
		ID: "action-1", Type: domain.ActionReduceNodeGroup,
		Provider: domain.CloudProviderAzure, Cluster: "prod",
		NodeGroup: "workers", CurrentValue: 8, DesiredValue: 6,
		MonthlySavingsUSD: 150, AnnualizedSavingsUSD: 1800,
		Risk: domain.ActionRiskMedium, RequiresApproval: true,
	}
	first, err := BuildActionPlan(domain.CloudProviderAzure, "prod", []domain.Action{action})
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildActionPlan(domain.CloudProviderAzure, "prod", []domain.Action{action})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected deterministic plan id, got %q and %q", first.ID, second.ID)
	}
}

func TestBuildActionPlan_ShouldRejectMixedProviderOrCluster(t *testing.T) {
	action := domain.Action{
		ID: "action-1", Type: domain.ActionReduceNodeGroup,
		Provider: domain.CloudProviderGCP, Cluster: "other",
		NodeGroup: "workers", CurrentValue: 4, DesiredValue: 2,
		MonthlySavingsUSD: 100, AnnualizedSavingsUSD: 1200,
		Risk: domain.ActionRiskMedium, RequiresApproval: true,
	}
	if _, err := BuildActionPlan(domain.CloudProviderAWS, "prod", []domain.Action{action}); err == nil {
		t.Fatal("expected mixed provider/cluster to be rejected")
	}
}
