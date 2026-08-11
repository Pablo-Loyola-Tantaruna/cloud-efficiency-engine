package resolver

import (
	"cloud-efficiency-engine/internal/domain"
)

type ResourceType string

const (
	ResourceCPU      ResourceType = "CPU"
	ResourceMemory   ResourceType = "MEMORY"
	ResourceCombined ResourceType = "COMBINED"
	ResourceUnknown  ResourceType = "UNKNOWN"
)

type Resolver struct {
}

func NewResolver() *Resolver {
	return &Resolver{}
}

func (r *Resolver) Resolve(
	recommendations []domain.Recommendation,
) []domain.Recommendation {

	if len(recommendations) == 0 {
		return []domain.Recommendation{}
	}

	result := make(
		[]domain.Recommendation,
		0,
		len(recommendations),
	)

	indexByKey := make(
		map[string]int,
		len(recommendations),
	)

	for _, recommendation := range recommendations {

		key := recommendationKey(
			recommendation,
		)

		existingIndex, exists :=
			indexByKey[key]

		if !exists {

			indexByKey[key] = len(result)

			result = append(
				result,
				recommendation,
			)

			continue
		}

		existing :=
			result[existingIndex]

		if isBetter(
			recommendation,
			existing,
		) {

			result[existingIndex] =
				recommendation
		}
	}

	return result
}

func recommendationKey(
	recommendation domain.Recommendation,
) string {

	resource :=
		detectResource(
			recommendation,
		)

	return recommendation.Workload +
		"|" +
		string(resource)
}

func detectResource(
	recommendation domain.Recommendation,
) ResourceType {

	hasCPU :=
		recommendation.CurrentCPURequestMillicores > 0 ||
			recommendation.SuggestedCPURequestMillicores > 0

	hasMemory :=
		recommendation.CurrentMemoryRequestBytes > 0 ||
			recommendation.SuggestedMemoryRequestBytes > 0

	switch {

	case hasCPU && hasMemory:
		return ResourceCombined

	case hasCPU:
		return ResourceCPU

	case hasMemory:
		return ResourceMemory

	default:
		return ResourceUnknown
	}
}

func isBetter(
	candidate domain.Recommendation,
	current domain.Recommendation,
) bool {

	candidateSeverity :=
		severityRank(
			candidate.Severity,
		)

	currentSeverity :=
		severityRank(
			current.Severity,
		)

	if candidateSeverity != currentSeverity {
		return candidateSeverity > currentSeverity
	}

	candidateConfidence :=
		confidenceRank(
			candidate.Confidence,
		)

	currentConfidence :=
		confidenceRank(
			current.Confidence,
		)

	if candidateConfidence != currentConfidence {
		return candidateConfidence > currentConfidence
	}

	candidateReduction :=
		reductionPercentage(
			candidate,
		)

	currentReduction :=
		reductionPercentage(
			current,
		)

	if candidateReduction != currentReduction {
		return candidateReduction > currentReduction
	}

	return candidate.Rule < current.Rule
}

func severityRank(
	severity domain.Severity,
) int {

	switch severity {

	case domain.SeverityCritical:
		return 3

	case domain.SeverityWarning:
		return 2

	case domain.SeverityInfo:
		return 1

	default:
		return 0
	}
}

func confidenceRank(
	confidence domain.Confidence,
) int {

	switch confidence {

	case domain.ConfidenceHigh:
		return 3

	case domain.ConfidenceMedium:
		return 2

	case domain.ConfidenceLow:
		return 1

	default:
		return 0
	}
}

func reductionPercentage(
	recommendation domain.Recommendation,
) float64 {

	resource :=
		detectResource(
			recommendation,
		)

	switch resource {

	case ResourceCPU:

		return calculateReduction(
			float64(
				recommendation.CurrentCPURequestMillicores,
			),
			float64(
				recommendation.SuggestedCPURequestMillicores,
			),
		)

	case ResourceMemory:

		return calculateReduction(
			float64(
				recommendation.CurrentMemoryRequestBytes,
			),
			float64(
				recommendation.SuggestedMemoryRequestBytes,
			),
		)

	case ResourceCombined:

		cpuReduction :=
			calculateReduction(
				float64(
					recommendation.CurrentCPURequestMillicores,
				),
				float64(
					recommendation.SuggestedCPURequestMillicores,
				),
			)

		memoryReduction :=
			calculateReduction(
				float64(
					recommendation.CurrentMemoryRequestBytes,
				),
				float64(
					recommendation.SuggestedMemoryRequestBytes,
				),
			)

		if cpuReduction > memoryReduction {
			return cpuReduction
		}

		return memoryReduction

	default:
		return 0
	}
}

func calculateReduction(
	current float64,
	suggested float64,
) float64 {

	if current <= 0 {
		return 0
	}

	if suggested <= 0 {
		return 0
	}

	if suggested >= current {
		return 0
	}

	return ((current - suggested) / current) * 100
}
