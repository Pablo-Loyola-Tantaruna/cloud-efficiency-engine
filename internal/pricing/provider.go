package pricing

import (
	"context"

	"cloud-efficiency-engine/internal/domain"
)

type ResourcePricing struct {
	CPUPerCoreHour  float64
	MemoryPerGBHour float64
}

type Provider interface {
	GetPricing(
		ctx context.Context,
	) (ResourcePricing, error)
}

type ContextAwareProvider interface {
	GetPricingWithContext(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
	) (ResourcePricing, error)
}
