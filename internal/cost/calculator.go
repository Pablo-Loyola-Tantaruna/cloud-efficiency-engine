package cost

import (
	"math"

	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/pricing"
)

type CostEstimate struct {
	CurrentMonthlyCostUSD   float64 `json:"currentMonthlyCostUsd"`
	OptimizedMonthlyCostUSD float64 `json:"optimizedMonthlyCostUsd"`
	PotentialSavingsUSD     float64 `json:"potentialSavingsUsd"`
	SavingsPercentage       float64 `json:"savingsPercentage"`
	AnnualizedSavingsUSD    float64 `json:"annualizedSavingsUsd"`
}

type Calculator struct {
	hoursPerMonth float64
}

func NewCalculator(
	hoursPerMonth float64,
) *Calculator {

	return &Calculator{
		hoursPerMonth: hoursPerMonth,
	}
}

func (c *Calculator) Estimate(
	workload domain.WorkloadMetrics,
	recommendations []domain.Recommendation,
	prices pricing.ResourcePricing,
) CostEstimate {

	current :=
		c.workloadCost(
			workload.CPURequestMillicores,
			workload.MemoryRequestBytes,
			workload.Replicas,
			prices,
		)

	optimizedCPU :=
		workload.CPURequestMillicores

	optimizedMemory :=
		workload.MemoryRequestBytes

	for _, recommendation := range recommendations {

		if recommendation.Workload !=
			workloadKey(workload) {

			continue
		}

		if recommendation.
			SuggestedCPURequestMillicores > 0 {

			optimizedCPU =
				recommendation.
					SuggestedCPURequestMillicores
		}

		if recommendation.
			SuggestedMemoryRequestBytes > 0 {

			optimizedMemory =
				recommendation.
					SuggestedMemoryRequestBytes
		}
	}

	optimized :=
		c.workloadCost(
			optimizedCPU,
			optimizedMemory,
			workload.Replicas,
			prices,
		)

	savings :=
		current - optimized

	if savings < 0 {
		savings = 0
	}

	percentage := 0.0

	if current > 0 {

		percentage =
			savings /
				current *
				100
	}

	return CostEstimate{
		CurrentMonthlyCostUSD: round(current),

		OptimizedMonthlyCostUSD: round(optimized),

		PotentialSavingsUSD: round(savings),

		SavingsPercentage: round(percentage),

		AnnualizedSavingsUSD: round(savings * 12),
	}
}

func (c *Calculator) workloadCost(
	cpuMillicores int64,
	memoryBytes int64,
	replicas int,
	prices pricing.ResourcePricing,
) float64 {

	return c.cpuCost(
		cpuMillicores,
		replicas,
		prices.CPUPerCoreHour,
	) + c.memoryCost(
		memoryBytes,
		replicas,
		prices.MemoryPerGBHour,
	)
}

func (c *Calculator) cpuCost(
	millicores int64,
	replicas int,
	pricePerCoreHour float64,
) float64 {

	cores :=
		float64(millicores) / 1000

	return cores *
		float64(replicas) *
		pricePerCoreHour *
		c.hoursPerMonth
}

func (c *Calculator) memoryCost(
	bytes int64,
	replicas int,
	pricePerGBHour float64,
) float64 {

	const bytesPerGB = 1024 * 1024 * 1024

	gb :=
		float64(bytes) /
			bytesPerGB

	return gb *
		float64(replicas) *
		pricePerGBHour *
		c.hoursPerMonth
}

func workloadKey(
	workload domain.WorkloadMetrics,
) string {

	return workload.Namespace +
		"/" +
		workload.Name
}

func round(
	value float64,
) float64 {

	return math.Round(
		value*100,
	) / 100
}
