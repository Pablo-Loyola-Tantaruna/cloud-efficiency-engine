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

type Calculator struct { hoursPerMonth float64 }

func NewCalculator(hoursPerMonth float64) *Calculator { return &Calculator{hoursPerMonth: hoursPerMonth} }

func (c *Calculator) Estimate(workload domain.WorkloadMetrics, recommendations []domain.Recommendation, prices pricing.ResourcePricing) CostEstimate {
	current := c.workloadCost(workload.CPURequestMillicores, workload.MemoryRequestBytes, workload.Replicas, prices)
	optimizedCPU := workload.CPURequestMillicores
	optimizedMemory := workload.MemoryRequestBytes
	for _, recommendation := range recommendations {
		if recommendation.Workload != workloadKey(workload) { continue }
		if recommendation.SuggestedCPURequestMillicores > 0 { optimizedCPU = recommendation.SuggestedCPURequestMillicores }
		if recommendation.SuggestedMemoryRequestBytes > 0 { optimizedMemory = recommendation.SuggestedMemoryRequestBytes }
	}
	optimized := c.workloadCost(optimizedCPU, optimizedMemory, workload.Replicas, prices)
	savings := current - optimized
	if savings < 0 { savings = 0 }
	percentage := 0.0
	if current > 0 { percentage = savings / current * 100 }
	return CostEstimate{CurrentMonthlyCostUSD: round(current), OptimizedMonthlyCostUSD: round(optimized), PotentialSavingsUSD: round(savings), SavingsPercentage: round(percentage), AnnualizedSavingsUSD: round(savings * 12)}
}

func (c *Calculator) EnrichRecommendations(workload domain.WorkloadMetrics, recommendations []domain.Recommendation, prices pricing.ResourcePricing) []domain.Recommendation {
	for index := range recommendations {
		if recommendations[index].Workload != workloadKey(workload) { continue }
		enrichRecommendation(&recommendations[index], workload, prices, c.hoursPerMonth)
	}
	return recommendations
}

func enrichRecommendation(recommendation *domain.Recommendation, workload domain.WorkloadMetrics, prices pricing.ResourcePricing, hoursPerMonth float64) {
	if recommendation == nil { return }
	current := workloadCostWithHours(workload.CPURequestMillicores, workload.MemoryRequestBytes, workload.Replicas, prices, hoursPerMonth)
	optimizedCPU := workload.CPURequestMillicores
	optimizedMemory := workload.MemoryRequestBytes
	if recommendation.SuggestedCPURequestMillicores > 0 && recommendation.SuggestedCPURequestMillicores < optimizedCPU { optimizedCPU = recommendation.SuggestedCPURequestMillicores }
	if recommendation.SuggestedMemoryRequestBytes > 0 && recommendation.SuggestedMemoryRequestBytes < optimizedMemory { optimizedMemory = recommendation.SuggestedMemoryRequestBytes }
	optimized := workloadCostWithHours(optimizedCPU, optimizedMemory, workload.Replicas, prices, hoursPerMonth)
	savings := current - optimized
	if savings <= 0 || current <= 0 {
		recommendation.MonthlySavingsUSD = 0
		recommendation.AnnualizedSavingsUSD = 0
		recommendation.SavingsPercentage = 0
		recommendation.Actionable = false
		recommendation.Priority = domain.RecommendationPriorityLow
		return
	}
	percentage := savings / current * 100
	recommendation.MonthlySavingsUSD = round(savings)
	recommendation.AnnualizedSavingsUSD = round(savings * 12)
	recommendation.SavingsPercentage = round(percentage)
	if recommendation.SafetyScore == 0 {
		recommendation.SafetyScore = domain.SafetyScoreForConfidence(recommendation.Confidence)
	}
	recommendation.Actionable = savings > 0 && percentage >= 20 && recommendation.Confidence != domain.ConfidenceLow && recommendation.SafetyScore >= 50
	if recommendation.Confidence == domain.ConfidenceLow {
		recommendation.Priority = domain.RecommendationPriorityLow
		return
	}
	recommendation.Priority = recommendationPriority(recommendation.AnnualizedSavingsUSD, recommendation.Confidence)
}

func recommendationPriority(annualSavings float64, confidence domain.Confidence) domain.RecommendationPriority {
	switch {
	case annualSavings >= 1000 && confidence == domain.ConfidenceHigh: return domain.RecommendationPriorityCritical
	case annualSavings >= 500: return domain.RecommendationPriorityHigh
	case annualSavings >= 100: return domain.RecommendationPriorityMedium
	default: return domain.RecommendationPriorityLow
	}
}

func workloadCostWithHours(cpuMillicores int64, memoryBytes int64, replicas int, prices pricing.ResourcePricing, hoursPerMonth float64) float64 {
	cpuCores := float64(cpuMillicores) / 1000
	memoryGB := float64(memoryBytes) / (1024 * 1024 * 1024)
	return cpuCores*float64(replicas)*prices.CPUPerCoreHour*hoursPerMonth + memoryGB*float64(replicas)*prices.MemoryPerGBHour*hoursPerMonth
}

func (c *Calculator) workloadCost(cpuMillicores int64, memoryBytes int64, replicas int, prices pricing.ResourcePricing) float64 {
	return c.cpuCost(cpuMillicores, replicas, prices.CPUPerCoreHour) + c.memoryCost(memoryBytes, replicas, prices.MemoryPerGBHour)
}

func (c *Calculator) cpuCost(millicores int64, replicas int, pricePerCoreHour float64) float64 {
	return float64(millicores) / 1000 * float64(replicas) * pricePerCoreHour * c.hoursPerMonth
}

func (c *Calculator) memoryCost(memoryBytes int64, replicas int, pricePerGBHour float64) float64 {
	return float64(memoryBytes) / (1024 * 1024 * 1024) * float64(replicas) * pricePerGBHour * c.hoursPerMonth
}

func round(value float64) float64 { return math.Round(value*100) / 100 }
