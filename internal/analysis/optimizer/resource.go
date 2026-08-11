package optimizer

import (
	"cloud-efficiency-engine/internal/analysis/statistics"
	"fmt"
	"math"
)

type ResourceRecommendation struct {
	CurrentRequest int64
	Recommended    int64

	PercentileValue float64
	Percentile      float64

	SafetyMargin float64

	ReductionPercentage float64

	Confidence string
	Samples    int

	Reason string
}

func selectPercentile(
	stats statistics.ResourceStatistics,
	percentile float64,
) (float64, error) {

	switch percentile {

	case 50:
		return stats.P50, nil

	case 90:
		return stats.P90, nil

	case 95:
		return stats.P95, nil

	case 99:
		return stats.P99, nil

	default:
		return 0, fmt.Errorf(
			"unsupported percentile %.2f",
			percentile,
		)
	}
}

func roundUp(
	value float64,
	granularity int64,
) int64 {

	if granularity <= 0 {
		return int64(math.Ceil(value))
	}

	rounded := math.Ceil(
		value/float64(granularity),
	) * float64(granularity)

	return int64(rounded)
}

func reductionPercentage(
	current int64,
	recommended int64,
) float64 {

	if current <= 0 {
		return 0
	}

	reduction := float64(current-recommended) /
		float64(current)

	return reduction * 100
}

func confidenceForSamples(
	samples int,
	policy OptimizationPolicy,
) string {

	if samples >= policy.MinimumSamplesForHighConfidence {
		return "HIGH"
	}

	if samples >= policy.MinimumSamplesForMediumConfidence {
		return "MEDIUM"
	}

	return "LOW"
}

func RecommendCPU(
	currentRequest int64,
	stats statistics.ResourceStatistics,
	policy OptimizationPolicy,
) (*ResourceRecommendation, error) {

	if currentRequest <= 0 {
		return nil, fmt.Errorf(
			"current CPU request must be greater than zero",
		)
	}

	percentileValue, err := selectPercentile(
		stats,
		policy.CPUPercentile,
	)

	if err != nil {
		return nil, err
	}

	target := percentileValue *
		(1 + policy.CPUSafetyMargin)

	recommended := roundUp(
		target,
		policy.CPUGranularityMillicores,
	)

	if recommended < policy.MinCPURequestMillicores {
		recommended =
			policy.MinCPURequestMillicores
	}

	if recommended >= currentRequest {
		return nil, nil
	}

	reduction := reductionPercentage(
		currentRequest,
		recommended,
	)

	if reduction < policy.MinReductionPercentage {
		return nil, nil
	}

	return &ResourceRecommendation{
		CurrentRequest: currentRequest,
		Recommended:    recommended,

		PercentileValue: percentileValue,
		Percentile:      policy.CPUPercentile,

		SafetyMargin: policy.CPUSafetyMargin,

		ReductionPercentage: reduction,

		Confidence: confidenceForSamples(
			stats.Samples,
			policy,
		),

		Samples: stats.Samples,

		Reason: buildCPUReason(
			policy.CPUPercentile,
			percentileValue,
			policy.CPUSafetyMargin,
		),
	}, nil
}

func buildCPUReason(
	percentile float64,
	percentileValue float64,
	safetyMargin float64,
) string {

	return fmt.Sprintf(
		"CPU request is above P%.0f historical usage "+
			"with a %.0f%% safety margin. "+
			"Observed percentile: %.0fm.",
		percentile,
		safetyMargin*100,
		percentileValue,
	)
}

func RecommendMemory(
	currentRequest int64,
	stats statistics.ResourceStatistics,
	policy OptimizationPolicy,
) (*ResourceRecommendation, error) {

	if currentRequest <= 0 {
		return nil, fmt.Errorf(
			"current memory request must be greater than zero",
		)
	}

	percentileValue, err := selectPercentile(
		stats,
		policy.MemoryPercentile,
	)

	if err != nil {
		return nil, err
	}

	target := percentileValue *
		(1 + policy.MemorySafetyMargin)

	recommended := roundUp(
		target,
		policy.MemoryGranularityBytes,
	)

	if recommended < policy.MinMemoryRequestBytes {
		recommended =
			policy.MinMemoryRequestBytes
	}

	if recommended >= currentRequest {
		return nil, nil
	}

	reduction := reductionPercentage(
		currentRequest,
		recommended,
	)

	if reduction < policy.MinReductionPercentage {
		return nil, nil
	}

	return &ResourceRecommendation{
		CurrentRequest: currentRequest,
		Recommended:    recommended,

		PercentileValue: percentileValue,
		Percentile:      policy.MemoryPercentile,

		SafetyMargin: policy.MemorySafetyMargin,

		ReductionPercentage: reduction,

		Confidence: confidenceForSamples(
			stats.Samples,
			policy,
		),

		Samples: stats.Samples,

		Reason: buildMemoryReason(
			policy.MemoryPercentile,
			percentileValue,
			policy.MemorySafetyMargin,
		),
	}, nil
}

func buildMemoryReason(
	percentile float64,
	percentileValue float64,
	safetyMargin float64,
) string {

	return fmt.Sprintf(
		"Memory request is above P%.0f historical usage "+
			"with a %.0f%% safety margin. "+
			"Observed percentile: %.0f bytes.",
		percentile,
		safetyMargin*100,
		percentileValue,
	)
}
