package analysis

import (
	"sort"

	capacityengine "cloud-efficiency-engine/internal/analysis/capacity"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

const TopRecommendationsLimit = 10

func buildTopRecommendations(
	workloads []WorkloadAnalysis,
	clusterName string,
	attribution *cost.AttributionReport,
) []domain.Recommendation {

	recommendations := make([]domain.Recommendation, 0)

	for _, workload := range workloads {
		for _, recommendation := range workload.Recommendations {
			if !recommendation.Actionable || recommendation.AnnualizedSavingsUSD <= 0 {
				continue
			}
			recommendations = append(recommendations, recommendation)
		}
	}

	if attribution != nil {
		workloadViews := make([]capacityengine.WorkloadAnalysisView, 0, len(workloads))
		for _, workload := range workloads {
			workloadViews = append(workloadViews, capacityengine.WorkloadAnalysisView{
				CurrentCPURequestMillicores: workload.Workload.CPURequestMillicores,
				CurrentMemoryRequestBytes:   workload.Workload.MemoryRequestBytes,
				Recommendations:             workload.Recommendations,
			})
		}

		if len(attribution.Cluster.NodeGroups) == 0 {
			clusterRecommendation, err := capacityengine.BuildClusterRightsizingRecommendation(
				clusterName,
				attribution.Cluster,
				workloadViews,
			)
			if err == nil && clusterRecommendation != nil &&
				clusterRecommendation.Actionable && clusterRecommendation.AnnualizedSavingsUSD > 0 {
				recommendations = append(recommendations, *clusterRecommendation)
			}
		}

		if len(attribution.Cluster.NodeGroups) > 0 {
			groups := make([]cost.NodeGroupCapacity, 0, len(attribution.Cluster.NodeGroups))
			totalNodes := int64(0)
			for _, group := range attribution.Cluster.NodeGroups {
				if group.NodeCount > 0 {
					totalNodes += group.NodeCount
				}
			}

			for _, group := range attribution.Cluster.NodeGroups {
				if group.MonthlyCostUSD <= 0 && totalNodes > 0 && attribution.Cluster.MonthlyCostUSD > 0 {
					group.MonthlyCostUSD = attribution.Cluster.MonthlyCostUSD *
						float64(group.NodeCount) / float64(totalNodes)
				}
				groups = append(groups, group)
			}

			assignments := make([]capacityengine.NodeGroupWorkloadView, 0)
			for _, workload := range workloads {
				if workload.Workload.NodeGroup == "" {
					continue
				}

				assignments = append(assignments, capacityengine.NodeGroupWorkloadView{
					GroupName: workload.Workload.NodeGroup,
					Workloads: []capacityengine.WorkloadAnalysisView{
						{
							CurrentCPURequestMillicores: workload.Workload.CPURequestMillicores,
							CurrentMemoryRequestBytes:   workload.Workload.MemoryRequestBytes,
							Recommendations:             workload.Recommendations,
						},
					},
				})
			}

			nodeGroupRecommendations, groupErr :=
				capacityengine.BuildNodeGroupRightsizingRecommendations(groups, assignments)
			if groupErr == nil {
				for _, recommendation := range nodeGroupRecommendations {
					if recommendation.Actionable && recommendation.AnnualizedSavingsUSD > 0 {
						recommendations = append(recommendations, recommendation)
					}
				}
			}
		}
	}

	sort.SliceStable(
		recommendations,
		func(i, j int) bool {
			if recommendations[i].AnnualizedSavingsUSD != recommendations[j].AnnualizedSavingsUSD {
				return recommendations[i].AnnualizedSavingsUSD > recommendations[j].AnnualizedSavingsUSD
			}
			if recommendations[i].Priority != recommendations[j].Priority {
				return recommendationPriorityRank(recommendations[i].Priority) >
					recommendationPriorityRank(recommendations[j].Priority)
			}
			if recommendations[i].Workload != recommendations[j].Workload {
				return recommendations[i].Workload < recommendations[j].Workload
			}
			return recommendations[i].Rule < recommendations[j].Rule
		},
	)

	if len(recommendations) > TopRecommendationsLimit {
		recommendations = recommendations[:TopRecommendationsLimit]
	}

	return recommendations
}

func recommendationPriorityRank(
	priority domain.RecommendationPriority,
) int {

	switch priority {
	case domain.RecommendationPriorityCritical:
		return 4
	case domain.RecommendationPriorityHigh:
		return 3
	case domain.RecommendationPriorityMedium:
		return 2
	case domain.RecommendationPriorityLow:
		return 1
	default:
		return 0
	}
}
