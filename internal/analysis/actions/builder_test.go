package actions

import (
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func TestFromRecommendation_NodeGroup(t *testing.T) {
	recommendation := domain.Recommendation{
		Rule:                 "NODE_GROUP_RIGHTSIZING",
		Workload:             "workers",
		CurrentNodeCount:     8,
		SuggestedNodeCount:   6,
		MonthlySavingsUSD:    200,
		AnnualizedSavingsUSD: 2400,
		Priority:             domain.RecommendationPriorityHigh,
		Actionable:           true,
	}

	action, err := FromRecommendation(recommendation, domain.CloudProviderAWS, "prod")
	if err != nil {
		t.Fatal(err)
	}
	if action.Type != domain.ActionReduceNodeGroup {
		t.Fatalf("unexpected type: %s", action.Type)
	}
	if action.NodeGroup != "workers" {
		t.Fatalf("unexpected node group: %s", action.NodeGroup)
	}
	if action.CurrentValue != 8 || action.DesiredValue != 6 {
		t.Fatalf("unexpected values: %d -> %d", action.CurrentValue, action.DesiredValue)
	}
	if !action.RequiresApproval {
		t.Fatal("expected explicit approval")
	}
	if err := action.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestFromRecommendation_CPU(t *testing.T) {
	recommendation := domain.Recommendation{
		Rule:                          "CPU_HISTORICAL_OPTIMIZATION",
		Workload:                      "payments/payments-api",
		CurrentCPURequestMillicores:   2000,
		SuggestedCPURequestMillicores: 750,
		MonthlySavingsUSD:             50,
		AnnualizedSavingsUSD:          600,
		Priority:                      domain.RecommendationPriorityMedium,
		Actionable:                    true,
	}

	action, err := FromRecommendation(recommendation, domain.CloudProviderAzure, "aks-prod")
	if err != nil {
		t.Fatal(err)
	}
	if action.Type != domain.ActionRightsizeWorkloadCPU {
		t.Fatalf("unexpected type: %s", action.Type)
	}
	if action.Workload != recommendation.Workload {
		t.Fatalf("unexpected workload: %s", action.Workload)
	}
	if action.CurrentValue != 2000 || action.DesiredValue != 750 {
		t.Fatalf("unexpected values: %d -> %d", action.CurrentValue, action.DesiredValue)
	}
}

func TestFromRecommendation_ShouldRejectNonActionable(t *testing.T) {
	recommendation := domain.Recommendation{Rule: "NODE_GROUP_RIGHTSIZING", Workload: "workers"}
	if _, err := FromRecommendation(recommendation, domain.CloudProviderGCP, "prod"); err == nil {
		t.Fatal("expected non-actionable recommendation to be rejected")
	}
}

func TestFromRecommendation_ShouldProduceDeterministicID(t *testing.T) {
	recommendation := domain.Recommendation{
		Rule:                 "NODE_GROUP_RIGHTSIZING",
		Workload:             "workers",
		CurrentNodeCount:     4,
		SuggestedNodeCount:   2,
		MonthlySavingsUSD:    100,
		AnnualizedSavingsUSD: 1200,
		Priority:             domain.RecommendationPriorityHigh,
		Actionable:           true,
	}

	first, err := FromRecommendation(recommendation, domain.CloudProviderGCP, "prod")
	if err != nil {
		t.Fatal(err)
	}
	second, err := FromRecommendation(recommendation, domain.CloudProviderGCP, "prod")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected deterministic IDs, got %q and %q", first.ID, second.ID)
	}
}
