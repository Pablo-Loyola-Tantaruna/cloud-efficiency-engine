package cost

import (
	"math"

	"cloud-efficiency-engine/internal/domain"
)

type Pricing struct {
	CPUPerCoreHour  float64
	MemoryPerGBHour float64
	HoursPerMonth   float64
}

type CostEstimate struct {
	CurrentMonthlyCostUSD   float64 `json:"currentMonthlyCostUsd"`
	OptimizedMonthlyCostUSD float64 `json:"optimizedMonthlyCostUsd"`
	PotentialSavingsUSD     float64 `json:"potentialSavingsUsd"`
	SavingsPercentage       float64 `json:"savingsPercentage"`
	AnnualizedSavingsUSD    float64 `json:"annualizedSavingsUsd"`
}

type Calculator struct {
	pricing Pricing
}

func NewCalculator(pricing Pricing) *Calculator {
	return &Calculator{
		pricing: pricing,
	}
}

func (c *Calculator) Estimate(
	workload domain.WorkloadMetrics,
	recommendations []domain.Recommendation,
) CostEstimate {

	current := c.workloadCost(
		workload.CPURequestMillicores,
		workload.MemoryRequestBytes,
		workload.Replicas,
	)

	optimizedCPU := workload.CPURequestMillicores
	optimizedMemory := workload.MemoryRequestBytes

	for _, recommendation := range recommendations {

		if recommendation.Workload != workloadKey(workload) {
			continue
		}

		if recommendation.SuggestedCPURequestMillicores > 0 {
			optimizedCPU =
				recommendation.SuggestedCPURequestMillicores
		}

		if recommendation.SuggestedMemoryRequestBytes > 0 {
			optimizedMemory =
				recommendation.SuggestedMemoryRequestBytes
		}
	}

	optimized := c.workloadCost(
		optimizedCPU,
		optimizedMemory,
		workload.Replicas,
	)

	savings := current - optimized

	if savings < 0 {
		savings = 0
	}

	percentage := 0.0

	if current > 0 {
		percentage = savings / current * 100
	}

	return CostEstimate{
		CurrentMonthlyCostUSD:   round(current),
		OptimizedMonthlyCostUSD: round(optimized),
		PotentialSavingsUSD:     round(savings),
		SavingsPercentage:       round(percentage),
		AnnualizedSavingsUSD:    round(savings * 12),
	}
}

func (c *Calculator) workloadCost(
	cpuMillicores int64,
	memoryBytes int64,
	replicas int,
) float64 {

	return c.cpuCost(
		cpuMillicores,
		replicas,
	) + c.memoryCost(
		memoryBytes,
		replicas,
	)
}

func (c *Calculator) cpuCost(
	millicores int64,
	replicas int,
) float64 {

	cores := float64(millicores) / 1000

	return cores *
		float64(replicas) *
		c.pricing.CPUPerCoreHour *
		c.pricing.HoursPerMonth
}

func (c *Calculator) memoryCost(
	bytes int64,
	replicas int,
) float64 {

	const bytesPerGB = 1024 * 1024 * 1024

	gb := float64(bytes) / bytesPerGB

	return gb *
		float64(replicas) *
		c.pricing.MemoryPerGBHour *
		c.pricing.HoursPerMonth
}

func workloadKey(
	workload domain.WorkloadMetrics,
) string {

	return workload.Namespace + "/" + workload.Name
}

func round(value float64) float64 {
	return math.Round(value*100) / 100
}
