package actions

import (
	"strings"
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func TestRenderPreview_ShouldRenderPlanWithoutExecutingActions(t *testing.T) {
	plan := domain.ActionPlan{
		ID:       "plan-123",
		Provider: domain.CloudProviderAWS,
		Cluster:  "production",
		Status:   domain.ActionPlanPreview,
		Actions: []domain.Action{
			{
				ID:                   "action-2",
				Type:                 domain.ActionRightsizeWorkloadCPU,
				Provider:             domain.CloudProviderAWS,
				Cluster:              "production",
				Workload:             "payments/payments-api",
				CurrentValue:         2000,
				DesiredValue:         750,
				MonthlySavingsUSD:    83,
				AnnualizedSavingsUSD: 996,
				Risk:                 domain.ActionRiskLow,
				RequiresApproval:     true,
			},
			{
				ID:                   "action-1",
				Type:                 domain.ActionReduceNodeGroup,
				Provider:             domain.CloudProviderAWS,
				Cluster:              "production",
				NodeGroup:            "workers",
				CurrentValue:         8,
				DesiredValue:         6,
				MonthlySavingsUSD:    200,
				AnnualizedSavingsUSD: 2400,
				Risk:                 domain.ActionRiskMedium,
				RequiresApproval:     true,
			},
		},
		TotalMonthlySavingsUSD:    283,
		TotalAnnualizedSavingsUSD: 3396,
		RequiresApproval:          true,
	}

	preview, err := RenderPreview(plan)
	if err != nil {
		t.Fatal(err)
	}

	assertContains := func(value string) {
		t.Helper()
		if !strings.Contains(preview, value) {
			t.Fatalf("expected preview to contain %q, got:\n%s", value, preview)
		}
	}

	assertContains("PLAN plan-123")
	assertContains("AWS / production")
	assertContains("REDUCE_NODE_GROUP")
	assertContains("workers")
	assertContains("RIGHTSIZE_WORKLOAD_CPU")
	assertContains("payments/payments-api")
	assertContains("$283.00/month")
	assertContains("$3396.00/year")
	assertContains("STATUS: PREVIEW")
	assertContains("APPROVAL: REQUIRED")

	if strings.Index(preview, "REDUCE_NODE_GROUP") > strings.Index(preview, "RIGHTSIZE_WORKLOAD_CPU") {
		t.Fatal("expected deterministic action ordering")
	}
}

func TestRenderPreview_ShouldRejectInvalidPlan(t *testing.T) {
	_, err := RenderPreview(domain.ActionPlan{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
