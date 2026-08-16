package capacity

import (
	"testing"

	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

func TestBuildNodeGroupRightsizingRecommendations_ShouldRecommendReduction(t *testing.T) {
	groups := []cost.NodeGroupCapacity{
		{
			Name:                  "general",
			CPUCapacityMillicores: 8000,
			MemoryCapacityBytes:   16 * 1024 * 1024 * 1024,
			NodeCount:             4,
			MonthlyCostUSD:        400,
		},
	}

	workloads := []NodeGroupWorkloadView{
		{
			GroupName: "general",
			Workloads: []WorkloadAnalysisView{
				{
					CurrentCPURequestMillicores: 1500,
					CurrentMemoryRequestBytes:   3 * 1024 * 1024 * 1024,
				},
			},
		},
	}

	result, err := BuildNodeGroupRightsizingRecommendations(groups, workloads)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 1 {
		t.Fatalf("expected one recommendation, got %d", len(result))
	}

	recommendation := result[0]
	if recommendation.Rule != NodeGroupRightsizingRule {
		t.Fatalf("unexpected rule: %s", recommendation.Rule)
	}
	if recommendation.CurrentNodeCount != 4 || recommendation.SuggestedNodeCount != 1 {
		t.Fatalf("unexpected node counts: %d -> %d", recommendation.CurrentNodeCount, recommendation.SuggestedNodeCount)
	}
	if recommendation.MonthlySavingsUSD != 300 {
		t.Fatalf("expected monthly savings 300, got %.2f", recommendation.MonthlySavingsUSD)
	}
	if recommendation.AnnualizedSavingsUSD != 3600 {
		t.Fatalf("expected annualized savings 3600, got %.2f", recommendation.AnnualizedSavingsUSD)
	}
	if recommendation.Confidence != domain.ConfidenceMedium {
		t.Fatalf("expected medium confidence, got %s", recommendation.Confidence)
	}
	if recommendation.SafetyScore != 70 {
		t.Fatalf("expected safety score 70, got %d", recommendation.SafetyScore)
	}
	if recommendation.SavingsSource != domain.SavingsSourceEstimated {
		t.Fatalf("expected estimated savings source, got %s", recommendation.SavingsSource)
	}
	if !recommendation.Actionable {
		t.Fatal("expected recommendation to be actionable")
	}
}

func TestBuildNodeGroupRightsizingRecommendations_ShouldUseProviderHourlyPricing(t *testing.T) {
	groups := []cost.NodeGroupCapacity{
		{
			Name:                  "priced",
			CPUCapacityMillicores: 8000,
			MemoryCapacityBytes:   16 * 1024 * 1024 * 1024,
			NodeCount:             4,
			HourlyCostUSD:         0.10,
			PricingSource:         cost.PricingSourceProviderPriced,
		},
	}

	workloads := []NodeGroupWorkloadView{
		{
			GroupName: "priced",
			Workloads: []WorkloadAnalysisView{
				{
					CurrentCPURequestMillicores: 1500,
					CurrentMemoryRequestBytes:   3 * 1024 * 1024 * 1024,
				},
			},
		},
	}

	result, err := BuildNodeGroupRightsizingRecommendations(groups, workloads)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected one recommendation, got %d", len(result))
	}

	recommendation := result[0]
	if recommendation.MonthlySavingsUSD <= 0 {
		t.Fatal("expected savings from provider hourly price")
	}
	if recommendation.Confidence != domain.ConfidenceHigh {
		t.Fatalf("expected high confidence, got %s", recommendation.Confidence)
	}
	if recommendation.SafetyScore != 90 {
		t.Fatalf("expected safety score 90, got %d", recommendation.SafetyScore)
	}
	if recommendation.SavingsSource != domain.SavingsSourceProviderPriced {
		t.Fatalf("expected provider priced source, got %s", recommendation.SavingsSource)
	}
}

func TestBuildNodeGroupRightsizingRecommendations_ShouldIgnoreGroupsWithoutWorkloads(t *testing.T) {
	groups := []cost.NodeGroupCapacity{
		{
			Name:                  "general",
			CPUCapacityMillicores: 4000,
			MemoryCapacityBytes:   8 * 1024 * 1024 * 1024,
			NodeCount:             2,
			MonthlyCostUSD:        100,
		},
	}

	result, err := BuildNodeGroupRightsizingRecommendations(groups, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 0 {
		t.Fatalf("expected no recommendations, got %d", len(result))
	}
}

func TestBuildNodeGroupRightsizingRecommendations_ShouldRejectInvalidCapacity(t *testing.T) {
	groups := []cost.NodeGroupCapacity{
		{
			Name:                  "broken",
			CPUCapacityMillicores: 0,
			MemoryCapacityBytes:   8 * 1024 * 1024 * 1024,
			NodeCount:             2,
			MonthlyCostUSD:        100,
		},
	}

	_, err := BuildNodeGroupRightsizingRecommendations(groups, nil)
	if err == nil {
		t.Fatal("expected validation error")
	}
}
