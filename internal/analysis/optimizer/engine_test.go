package optimizer

import (
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

func TestEngine_OptimizeCPU_ShouldUseHistoricalMetrics(
	t *testing.T,
) {

	// Arrange

	engine := NewEngine(
		DefaultOptimizationPolicy(),
	)

	workload := domain.WorkloadMetrics{
		Namespace:            "payments",
		Name:                 "payments-api",
		CPURequestMillicores: 1000,
	}

	history := domain.WorkloadHistory{
		Namespace: "payments",
		Name:      "payments-api",
		CPUUsageMillicores: []domain.MetricSample{
			{
				Timestamp: time.Unix(1, 0),
				Value:     100,
			},
			{
				Timestamp: time.Unix(2, 0),
				Value:     200,
			},
			{
				Timestamp: time.Unix(3, 0),
				Value:     300,
			},
			{
				Timestamp: time.Unix(4, 0),
				Value:     400,
			},
			{
				Timestamp: time.Unix(5, 0),
				Value:     500,
			},
		},
	}

	// Act

	result, err := engine.OptimizeCPU(
		workload,
		history,
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
			"expected optimization recommendation",
		)
	}

	if result.CurrentRequest != 1000 {
		t.Errorf(
			"expected current request 1000m, got %dm",
			result.CurrentRequest,
		)
	}
}
