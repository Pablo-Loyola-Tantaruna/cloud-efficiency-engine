package cost

import (
	"cloud-efficiency-engine/internal/domain"
)

type CostSource string

const (
	CostSourceEstimated CostSource = "ESTIMATED"

	CostSourceAllocated CostSource = "ALLOCATED"

	CostSourceActual CostSource = "ACTUAL"
)

type CostBreakdown struct {
	Source CostSource `json:"source"`

	ComputeUSD float64 `json:"computeUsd"`

	StorageUSD float64 `json:"storageUsd"`

	NetworkUSD float64 `json:"networkUsd"`

	OtherUSD float64 `json:"otherUsd"`

	TotalUSD float64 `json:"totalUsd"`
}

func workloadKey(workload domain.WorkloadMetrics) string {
	return workload.Namespace + "/" + workload.Name
}
