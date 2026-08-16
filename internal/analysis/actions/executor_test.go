package actions

import (
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func TestBuildDryRunExecution_ShouldBuildDeterministicOperations(t *testing.T) {
	plan := domain.ActionPlan{
		ID:                        "plan-1",
		Provider:                  domain.CloudProviderAWS,
		Cluster:                   "production",
		Status:                    domain.ActionPlanReadyToApply,
		TotalMonthlySavingsUSD:    300,
		TotalAnnualizedSavingsUSD: 3600,
		RequiresApproval:          true,
		Actions: []domain.Action{
			{
				ID:                   "b",
				Type:                 domain.ActionReduceNodeGroup,
				Provider:             domain.CloudProviderAWS,
				Cluster:              "production",
				CurrentValue:         8,
				DesiredValue:         6,
				MonthlySavingsUSD:    200,
				AnnualizedSavingsUSD: 2400,
				Risk:                 domain.ActionRiskMedium,
				RequiresApproval:     true,
			},
			{
				ID:                   "a",
				Type:                 domain.ActionRightsizeWorkloadCPU,
				Provider:             domain.CloudProviderAWS,
				Cluster:              "production",
				Workload:             "payments/api",
				CurrentValue:         2000,
				DesiredValue:         750,
				MonthlySavingsUSD:    100,
				AnnualizedSavingsUSD: 1200,
				Risk:                 domain.ActionRiskLow,
				RequiresApproval:     true,
			},
		},
	}

	result, err := BuildDryRunExecution(plan)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Mode != ExecutionModeDryRun {
		t.Fatalf("expected dry-run mode, got %q", result.Mode)
	}
	if len(result.Operations) != 2 {
		t.Fatalf("expected two operations, got %d", len(result.Operations))
	}
	if result.Operations[0].ActionID != "a" || result.Operations[1].ActionID != "b" {
		t.Fatalf("expected deterministic action ordering, got %q then %q", result.Operations[0].ActionID, result.Operations[1].ActionID)
	}
}

func TestBuildDryRunExecution_ShouldRejectUnapprovedPlan(t *testing.T) {
	plan := domain.ActionPlan{
		ID:                        "plan-1",
		Provider:                  domain.CloudProviderAWS,
		Cluster:                   "production",
		Status:                    domain.ActionPlanApproved,
		TotalMonthlySavingsUSD:    100,
		TotalAnnualizedSavingsUSD: 1200,
		RequiresApproval:          true,
		Actions: []domain.Action{{
			ID:                   "a",
			Type:                 domain.ActionReduceNodeGroup,
			Provider:             domain.CloudProviderAWS,
			Cluster:              "production",
			CurrentValue:         4,
			DesiredValue:         2,
			MonthlySavingsUSD:    100,
			AnnualizedSavingsUSD: 1200,
			Risk:                 domain.ActionRiskMedium,
			RequiresApproval:     true,
		}},
	}

	if _, err := BuildDryRunExecution(plan); err == nil {
		t.Fatal("expected non-ready plan to be rejected")
	}
}
