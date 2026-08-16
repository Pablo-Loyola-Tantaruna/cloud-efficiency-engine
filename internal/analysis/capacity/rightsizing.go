package capacity

import (
	"fmt"
	"math"

	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

const (
	ClusterRightsizingRule            = "CLUSTER_NODE_RIGHTSIZING"
	ClusterRightsizingSafetyMargin    = 0.20
	ClusterRightsizingMinReductionPct = 20.0
	ClusterRightsizingConfidence      = domain.ConfidenceMedium
)

func BuildClusterRightsizingRecommendation(clusterName string, cluster cost.ClusterCapacity, workloads []WorkloadAnalysisView) (*domain.Recommendation, error) {
	if cluster.NodeCount <= 0 {
		return nil, nil
	}
	if cluster.CPUCapacityMillicores <= 0 || cluster.MemoryCapacityBytes <= 0 {
		return nil, fmt.Errorf("cluster capacity must include positive CPU and memory")
	}
	if cluster.MonthlyCostUSD < 0 {
		return nil, fmt.Errorf("cluster monthly cost must not be negative")
	}

	perNodeCPU := float64(cluster.CPUCapacityMillicores) / float64(cluster.NodeCount)
	perNodeMemory := float64(cluster.MemoryCapacityBytes) / float64(cluster.NodeCount)
	var demandedCPU int64
	var demandedMemory int64
	for _, workload := range workloads {
		cpu, memory := effectiveRequests(workload)
		if cpu > 0 {
			demandedCPU += cpu
		}
		if memory > 0 {
			demandedMemory += memory
		}
	}
	if demandedCPU <= 0 || demandedMemory <= 0 {
		return nil, nil
	}

	requiredCPU := int64(math.Ceil((float64(demandedCPU) * (1 + ClusterRightsizingSafetyMargin)) / perNodeCPU))
	requiredMemory := int64(math.Ceil((float64(demandedMemory) * (1 + ClusterRightsizingSafetyMargin)) / perNodeMemory))
	recommendedNodes := maxInt64(1, requiredCPU, requiredMemory)
	if recommendedNodes >= cluster.NodeCount {
		return nil, nil
	}

	reductionPercentage := float64(cluster.NodeCount-recommendedNodes) / float64(cluster.NodeCount) * 100
	if reductionPercentage < ClusterRightsizingMinReductionPct {
		return nil, nil
	}

	monthlySavings := cluster.MonthlyCostUSD * float64(cluster.NodeCount-recommendedNodes) / float64(cluster.NodeCount)
	recommendation := &domain.Recommendation{
		Rule:                 ClusterRightsizingRule,
		Workload:             clusterName,
		Description:          fmt.Sprintf("Cluster capacity can be normalized from %d nodes to %d nodes based on aggregated workload requests with a %.0f%% safety margin. The estimate uses average node capacity, so mixed node pools should be reviewed before applying.", cluster.NodeCount, recommendedNodes, ClusterRightsizingSafetyMargin*100),
		Severity:             severityForReduction(reductionPercentage),
		Confidence:           ClusterRightsizingConfidence,
		Priority:             priorityForReduction(reductionPercentage),
		Actionable:           cluster.MonthlyCostUSD > 0,
		SafetyScore:          domain.SafetyScoreForConfidence(ClusterRightsizingConfidence),
		SavingsSource:        domain.SavingsSourceEstimated,
		MonthlySavingsUSD:    roundValue(monthlySavings),
		AnnualizedSavingsUSD: roundValue(monthlySavings * 12),
		SavingsPercentage:    roundValue(reductionPercentage),
		CurrentNodeCount:     cluster.NodeCount,
		SuggestedNodeCount:   recommendedNodes,
	}
	if recommendation.MonthlySavingsUSD <= 0 {
		recommendation.Actionable = false
		recommendation.Priority = domain.RecommendationPriorityLow
	}
	return recommendation, nil
}

type WorkloadAnalysisView struct {
	CurrentCPURequestMillicores int64
	CurrentMemoryRequestBytes   int64
	Recommendations             []domain.Recommendation
}

func effectiveRequests(workload WorkloadAnalysisView) (int64, int64) {
	cpu := workload.CurrentCPURequestMillicores
	memory := workload.CurrentMemoryRequestBytes
	for _, recommendation := range workload.Recommendations {
		if recommendation.SuggestedCPURequestMillicores > 0 && (cpu <= 0 || recommendation.SuggestedCPURequestMillicores < cpu) {
			cpu = recommendation.SuggestedCPURequestMillicores
		}
		if recommendation.SuggestedMemoryRequestBytes > 0 && (memory <= 0 || recommendation.SuggestedMemoryRequestBytes < memory) {
			memory = recommendation.SuggestedMemoryRequestBytes
		}
	}
	return cpu, memory
}
