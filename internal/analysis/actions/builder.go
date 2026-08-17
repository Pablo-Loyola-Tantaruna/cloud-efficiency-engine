package actions

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"cloud-efficiency-engine/internal/domain"
)

// FromRecommendation converts an actionable recommendation into a safe,
// auditable action plan. Execution is deliberately outside this package.
func FromRecommendation(
	recommendation domain.Recommendation,
	provider domain.CloudProvider,
	cluster string,
) (*domain.Action, error) {
	if !recommendation.Actionable {
		return nil, fmt.Errorf("recommendation %q is not actionable", recommendation.Rule)
	}
	if provider == domain.CloudProviderUnknown {
		return nil, fmt.Errorf("action provider must be known")
	}
	cluster = strings.TrimSpace(cluster)
	if cluster == "" {
		return nil, fmt.Errorf("action cluster must not be empty")
	}

	action := &domain.Action{
		ID:                   actionID(recommendation, provider, cluster),
		Provider:             provider,
		Cluster:              cluster,
		MonthlySavingsUSD:    recommendation.MonthlySavingsUSD,
		AnnualizedSavingsUSD: recommendation.AnnualizedSavingsUSD,
		Risk:                 riskForRecommendation(recommendation),
		RequiresApproval:     true,
		WorkloadType:         recommendation.WorkloadType,
		Container:            recommendation.ContainerName,
	}

	switch recommendation.Rule {
	case "NODE_GROUP_RIGHTSIZING", "CLUSTER_NODE_RIGHTSIZING":
		if recommendation.CurrentNodeCount <= 0 || recommendation.SuggestedNodeCount <= 0 {
			return nil, fmt.Errorf("node reduction recommendation has invalid node counts")
		}
		action.Type = domain.ActionReduceNodeGroup
		action.NodeGroup = recommendation.Workload
		action.CurrentValue = recommendation.CurrentNodeCount
		action.DesiredValue = recommendation.SuggestedNodeCount
	case "CPU_HISTORICAL_OPTIMIZATION", "CPU_RIGHTSIZING":
		if recommendation.CurrentCPURequestMillicores <= 0 || recommendation.SuggestedCPURequestMillicores <= 0 {
			return nil, fmt.Errorf("CPU recommendation has invalid request values")
		}
		action.Type = domain.ActionRightsizeWorkloadCPU
		action.Workload = recommendation.Workload
		action.CurrentValue = recommendation.CurrentCPURequestMillicores
		action.DesiredValue = recommendation.SuggestedCPURequestMillicores
	case "MEMORY_HISTORICAL_OPTIMIZATION", "MEMORY_RIGHTSIZING":
		if recommendation.CurrentMemoryRequestBytes <= 0 || recommendation.SuggestedMemoryRequestBytes <= 0 {
			return nil, fmt.Errorf("memory recommendation has invalid request values")
		}
		action.Type = domain.ActionRightsizeWorkloadMemory
		action.Workload = recommendation.Workload
		action.CurrentValue = recommendation.CurrentMemoryRequestBytes
		action.DesiredValue = recommendation.SuggestedMemoryRequestBytes
	default:
		return nil, fmt.Errorf("recommendation rule %q cannot be converted to an action", recommendation.Rule)
	}

	if action.Type == domain.ActionRightsizeWorkloadCPU || action.Type == domain.ActionRightsizeWorkloadMemory {
		if action.WorkloadType == domain.WorkloadJob {
			return nil, fmt.Errorf("workload rightsizing is not supported for Job resources")
		}
	}

	if err := action.Validate(); err != nil {
		return nil, err
	}
	return action, nil
}

func actionID(recommendation domain.Recommendation, provider domain.CloudProvider, cluster string) string {
	seed := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%d|%d", provider, cluster, recommendation.Rule, recommendation.Workload, recommendation.WorkloadType, recommendation.ContainerName, recommendation.CurrentNodeCount, recommendation.SuggestedNodeCount)
	sum := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(sum[:8])
}

func riskForRecommendation(recommendation domain.Recommendation) domain.ActionRisk {
	switch recommendation.Priority {
	case domain.RecommendationPriorityCritical, domain.RecommendationPriorityHigh:
		return domain.ActionRiskMedium
	case domain.RecommendationPriorityMedium:
		return domain.ActionRiskLow
	default:
		return domain.ActionRiskHigh
	}
}
