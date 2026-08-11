package optimizer

import (
	"cloud-efficiency-engine/internal/domain"
)

func ToCPURecommendation(
	workload domain.WorkloadMetrics,
	recommendation *ResourceRecommendation,
) *domain.Recommendation {

	if recommendation == nil {
		return nil
	}

	return &domain.Recommendation{
		Rule:        "CPU_HISTORICAL_OPTIMIZATION",
		Workload:    workload.Namespace + "/" + workload.Name,
		Description: recommendation.Reason,
		Severity: determineSeverity(
			recommendation.ReductionPercentage,
		),
		Confidence: toDomainConfidence(
			recommendation.Confidence,
		),

		CurrentCPURequestMillicores: recommendation.CurrentRequest,

		SuggestedCPURequestMillicores: recommendation.Recommended,
	}
}

func ToMemoryRecommendation(
	workload domain.WorkloadMetrics,
	recommendation *ResourceRecommendation,
) *domain.Recommendation {

	if recommendation == nil {
		return nil
	}

	return &domain.Recommendation{
		Rule:        "MEMORY_HISTORICAL_OPTIMIZATION",
		Workload:    workload.Namespace + "/" + workload.Name,
		Description: recommendation.Reason,
		Severity: determineSeverity(
			recommendation.ReductionPercentage,
		),
		Confidence: toDomainConfidence(
			recommendation.Confidence,
		),

		CurrentMemoryRequestBytes: recommendation.CurrentRequest,

		SuggestedMemoryRequestBytes: recommendation.Recommended,
	}
}

func toDomainConfidence(
	confidence string,
) domain.Confidence {

	switch confidence {

	case "HIGH":
		return domain.ConfidenceHigh

	case "MEDIUM":
		return domain.ConfidenceMedium

	default:
		return domain.ConfidenceLow
	}
}

func determineSeverity(
	reductionPercentage float64,
) domain.Severity {

	switch {
	case reductionPercentage > 60:
		return domain.SeverityCritical

	case reductionPercentage >= 30:
		return domain.SeverityWarning

	default:
		return domain.SeverityInfo
	}
}
