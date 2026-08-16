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

func TestCalculator_EnrichRecommendations_ShouldCalculateImpactAndPriority(
	t *testing.T,
) {

	// Arrange
	calculator := NewCalculator(730)

	workload := domain.WorkloadMetrics{
		Namespace:            "payments",
		Name:                 "payments-api",
		CPURequestMillicores: 2000,
		MemoryRequestBytes:   2 * 1024 * 1024 * 1024,
		Replicas:             2,
	}

	recommendations := []domain.Recommendation{
		{
			Rule:                          "CPU_HISTORICAL_OPTIMIZATION",
			Workload:                      "payments/payments-api",
			Confidence:                    domain.ConfidenceHigh,
			SuggestedCPURequestMillicores: 500,
		},
	}

	prices := pricing.ResourcePricing{
		CPUPerCoreHour:  0.04,
		MemoryPerGBHour: 0.005,
	}

	// Act
	result := calculator.EnrichRecommendations(
		workload,
		recommendations,
		prices,
	)

	// Assert
	if len(result) != 1 {
		t.Fatalf("expected one recommendation, got %d", len(result))
	}

	recommendation := result[0]
	if recommendation.MonthlySavingsUSD <= 0 {
		t.Fatalf("expected monthly savings greater than zero")
	}
	if recommendation.AnnualizedSavingsUSD <= recommendation.MonthlySavingsUSD {
		t.Fatalf("expected annualized savings to exceed monthly savings")
	}
	if recommendation.SavingsPercentage < 20 {
		t.Fatalf("expected savings percentage >= 20, got %.2f", recommendation.SavingsPercentage)
	}
	if !recommendation.Actionable {
		t.Fatal("expected recommendation to be actionable")
	}
	if recommendation.Priority != domain.RecommendationPriorityCritical {
		t.Fatalf("expected critical priority, got %q", recommendation.Priority)
	}
}

func TestCalculator_EnrichRecommendations_ShouldMarkLowConfidenceNonActionable(
	t *testing.T,
) {

	// Arrange
	calculator := NewCalculator(730)

	workload := domain.WorkloadMetrics{
		Namespace:            "orders",
		Name:                 "orders-api",
		CPURequestMillicores: 1000,
		MemoryRequestBytes:   2 * 1024 * 1024 * 1024,
		Replicas:             1,
	}

	recommendations := []domain.Recommendation{
		{
			Rule:                          "CPU_HISTORICAL_OPTIMIZATION",
			Workload:                      "orders/orders-api",
			Confidence:                    domain.ConfidenceLow,
			SuggestedCPURequestMillicores: 500,
		},
	}

	prices := pricing.ResourcePricing{
		CPUPerCoreHour:  0.04,
		MemoryPerGBHour: 0.005,
	}

	// Act
	result := calculator.EnrichRecommendations(
		workload,
		recommendations,
		prices,
	)

	// Assert
	if len(result) != 1 {
		t.Fatalf("expected one recommendation, got %d", len(result))
	}

	if result[0].Actionable {
		t.Fatal("expected low-confidence recommendation to be non-actionable")
	}
	if result[0].Priority != domain.RecommendationPriorityLow {
		t.Fatalf("expected low priority, got %q", result[0].Priority)
	}
}
