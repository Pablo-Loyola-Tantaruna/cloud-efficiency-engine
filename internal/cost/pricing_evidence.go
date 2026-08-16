package cost

type PricingSource string

const (
	PricingSourceEstimated      PricingSource = "ESTIMATED"
	PricingSourceProviderPriced PricingSource = "PROVIDER_PRICED"
	PricingSourceActual         PricingSource = "ACTUAL"
)
