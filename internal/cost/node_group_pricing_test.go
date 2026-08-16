package cost

import (
	"testing"

	"cloud-efficiency-engine/internal/pricing"
)

func TestEnrichNodeGroupPricing_ShouldCalculateProviderPricedCost(t *testing.T) {
	groups := []NodeGroupCapacity{
		{
			Name:                  "general",
			CPUCapacityMillicores: 4000,
			MemoryCapacityBytes:   8 * 1024 * 1024 * 1024,
			NodeCount:             2,
		},
	}

	result := EnrichNodeGroupPricing(
		groups,
		pricing.ResourcePricing{
			CPUPerCoreHour:  0.05,
			MemoryPerGBHour: 0.01,
		},
		730,
	)

	if len(result) != 1 {
		t.Fatalf("expected one group, got %d", len(result))
	}
	if result[0].HourlyCostUSD != 0.28 {
		t.Fatalf("expected hourly cost 0.28, got %.2f", result[0].HourlyCostUSD)
	}
	if result[0].MonthlyCostUSD != 204.40 {
		t.Fatalf("expected monthly cost 204.40, got %.2f", result[0].MonthlyCostUSD)
	}
	if result[0].PricingSource != PricingSourceProviderPriced {
		t.Fatalf("expected provider priced source, got %q", result[0].PricingSource)
	}
}

func TestEnrichNodeGroupPricing_ShouldPreserveGroupsWhenPricingIsUnavailable(t *testing.T) {
	groups := []NodeGroupCapacity{{Name: "general", NodeCount: 2}}

	result := EnrichNodeGroupPricing(
		groups,
		pricing.ResourcePricing{},
		730,
	)

	if result[0].MonthlyCostUSD != 0 {
		t.Fatalf("expected zero monthly cost, got %.2f", result[0].MonthlyCostUSD)
	}
	if result[0].PricingSource != "" {
		t.Fatalf("expected empty pricing source, got %q", result[0].PricingSource)
	}
}
