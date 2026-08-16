package actions

import (
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

func TestApproveActionPlan_ShouldRecordApprovalAndChangeStatus(t *testing.T) {
	plan := domain.ActionPlan{
		ID:       "plan-1",
		Provider: domain.CloudProviderAWS,
		Cluster:  "production",
		Status:   domain.ActionPlanPendingApproval,
		Actions: []domain.Action{{
			ID:                   "action-1",
			Type:                 domain.ActionReduceNodeGroup,
			Provider:             domain.CloudProviderAWS,
			Cluster:              "production",
			CurrentValue:         8,
			DesiredValue:         6,
			MonthlySavingsUSD:    100,
			AnnualizedSavingsUSD: 1200,
			Risk:                 domain.ActionRiskMedium,
			RequiresApproval:     true,
		}},
		TotalMonthlySavingsUSD:    100,
		TotalAnnualizedSavingsUSD: 1200,
		RequiresApproval:          true,
	}
	approvedAt := time.Date(2026, 8, 16, 19, 0, 0, 0, time.FixedZone("-05", -5*60*60))

	updated, approval, err := ApproveActionPlan(plan, "sergi", "looks good", approvedAt)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.ActionPlanApproved {
		t.Fatalf("expected approved status, got %q", updated.Status)
	}
	if approval.PlanID != "plan-1" || approval.ApprovedBy != "sergi" {
		t.Fatalf("unexpected approval: %+v", approval)
	}
	if !approval.ApprovedAt.Equal(approvedAt.UTC()) {
		t.Fatalf("unexpected approval time: %v", approval.ApprovedAt)
	}
	if approval.Comment != "looks good" {
		t.Fatalf("unexpected comment: %q", approval.Comment)
	}
}

func TestApproveActionPlan_ShouldRejectWrongStatus(t *testing.T) {
	plan := domain.ActionPlan{ID: "plan-1", Provider: domain.CloudProviderAWS, Cluster: "production", Status: domain.ActionPlanPreview, Actions: []domain.Action{{ID: "a", Type: domain.ActionReduceNodeGroup, Provider: domain.CloudProviderAWS, Cluster: "production", CurrentValue: 8, DesiredValue: 6, MonthlySavingsUSD: 100, AnnualizedSavingsUSD: 1200, Risk: domain.ActionRiskMedium, RequiresApproval: true}}, TotalMonthlySavingsUSD: 100, TotalAnnualizedSavingsUSD: 1200, RequiresApproval: true}
	_, _, err := ApproveActionPlan(plan, "sergi", "", time.Time{})
	if err == nil {
		t.Fatal("expected error")
	}
}
