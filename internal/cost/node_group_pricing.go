package cost

import "cloud-efficiency-engine/internal/pricing"

const DefaultPricingHoursPerMonth = 730.0

// EnrichNodeGroupPricing converts provider-normalized CPU and memory pricing
// into a monthly node-group estimate. The result is explicitly marked
// PROVIDER_PRICED because the normalized rates originate from the cloud
// provider pricing source, while preserving the fact that the estimate is
// not actual billing.
func EnrichNodeGroupPricing(
	groups []NodeGroupCapacity,
	resourcePricing pricing.ResourcePricing,
	hoursPerMonth float64,
) []NodeGroupCapacity {
	if len(groups) == 0 ||
		resourcePricing.CPUPerCoreHour <= 0 ||
		resourcePricing.MemoryPerGBHour <= 0 {
		return groups
	}
	if hoursPerMonth <= 0 {
		hoursPerMonth = DefaultPricingHoursPerMonth
	}

	result := make([]NodeGroupCapacity, len(groups))
	copy(result, groups)

	for index := range result {
		group := &result[index]
		cpuCores := float64(group.CPUCapacityMillicores) / 1000
		memoryGB := float64(group.MemoryCapacityBytes) / (1024 * 1024 * 1024)
		hourlyCost :=
			cpuCores*resourcePricing.CPUPerCoreHour +
				memoryGB*resourcePricing.MemoryPerGBHour
		if hourlyCost <= 0 {
			continue
		}

		group.HourlyCostUSD = roundNodeGroupPricing(group.HourlyCostUSD + hourlyCost)
		group.MonthlyCostUSD = roundNodeGroupPricing(hourlyCost * hoursPerMonth)
		group.PricingSource = PricingSourceProviderPriced
	}

	return result
}

func roundNodeGroupPricing(value float64) float64 {
	return float64(int64(value*100+0.5)) / 100
}
