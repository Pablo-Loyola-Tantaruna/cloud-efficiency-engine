package capacity

import (
	"math"

	"cloud-efficiency-engine/internal/domain"
)

func maxInt64(values ...int64) int64 {
	if len(values) == 0 {
		return 0
	}
	result := values[0]
	for _, value := range values[1:] {
		if value > result {
			result = value
		}
	}
	return result
}

func severityForReduction(reductionPercentage float64) domain.Severity {
	switch {
	case reductionPercentage >= 50:
		return domain.SeverityCritical
	case reductionPercentage >= 30:
		return domain.SeverityWarning
	default:
		return domain.SeverityInfo
	}
}

func priorityForReduction(reductionPercentage float64) domain.RecommendationPriority {
	switch {
	case reductionPercentage >= 50:
		return domain.RecommendationPriorityCritical
	case reductionPercentage >= 30:
		return domain.RecommendationPriorityHigh
	case reductionPercentage >= 20:
		return domain.RecommendationPriorityMedium
	default:
		return domain.RecommendationPriorityLow
	}
}

func roundValue(value float64) float64 {
	return math.Round(value*100) / 100
}
