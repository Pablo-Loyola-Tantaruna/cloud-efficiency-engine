package cost

import (
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func TestCalculator_Estimate_ShouldCalculateAnnualizedSavings(t *testing.T) {
	// Arrange
	calculator := NewCalculator(
		Pricing{
			CPUPerCoreHour:  0.04,
			MemoryPerGBHour: 0.005,
			HoursPerMonth:   730,
		},
	)

	workload := domain.WorkloadMetrics{
		Namespace:            "payments",
		Name:                 "payments-api",
		Type:                 domain.WorkloadDeployment,
		Replicas:             3,
		CPURequestMillicores: 1000,
		CPUUsageMillicores:   180,
		MemoryRequestBytes:   2 * 1024 * 1024 * 1024,
		MemoryUsageBytes:     640 * 1024 * 1024,
	}

	recommendations := []domain.Recommendation{
		{
			Rule:                          "CPU_OVERPROVISIONING",
			Workload:                      "payments/payments-api",
			CurrentCPURequestMillicores:   1000,
			SuggestedCPURequestMillicores: 600,
		},
		{
			Rule:                        "MEMORY_OVERPROVISIONING",
			Workload:                    "payments/payments-api",
			CurrentMemoryRequestBytes:   2 * 1024 * 1024 * 1024,
			SuggestedMemoryRequestBytes: 1 * 1024 * 1024 * 1024,
		},
	}

	// Act
	result := calculator.Estimate(
		workload,
		recommendations,
	)

	// Assert
	if result.CurrentMonthlyCostUSD <= 0 {
		t.Fatalf(
			"expected current monthly cost > 0, got %f",
			result.CurrentMonthlyCostUSD,
		)
	}

	if result.PotentialSavingsUSD <= 0 {
		t.Fatalf(
			"expected potential savings > 0, got %f",
			result.PotentialSavingsUSD,
		)
	}

	expectedAnnualSavings := result.PotentialSavingsUSD * 12

	if result.AnnualizedSavingsUSD != expectedAnnualSavings {
		t.Fatalf(
			"expected annualized savings %f, got %f",
			expectedAnnualSavings,
			result.AnnualizedSavingsUSD,
		)
	}
}
