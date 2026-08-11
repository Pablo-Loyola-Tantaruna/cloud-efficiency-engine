package pricing

import "context"

type ResourcePricing struct {
	CPUPerCoreHour  float64
	MemoryPerGBHour float64
}

type Provider interface {
	GetPricing(
		ctx context.Context,
	) (ResourcePricing, error)
}
