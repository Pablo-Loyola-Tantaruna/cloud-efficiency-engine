package aws

import (
	"context"
	"fmt"

	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/pricing"
)

type PricingSource struct {
	client PricingClient
}

func NewPricingSource(
	client PricingClient,
) *PricingSource {

	return &PricingSource{
		client: client,
	}
}

func (s *PricingSource) GetPricing(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (pricing.ResourcePricing, error) {

	if s.client == nil {
		return pricing.ResourcePricing{},
			fmt.Errorf(
				"AWS pricing client is not configured",
			)
	}

	resourcePrice, err :=
		s.client.GetPricing(
			ctx,
			analysisContext,
		)

	if err != nil {
		return pricing.ResourcePricing{}, err
	}

	return pricing.ResourcePricing{
		CPUPerCoreHour: resourcePrice.CPUPerCoreHour,

		MemoryPerGBHour: resourcePrice.MemoryPerGBHour,
	}, nil
}
