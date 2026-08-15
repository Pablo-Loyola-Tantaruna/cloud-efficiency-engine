package kubernetes

import (
	"context"

	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/pricing"
)

type PricingSource struct {
	provider pricing.Provider
}

func NewPricingSource(
	cpuPerCoreHour float64,
	memoryPerGBHour float64,
) *PricingSource {

	return &PricingSource{
		provider: pricing.NewStaticProvider(
			pricing.ResourcePricing{
				CPUPerCoreHour: cpuPerCoreHour,

				MemoryPerGBHour: memoryPerGBHour,
			},
		),
	}
}

func (s *PricingSource) GetPricing(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (pricing.ResourcePricing, error) {

	return s.provider.GetPricing(
		ctx,
	)
}
