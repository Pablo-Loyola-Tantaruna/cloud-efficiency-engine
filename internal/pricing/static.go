package pricing

import (
	"context"

	"cloud-efficiency-engine/internal/domain"
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

func (p *StaticProvider) GetPricingWithContext(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (ResourcePricing, error) {

	return p.GetPricing(
		ctx,
	)
}
