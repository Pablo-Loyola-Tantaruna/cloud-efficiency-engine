package capacity

import (
	"fmt"
	"math"

	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

const (
	NodeGroupRightsizingRule            = "NODE_GROUP_RIGHTSIZING"
	NodeGroupRightsizingSafetyMargin    = 0.20
	NodeGroupRightsizingMinReductionPct = 20.0
	NodeGroupHoursPerMonth              = 730.0
)

type NodeGroupWorkloadView struct {
	GroupName string
	Workloads []WorkloadAnalysisView
}

func BuildNodeGroupRightsizingRecommendations(groups []cost.NodeGroupCapacity, workloads []NodeGroupWorkloadView) ([]domain.Recommendation, error) {
	if len(groups) == 0 {
		return nil, nil
	}

	workloadsByGroup := make(map[string][]WorkloadAnalysisView, len(workloads))
	for _, assignment := range workloads {
		if assignment.GroupName == "" {
			continue
		}
		workloadsByGroup[assignment.GroupName] = assignment.Workloads
	}

	result := make([]domain.Recommendation, 0, len(groups))
	for _, group := range groups {
		if group.Name == "" || group.NodeCount <= 0 {
			continue
		}
		if group.CPUCapacityMillicores <= 0 || group.MemoryCapacityBytes <= 0 {
			return nil, fmt.Errorf("node group %q must include positive CPU and memory capacity", group.Name)
		}

		monthlyCost := group.MonthlyCostUSD
		pricingSource := group.PricingSource
		if monthlyCost <= 0 && group.HourlyCostUSD > 0 {
			monthlyCost = group.HourlyCostUSD * float64(group.NodeCount) * NodeGroupHoursPerMonth
			if pricingSource == "" {
				pricingSource = cost.PricingSourceProviderPriced
			}
		}
		if monthlyCost < 0 {
			return nil, fmt.Errorf("node group %q monthly cost must not be negative", group.Name)
		}

		assigned := workloadsByGroup[group.Name]
		if len(assigned) == 0 {
			continue
		}

		cpuDemand, memoryDemand := aggregateEffectiveRequests(assigned)
		if cpuDemand <= 0 || memoryDemand <= 0 {
			continue
		}

		perNodeCPU := float64(group.CPUCapacityMillicores) / float64(group.NodeCount)
		perNodeMemory := float64(group.MemoryCapacityBytes) / float64(group.NodeCount)
		requiredCPU := int64(math.Ceil((float64(cpuDemand) * (1 + NodeGroupRightsizingSafetyMargin)) / perNodeCPU))
		requiredMemory := int64(math.Ceil((float64(memoryDemand) * (1 + NodeGroupRightsizingSafetyMargin)) / perNodeMemory))
		recommendedNodes := maxInt64(1, requiredCPU, requiredMemory)
		if recommendedNodes >= group.NodeCount {
			continue
		}

		reductionPercentage := float64(group.NodeCount-recommendedNodes) / float64(group.NodeCount) * 100
		if reductionPercentage < NodeGroupRightsizingMinReductionPct {
			continue
		}

		monthlySavings := monthlyCost * float64(group.NodeCount-recommendedNodes) / float64(group.NodeCount)
		confidence := domain.ConfidenceMedium
		if pricingSource == cost.PricingSourceProviderPriced || pricingSource == cost.PricingSourceActual {
			confidence = domain.ConfidenceHigh
		}

		recommendation := domain.Recommendation{
			Rule:                 NodeGroupRightsizingRule,
			Workload:             group.Name,
			Severity:             severityForReduction(reductionPercentage),
			Confidence:           confidence,
			Priority:             priorityForReduction(reductionPercentage),
			Actionable:           monthlyCost > 0,
			SafetyScore:          domain.SafetyScoreForConfidence(confidence),
			SavingsSource:        savingsSourceFromCost(pricingSource),
			CurrentNodeCount:     group.NodeCount,
			SuggestedNodeCount:   recommendedNodes,
			MonthlySavingsUSD:    roundSavings(monthlySavings),
			AnnualizedSavingsUSD: roundSavings(monthlySavings * 12),
			SavingsPercentage:    roundSavings(reductionPercentage),
			Description: fmt.Sprintf(
				"Node group %q can be reduced from %d nodes to %d nodes based on its assigned workload requests with a %.0f%% safety margin.",
				group.Name, group.NodeCount, recommendedNodes, NodeGroupRightsizingSafetyMargin*100,
			),
		}

		if recommendation.MonthlySavingsUSD <= 0 {
			recommendation.Actionable = false
			recommendation.Priority = domain.RecommendationPriorityLow
		}
		result = append(result, recommendation)
	}
	return result, nil
}

func savingsSourceFromCost(source cost.PricingSource) domain.SavingsSource {
	switch source {
	case cost.PricingSourceProviderPriced:
		return domain.SavingsSourceProviderPriced
	case cost.PricingSourceActual:
		return domain.SavingsSourceActual
	default:
		return domain.SavingsSourceEstimated
	}
}

func aggregateEffectiveRequests(workloads []WorkloadAnalysisView) (int64, int64) {
	var cpu int64
	var memory int64
	for _, workload := range workloads {
		workloadCPU, workloadMemory := effectiveRequests(workload)
		if workloadCPU > 0 {
			cpu += workloadCPU
		}
		if workloadMemory > 0 {
			memory += workloadMemory
		}
	}
	return cpu, memory
}

func roundSavings(value float64) float64 { return math.Round(value*100) / 100 }
