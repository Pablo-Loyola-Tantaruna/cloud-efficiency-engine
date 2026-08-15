package billing

type CostAttribution struct {
	Provider string `json:"provider"`

	Service string `json:"service"`

	Resource string `json:"resource,omitempty"`

	Namespace string `json:"namespace,omitempty"`

	Workload string `json:"workload,omitempty"`

	AmountUSD float64 `json:"amountUsd"`

	AllocationMethod string `json:"allocationMethod"`

	Confidence float64 `json:"confidence"`
}

type AttributionReport struct {
	TotalCostUSD float64 `json:"totalCostUsd"`

	AttributedCostUSD float64 `json:"attributedCostUsd"`

	UnallocatedCostUSD float64 `json:"unallocatedCostUsd"`

	AttributionPercentage float64 `json:"attributionPercentage"`

	Items []CostAttribution `json:"items"`
}
