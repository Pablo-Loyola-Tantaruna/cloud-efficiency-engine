package optimizer

import (
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func TestToCPURecommendation_ShouldConvertResourceRecommendation(
	t *testing.T,
) {

	// Arrange

	workload := domain.WorkloadMetrics{
		Namespace: "payments",
		Name:      "payments-api",
	}

	resourceRecommendation := &ResourceRecommendation{
		CurrentRequest:      1000,
		Recommended:         400,
		PercentileValue:     300,
		Percentile:          95,
		SafetyMargin:        0.20,
		ReductionPercentage: 60,
		Confidence:          "HIGH",
		Samples:             1500,
		Reason:              "CPU request is above P95 historical usage.",
	}

	// Act

	result := ToCPURecommendation(
		workload,
		resourceRecommendation,
	)

	// Assert

	if result == nil {
		t.Fatal("expected recommendation")
	}

	if result.Rule != "CPU_HISTORICAL_OPTIMIZATION" {
		t.Errorf(
			"expected CPU_HISTORICAL_OPTIMIZATION, got %s",
			result.Rule,
		)
	}

	if result.Workload != "payments/payments-api" {
		t.Errorf(
			"expected payments/payments-api, got %s",
			result.Workload,
		)
	}

	if result.Severity != domain.SeverityWarning {
		t.Errorf(
			"expected WARNING, got %s",
			result.Severity,
		)
	}

	if result.Confidence != domain.ConfidenceHigh {
		t.Errorf(
			"expected HIGH confidence, got %s",
			result.Confidence,
		)
	}

	if result.CurrentCPURequestMillicores != 1000 {
		t.Errorf(
			"expected current CPU 1000m, got %d",
			result.CurrentCPURequestMillicores,
		)
	}

	if result.SuggestedCPURequestMillicores != 400 {
		t.Errorf(
			"expected suggested CPU 400m, got %d",
			result.SuggestedCPURequestMillicores,
		)
	}
}

func TestToCPURecommendation_ShouldReturnNilWhenRecommendationIsNil(
	t *testing.T,
) {

	// Arrange

	workload := domain.WorkloadMetrics{
		Namespace: "payments",
		Name:      "payments-api",
	}

	// Act

	result := ToCPURecommendation(
		workload,
		nil,
	)

	// Assert

	if result != nil {
		t.Fatal(
			"expected nil recommendation",
		)
	}
}
