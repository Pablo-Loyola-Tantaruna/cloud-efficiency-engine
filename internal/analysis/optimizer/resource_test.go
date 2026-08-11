package optimizer

import (
	"testing"

	"cloud-efficiency-engine/internal/analysis/statistics"
)

func TestRecommendCPU_ShouldRecommendLowerRequest(
	t *testing.T,
) {

	// Arrange

	policy := DefaultOptimizationPolicy()

	stats := statistics.ResourceStatistics{
		P50:     100,
		P90:     200,
		P95:     300,
		P99:     450,
		Max:     800,
		Samples: 1500,
	}

	// Act

	result, err := RecommendCPU(
		1000,
		stats,
		policy,
	)

	// Assert

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if result == nil {
		t.Fatal(
			"expected recommendation",
		)
	}

	if result.Recommended != 400 {
		t.Errorf(
			"expected recommendation 400m, got %dm",
			result.Recommended,
		)
	}

	if result.CurrentRequest != 1000 {
		t.Errorf(
			"expected current request 1000m, got %dm",
			result.CurrentRequest,
		)
	}

	if result.ReductionPercentage != 60 {
		t.Errorf(
			"expected 60%% reduction, got %.2f%%",
			result.ReductionPercentage,
		)
	}

	if result.Confidence != "HIGH" {
		t.Errorf(
			"expected HIGH confidence, got %s",
			result.Confidence,
		)
	}
}

func TestRecommendCPU_ShouldNotRecommendWhenReductionIsTooSmall(
	t *testing.T,
) {

	// Arrange

	policy := DefaultOptimizationPolicy()

	stats := statistics.ResourceStatistics{
		P50:     700,
		P90:     750,
		P95:     780,
		P99:     850,
		Max:     900,
		Samples: 1500,
	}

	// Act

	result, err := RecommendCPU(
		1000,
		stats,
		policy,
	)

	// Assert

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if result != nil {
		t.Fatal(
			"expected no recommendation for small reduction",
		)
	}
}

func TestRecommendCPU_ShouldApplySafetyMargin(
	t *testing.T,
) {

	// Arrange

	policy := DefaultOptimizationPolicy()

	stats := statistics.ResourceStatistics{
		P95:     500,
		Samples: 1500,
	}

	// Act

	result, err := RecommendCPU(
		1000,
		stats,
		policy,
	)

	// Assert

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if result == nil {
		t.Fatal(
			"expected recommendation",
		)
	}

	// 500m * 1.20 = 600m

	if result.Recommended != 600 {
		t.Errorf(
			"expected 600m, got %dm",
			result.Recommended,
		)
	}
}

func TestRecommendCPU_ShouldRespectMinimumRequest(
	t *testing.T,
) {

	// Arrange

	policy := DefaultOptimizationPolicy()

	stats := statistics.ResourceStatistics{
		P95:     10,
		Samples: 1500,
	}

	// Act

	result, err := RecommendCPU(
		1000,
		stats,
		policy,
	)

	// Assert

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if result == nil {
		t.Fatal(
			"expected recommendation",
		)
	}

	if result.Recommended != 100 {
		t.Errorf(
			"expected minimum CPU request 100m, got %dm",
			result.Recommended,
		)
	}
}
