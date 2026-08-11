package cost

import "context"

type PricingProvider interface {
	GetPricing(ctx context.Context) (Pricing, error)
}
