package analysis

import (
	"testing"

	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

func TestBuildTopRecommendations_ShouldIncludeNodeGroupRightsizing(t *testing.T) {
	workloads := []WorkloadAnalysis{
		{
			Workload: domain.WorkloadMetrics{
				Namespace:            "payments",
				Name:                 "api",
				CPURequestMillicores: 500,
				MemoryRequestBytes:   1024 * 1024 * 1024,
				NodeGroup:            "general",
			},
			Recommendations: nil,
		},
	}

	attribution := &cost.AttributionReport{
		Cluster: cost.ClusterCapacity{
			NodeCount:             4,
			CPUCapacityMillicores: 8000,
			MemoryCapacityBytes:   16 * 1024 * 1024 * 1024,
			MonthlyCostUSD:        400,
			NodeGroups: []cost.NodeGroupCapacity{
				{
					Name:                  "general",
					NodeCount:             4,
					CPUCapacityMillicores: 8000,
					MemoryCapacityBytes:   16 * 1024 * 1024 * 1024,
				},
			},
		},
	}

	result := buildTopRecommendations(workloads, "gke-prod", attribution)
	if len(result) != 1 {
		t.Fatalf("expected one recommendation, got %d", len(result))
	}

	if result[0].Rule != "NODE_GROUP_RIGHTSIZING" {
		t.Fatalf("expected node group rule, got %q", result[0].Rule)
	}
	if result[0].Workload != "general" {
		t.Fatalf("expected node group general, got %q", result[0].Workload)
	}
	if result[0].SuggestedNodeCount >= result[0].CurrentNodeCount {
		t.Fatalf("expected a node reduction, got %d -> %d", result[0].CurrentNodeCount, result[0].SuggestedNodeCount)
	}
}

func TestBuildTopRecommendations_ShouldIgnoreAmbiguousNodeGroupPlacement(t *testing.T) {
	workloads := []WorkloadAnalysis{
		{
			Workload: domain.WorkloadMetrics{
				CPURequestMillicores: 500,
				MemoryRequestBytes:   1024 * 1024 * 1024,
				NodeGroup:            "",
			},
		},
	}

	attribution := &cost.AttributionReport{
		Cluster: cost.ClusterCapacity{
			NodeCount:             4,
			CPUCapacityMillicores: 8000,
			MemoryCapacityBytes:   16 * 1024 * 1024 * 1024,
			MonthlyCostUSD:        400,
			NodeGroups: []cost.NodeGroupCapacity{
				{
					Name:                  "general-a",
					NodeCount:             4,
					CPUCapacityMillicores: 8000,
					MemoryCapacityBytes:   16 * 1024 * 1024 * 1024,
				},
			},
		},
	}

	result := buildTopRecommendations(workloads, "cluster", attribution)
	for _, recommendation := range result {
		if recommendation.Rule == "NODE_GROUP_RIGHTSIZING" {
			t.Fatal("did not expect node group recommendation without placement")
		}
	}
}
