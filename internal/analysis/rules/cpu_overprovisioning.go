package rules

import (
	"fmt"
	"math"

	"cloud-efficiency-engine/internal/domain"
)

const defaultCPUThreshold = 0.30

type CPUOverprovisioningRule struct {
	Threshold float64
}

func NewCPUOverprovisioningRule() CPUOverprovisioningRule {
	return CPUOverprovisioningRule{
		Threshold: defaultCPUThreshold,
	}
}

func (r CPUOverprovisioningRule) Evaluate(
	workload domain.WorkloadMetrics,
) *domain.Recommendation {

	if workload.CPURequestMillicores <= 0 {
		return nil
	}

	utilization := float64(workload.CPUUsageMillicores) /
		float64(workload.CPURequestMillicores)

	if utilization >= r.Threshold {
		return nil
	}

	suggestedRequest := int64(
		math.Ceil(
			float64(workload.CPUUsageMillicores) / r.Threshold,
		),
	)

	return &domain.Recommendation{
		Rule: "CPU_OVERPROVISIONING",
		Workload: fmt.Sprintf(
			"%s/%s",
			workload.Namespace,
			workload.Name,
		),
		Description: fmt.Sprintf(
			"CPU utilization is %.1f%% of the requested CPU. "+
				"The workload may be overprovisioned.",
			utilization*100,
		),
		Severity:                      domain.SeverityWarning,
		Confidence:                    domain.ConfidenceHigh,
		CurrentCPURequestMillicores:   workload.CPURequestMillicores,
		SuggestedCPURequestMillicores: suggestedRequest,
	}
}
