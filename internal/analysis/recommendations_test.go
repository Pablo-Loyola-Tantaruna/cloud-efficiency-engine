package analysis

import (
	"encoding/json"
	"testing"

	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

func TestBuildTopRecommendations_ShouldRankAndLimit(t *testing.T) {
	workloads := []WorkloadAnalysis{
		{
			Recommendations: []domain.Recommendation{
				{
					Rule:                 "CPU_A",
					Workload:             "ns/a",
					AnnualizedSavingsUSD: 100,
					Priority:             domain.RecommendationPriorityHigh,
					Actionable:           true,
				},
				{
					Rule:                 "CPU_B",
					Workload:             "ns/b",
					AnnualizedSavingsUSD: 300,
					Priority:             domain.RecommendationPriorityMedium,
					Actionable:           true,
				},
				{
					Rule:                 "CPU_C",
					Workload:             "ns/c",
					AnnualizedSavingsUSD: 200,
					Priority:             domain.RecommendationPriorityCritical,
					Actionable:           true,
				},
			},
		},
	}

	for i := 0; i < 11; i++ {
		workloads = append(workloads, WorkloadAnalysis{
			Recommendations: []domain.Recommendation{
				{
					Rule:                 "EXTRA",
					Workload:             "ns/extra" + string(rune('a'+i)),
					AnnualizedSavingsUSD: float64(i + 1),
					Priority:             domain.RecommendationPriorityLow,
					Actionable:           true,
				},
			},
		})
	}

	result := buildTopRecommendations(workloads, "", nil)

	if len(result) != TopRecommendationsLimit {
		t.Fatalf("expected %d recommendations, got %d", TopRecommendationsLimit, len(result))
	}

	if result[0].Rule != "CPU_B" || result[1].Rule != "CPU_C" || result[2].Rule != "CPU_A" {
		t.Fatalf("unexpected ranking: %#v", result[:3])
	}

	for _, recommendation := range result {
		if !recommendation.Actionable {
			t.Fatal("expected only actionable recommendations")
		}
		if recommendation.AnnualizedSavingsUSD <= 0 {
			t.Fatal("expected positive annualized savings")
		}
	}
}

func TestBuildTopRecommendations_ShouldIgnoreNonActionable(t *testing.T) {
	result := buildTopRecommendations([]WorkloadAnalysis{{
		Recommendations: []domain.Recommendation{
			{
				Rule:                 "LOW_CONFIDENCE",
				Workload:             "ns/api",
				AnnualizedSavingsUSD: 999,
				Actionable:           false,
			},
		},
	}}, "", nil)

	if len(result) != 0 {
		t.Fatalf("expected no top recommendations, got %d", len(result))
	}
}

func TestBuildTopRecommendations_ShouldIncludeClusterRightsizing(t *testing.T) {
	reportCost := &cost.AttributionReport{
		Cluster: cost.ClusterCapacity{
			NodeCount:             10,
			CPUCapacityMillicores: 20000,
			MemoryCapacityBytes:   40 * 1024 * 1024 * 1024,
			MonthlyCostUSD:        1000,
		},
	}

	workloads := []WorkloadAnalysis{
		{
			Workload: domain.WorkloadMetrics{
				Namespace:            "payments",
				Name:                 "api",
				CPURequestMillicores: 10000,
				MemoryRequestBytes:   20 * 1024 * 1024 * 1024,
			},
		},
	}

	result := buildTopRecommendations(workloads, "prod-aks", reportCost)
	if len(result) != 1 {
		t.Fatalf("expected one cluster recommendation, got %d", len(result))
	}
	if result[0].Rule != "CLUSTER_NODE_RIGHTSIZING" {
		t.Fatalf("unexpected rule: %s", result[0].Rule)
	}
	if result[0].SuggestedNodeCount != 6 || result[0].CurrentNodeCount != 10 {
		t.Fatalf("unexpected node counts: %#v", result[0])
	}
	if result[0].AnnualizedSavingsUSD <= 0 {
		t.Fatalf("expected positive annualized savings: %#v", result[0])
	}
}

func TestAnalysisReport_JSON_ShouldIncludeComputedTopRecommendations(t *testing.T) {
	report := AnalysisReport{
		Context: domain.AnalysisContext{
			ClusterName: "prod-aks",
		},
		Workloads: []WorkloadAnalysis{{
			Recommendations: []domain.Recommendation{
				{
					Rule:                 "CPU_HISTORICAL_OPTIMIZATION",
					Workload:             "payments/api",
					AnnualizedSavingsUSD: 480,
					Actionable:           true,
					Priority:             domain.RecommendationPriorityHigh,
				},
			},
		}},
	}

	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}

	var decoded struct {
		TopRecommendations []domain.Recommendation `json:"topRecommendations"`
	}

	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}

	if len(decoded.TopRecommendations) != 1 {
		t.Fatalf("expected one top recommendation, got %d", len(decoded.TopRecommendations))
	}
	if decoded.TopRecommendations[0].Rule != "CPU_HISTORICAL_OPTIMIZATION" {
		t.Fatalf("unexpected top recommendation: %#v", decoded.TopRecommendations[0])
	}
}
