package pricing

import (
	"context"
)

type StaticProvider struct {
	pricing ResourcePricing
}

func NewStaticProvider(
	pricing ResourcePricing,
) *StaticProvider {

	return &StaticProvider{
		pricing: pricing,
	}
}

func (p *StaticProvider) GetPricing(
	ctx context.Context,
) (ResourcePricing, error) {

	return p.pricing, nil
}
