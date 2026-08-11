package cost

import (
	"testing"

	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/pricing"
)

func TestCalculator_Estimate(
	t *testing.T,
) {

	// Arrange
	calculator :=
		NewCalculator(730)

	workload :=
		domain.WorkloadMetrics{
			Namespace: "payments",
			Name:      "payments-api",

			CPURequestMillicores: 1000,

			MemoryRequestBytes: 2 *
				1024 *
				1024 *
				1024,

			Replicas: 3,
		}

	recommendations :=
		[]domain.Recommendation{
			{
				Rule: "CPU_HISTORICAL_OPTIMIZATION",

				Workload: "payments/payments-api",

				SuggestedCPURequestMillicores: 500,
			},
			{
				Rule: "MEMORY_HISTORICAL_OPTIMIZATION",

				Workload: "payments/payments-api",

				SuggestedMemoryRequestBytes: 1024 *
					1024 *
					1024,
			},
		}

	prices :=
		pricing.ResourcePricing{
			CPUPerCoreHour:  0.04,
			MemoryPerGBHour: 0.005,
		}

	// Act
	result :=
		calculator.Estimate(
			workload,
			recommendations,
			prices,
		)

	// Assert
	if result.CurrentMonthlyCostUSD <= 0 {

		t.Fatalf(
			"expected current monthly cost greater than zero, got %.2f",
			result.CurrentMonthlyCostUSD,
		)
	}

	if result.OptimizedMonthlyCostUSD <= 0 {

		t.Fatalf(
			"expected optimized monthly cost greater than zero, got %.2f",
			result.OptimizedMonthlyCostUSD,
		)
	}

	if result.PotentialSavingsUSD <= 0 {

		t.Fatalf(
			"expected potential savings greater than zero, got %.2f",
			result.PotentialSavingsUSD,
		)
	}

	if result.SavingsPercentage <= 0 {

		t.Fatalf(
			"expected savings percentage greater than zero, got %.2f",
			result.SavingsPercentage,
		)
	}

	if result.AnnualizedSavingsUSD <=
		result.PotentialSavingsUSD {

		t.Fatalf(
			"expected annualized savings to exceed monthly savings",
		)
	}
}

func TestCalculator_Estimate_ShouldNotProduceNegativeSavings(
	t *testing.T,
) {

	// Arrange
	calculator :=
		NewCalculator(730)

	workload :=
		domain.WorkloadMetrics{
			Namespace: "orders",
			Name:      "orders-api",

			CPURequestMillicores: 500,

			MemoryRequestBytes: 1024 *
				1024 *
				1024,

			Replicas: 2,
		}

	recommendations :=
		[]domain.Recommendation{
			{
				Rule: "INVALID_RECOMMENDATION",

				Workload: "orders/orders-api",

				SuggestedCPURequestMillicores: 1000,

				SuggestedMemoryRequestBytes: 2 *
					1024 *
					1024 *
					1024,
			},
		}

	prices :=
		pricing.ResourcePricing{
			CPUPerCoreHour:  0.04,
			MemoryPerGBHour: 0.005,
		}

	// Act
	result :=
		calculator.Estimate(
			workload,
			recommendations,
			prices,
		)

	// Assert
	if result.PotentialSavingsUSD != 0 {

		t.Fatalf(
			"expected zero savings, got %.2f",
			result.PotentialSavingsUSD,
		)
	}

	if result.SavingsPercentage != 0 {

		t.Fatalf(
			"expected zero savings percentage, got %.2f",
			result.SavingsPercentage,
		)
	}
}

func TestCalculator_Estimate_ShouldIgnoreOtherWorkloads(
	t *testing.T,
) {

	// Arrange
	calculator :=
		NewCalculator(730)

	workload :=
		domain.WorkloadMetrics{
			Namespace: "payments",
			Name:      "payments-api",

			CPURequestMillicores: 1000,

			MemoryRequestBytes: 2 *
				1024 *
				1024 *
				1024,

			Replicas: 3,
		}

	recommendations :=
		[]domain.Recommendation{
			{
				Rule: "OTHER_WORKLOAD",

				Workload: "orders/orders-api",

				SuggestedCPURequestMillicores: 100,
			},
		}

	prices :=
		pricing.ResourcePricing{
			CPUPerCoreHour:  0.04,
			MemoryPerGBHour: 0.005,
		}

	// Act
	result :=
		calculator.Estimate(
			workload,
			recommendations,
			prices,
		)

	// Assert
	if result.PotentialSavingsUSD != 0 {

		t.Fatalf(
			"expected zero savings, got %.2f",
			result.PotentialSavingsUSD,
		)
	}

	if result.OptimizedMonthlyCostUSD !=
		result.CurrentMonthlyCostUSD {

		t.Fatalf(
			"expected optimized cost to equal current cost",
		)
	}
}
