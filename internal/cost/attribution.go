package cost

import (
	"fmt"
	"math"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type ClusterCapacity struct {
	CPUCapacityMillicores int64 `json:"cpuCapacityMillicores"`

	MemoryCapacityBytes int64 `json:"memoryCapacityBytes"`

	NodeCount int64 `json:"nodeCount"`

	MonthlyCostUSD float64 `json:"monthlyCostUsd"`

	PricingSource PricingSource `json:"pricingSource,omitempty"`

	NodeGroups []NodeGroupCapacity `json:"nodeGroups,omitempty"`
}

type NodeGroupCapacity struct {
	Name string `json:"name"`

	MachineType string `json:"machineType,omitempty"`

	CPUCapacityMillicores int64 `json:"cpuCapacityMillicores"`

	MemoryCapacityBytes int64 `json:"memoryCapacityBytes"`

	NodeCount int64 `json:"nodeCount"`

	MonthlyCostUSD float64 `json:"monthlyCostUsd"`

	HourlyCostUSD float64 `json:"hourlyCostUsd,omitempty"`

	PricingSource PricingSource `json:"pricingSource,omitempty"`
}

type AllocationWeights struct {
	CPU float64

	Memory float64
}

type WorkloadCostAllocation struct {
	Workload string `json:"workload"`

	Namespace string `json:"namespace"`

	CPURequestMillicores int64 `json:"cpuRequestMillicores"`

	MemoryRequestBytes int64 `json:"memoryRequestBytes"`

	CPUAllocationPercentage float64 `json:"cpuAllocationPercentage"`

	MemoryAllocationPercentage float64 `json:"memoryAllocationPercentage"`

	AllocatedCostUSD float64 `json:"allocatedCostUsd"`

	MonthlyCostUSD float64 `json:"monthlyCostUsd"`
}

type AttributionReport struct {
	GeneratedAt time.Time `json:"generatedAt"`

	Cluster ClusterCapacity `json:"cluster"`

	Workloads []WorkloadCostAllocation `json:"workloads"`

	AllocatedCostUSD float64 `json:"allocatedCostUsd"`

	UnallocatedCostUSD float64 `json:"unallocatedCostUsd"`
}

type CostAttributor struct {
	weights AllocationWeights
}

func NewCostAttributor() *CostAttributor {

	return &CostAttributor{
		weights: AllocationWeights{
			CPU:    0.5,
			Memory: 0.5,
		},
	}
}

func NewCostAttributorWithWeights(
	weights AllocationWeights,
) (*CostAttributor, error) {

	if weights.CPU < 0 ||
		weights.Memory < 0 {

		return nil,
			fmt.Errorf(
				"allocation weights must not be negative",
			)
	}

	if math.Abs(
		(weights.CPU+
			weights.Memory)-
			1,
	) > 0.000001 {

		return nil,
			fmt.Errorf(
				"allocation weights must sum to 1",
			)
	}

	return &CostAttributor{
		weights: weights,
	}, nil
}

func (a *CostAttributor) Attribute(
	workloads []domain.WorkloadMetrics,
	cluster ClusterCapacity,
) (AttributionReport, error) {

	if cluster.CPUCapacityMillicores <= 0 {

		return AttributionReport{},
			fmt.Errorf(
				"cluster CPU capacity must be greater than zero",
			)
	}

	if cluster.MemoryCapacityBytes <= 0 {

		return AttributionReport{},
			fmt.Errorf(
				"cluster memory capacity must be greater than zero",
			)
	}

	if cluster.MonthlyCostUSD < 0 {

		return AttributionReport{},
			fmt.Errorf(
				"cluster monthly cost must not be negative",
			)
	}

	report :=
		AttributionReport{
			GeneratedAt: time.Now().UTC(),

			Cluster: cluster,

			Workloads: make(
				[]WorkloadCostAllocation,
				0,
				len(workloads),
			),
		}

	for _, workload := range workloads {

		cpuPercentage :=
			allocationPercentage(
				workload.CPURequestMillicores,
				cluster.CPUCapacityMillicores,
			)

		memoryPercentage :=
			allocationPercentage(
				workload.MemoryRequestBytes,
				cluster.MemoryCapacityBytes,
			)

		cpuCost :=
			cluster.MonthlyCostUSD *
				(cpuPercentage / 100)

		memoryCost :=
			cluster.MonthlyCostUSD *
				(memoryPercentage / 100)

		allocatedCost :=
			a.weights.CPU*cpuCost +
				a.weights.Memory*memoryCost

		allocation :=
			WorkloadCostAllocation{
				Workload: workloadKey(
					workload,
				),

				Namespace: workload.Namespace,

				CPURequestMillicores: workload.CPURequestMillicores,

				MemoryRequestBytes: workload.MemoryRequestBytes,

				CPUAllocationPercentage: round(
					cpuPercentage,
				),

				MemoryAllocationPercentage: round(
					memoryPercentage,
				),

				AllocatedCostUSD: round(
					allocatedCost,
				),

				MonthlyCostUSD: round(
					allocatedCost,
				),
			}

		report.Workloads =
			append(
				report.Workloads,
				allocation,
			)

		report.AllocatedCostUSD +=
			allocatedCost
	}

	report.AllocatedCostUSD =
		round(
			report.AllocatedCostUSD,
		)

	report.UnallocatedCostUSD =
		round(
			math.Max(
				0,
				cluster.MonthlyCostUSD-
					report.AllocatedCostUSD,
			),
		)

	return report, nil
}

func allocationPercentage(
	value int64,
	capacity int64,
) float64 {

	if value <= 0 ||
		capacity <= 0 {

		return 0
	}

	return float64(value) /
		float64(capacity) *
		100
}
