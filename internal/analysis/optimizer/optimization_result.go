package optimizer

import "cloud-efficiency-engine/internal/domain"

type OptimizationResult struct {
	CPURecommendation    *domain.Recommendation
	MemoryRecommendation *domain.Recommendation
}
