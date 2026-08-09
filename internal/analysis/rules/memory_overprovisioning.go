package rules

import (
	"fmt"
	"math"

	"cloud-efficiency-engine/internal/domain"
)

const defaultMemoryThreshold = 0.40

type MemoryOverprovisioningRule struct {
	Threshold float64
}

func NewMemoryOverprovisioningRule() MemoryOverprovisioningRule {
	return MemoryOverprovisioningRule{
		Threshold: defaultMemoryThreshold,
	}
}

func (r MemoryOverprovisioningRule) Evaluate(
	workload domain.WorkloadMetrics,
) *Recommendation {

	if workload.MemoryRequestBytes <= 0 {
		return nil
	}

	utilization := float64(workload.MemoryUsageBytes) /
		float64(workload.MemoryRequestBytes)

	if utilization >= r.Threshold {
		return nil
	}

	suggestedRequest := int64(
		math.Ceil(float64(workload.MemoryUsageBytes) / r.Threshold),
	)

	return &Recommendation{
		Rule:     "MEMORY_OVERPROVISIONING",
		Workload: fmt.Sprintf("%s/%s", workload.Namespace, workload.Name),
		Description: fmt.Sprintf(
			"Memory utilization is %.1f%% of the requested memory. "+
				"The workload may be overprovisioned.",
			utilization*100,
		),
		Severity:                    SeverityWarning,
		Confidence:                  ConfidenceHigh,
		CurrentMemoryRequestBytes:   workload.MemoryRequestBytes,
		SuggestedMemoryRequestBytes: suggestedRequest,
	}
}
