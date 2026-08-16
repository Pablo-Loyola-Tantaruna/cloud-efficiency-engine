package actions

import (
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func validPlan(status domain.ActionPlanStatus) domain.ActionPlan {
	return domain.ActionPlan{
		ID:       "plan-1",
		Provider: domain.CloudProviderAWS,
		Cluster:  "production",
		Status:   status,
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
}

func TestTransitionActionPlan_ShouldFollowApprovalLifecycle(t *testing.T) {
	plan := validPlan(domain.ActionPlanPreview)

	var err error
	plan, err = TransitionActionPlan(plan, domain.ActionPlanPendingApproval)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != domain.ActionPlanPendingApproval {
		t.Fatalf("expected pending approval, got %s", plan.Status)
	}

	plan, err = TransitionActionPlan(plan, domain.ActionPlanApproved)
	if err != nil {
		t.Fatal(err)
	}
	plan, err = TransitionActionPlan(plan, domain.ActionPlanReadyToApply)
	if err != nil {
		t.Fatal(err)
	}
	plan, err = TransitionActionPlan(plan, domain.ActionPlanApplied)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTransitionActionPlan_ShouldAllowFailureBeforeApply(t *testing.T) {
	plan := validPlan(domain.ActionPlanPendingApproval)

	updated, err := TransitionActionPlan(plan, domain.ActionPlanFailed)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.ActionPlanFailed {
		t.Fatalf("expected failed, got %s", updated.Status)
	}
}

func TestTransitionActionPlan_ShouldRejectSkippingApproval(t *testing.T) {
	plan := validPlan(domain.ActionPlanPreview)

	_, err := TransitionActionPlan(plan, domain.ActionPlanReadyToApply)
	if err == nil {
		t.Fatal("expected transition error")
	}
}

func TestTransitionActionPlan_ShouldRejectTransitionFromTerminalState(t *testing.T) {
	plan := validPlan(domain.ActionPlanApplied)

	_, err := TransitionActionPlan(plan, domain.ActionPlanFailed)
	if err == nil {
		t.Fatal("expected transition error")
	}
}
