package cost

import (
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func TestCostAttributor_ShouldAllocateClusterCost(
	t *testing.T,
) {

	attributor :=
		NewCostAttributor()

	workloads :=
		[]domain.WorkloadMetrics{
			{
				Namespace: "payments",

				Name: "api",

				CPURequestMillicores: 1000,

				MemoryRequestBytes: 4 * 1024 * 1024 * 1024,
			},
		}

	report, err :=
		attributor.Attribute(
			workloads,
			ClusterCapacity{
				CPUCapacityMillicores: 4000,

				MemoryCapacityBytes: 16 * 1024 * 1024 * 1024,

				MonthlyCostUSD: 400,
			},
		)

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if len(report.Workloads) != 1 {

		t.Fatalf(
			"expected 1 workload, got %d",
			len(report.Workloads),
		)
	}

	allocation :=
		report.Workloads[0]

	if allocation.Workload !=
		"payments/api" {

		t.Fatalf(
			"unexpected workload: %s",
			allocation.Workload,
		)
	}

	if allocation.CPUAllocationPercentage !=
		25 {

		t.Fatalf(
			"expected CPU allocation 25%%, got %f",
			allocation.CPUAllocationPercentage,
		)
	}

	if allocation.MemoryAllocationPercentage !=
		25 {

		t.Fatalf(
			"expected memory allocation 25%%, got %f",
			allocation.MemoryAllocationPercentage,
		)
	}

	if allocation.MonthlyCostUSD !=
		100 {

		t.Fatalf(
			"expected monthly cost 100, got %f",
			allocation.MonthlyCostUSD,
		)
	}
}

func TestCostAttributor_ShouldCalculateUnallocatedCost(
	t *testing.T,
) {

	attributor :=
		NewCostAttributor()

	workloads :=
		[]domain.WorkloadMetrics{
			{
				Namespace: "payments",

				Name: "api",

				CPURequestMillicores: 500,

				MemoryRequestBytes: 2 * 1024 * 1024 * 1024,
			},
		}

	report, err :=
		attributor.Attribute(
			workloads,
			ClusterCapacity{
				CPUCapacityMillicores: 4000,

				MemoryCapacityBytes: 16 * 1024 * 1024 * 1024,

				MonthlyCostUSD: 400,
			},
		)

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if report.AllocatedCostUSD !=
		50 {

		t.Fatalf(
			"expected allocated cost 50, got %f",
			report.AllocatedCostUSD,
		)
	}

	if report.UnallocatedCostUSD !=
		350 {

		t.Fatalf(
			"expected unallocated cost 350, got %f",
			report.UnallocatedCostUSD,
		)
	}
}

func TestNewCostAttributorWithWeights_ShouldRejectInvalidWeights(
	t *testing.T,
) {

	_, err :=
		NewCostAttributorWithWeights(
			AllocationWeights{
				CPU:    0.8,
				Memory: 0.8,
			},
		)

	if err == nil {

		t.Fatal(
			"expected error",
		)
	}
}

func TestNewCostAttributorWithWeights_ShouldAcceptValidWeights(
	t *testing.T,
) {

	attributor, err :=
		NewCostAttributorWithWeights(
			AllocationWeights{
				CPU:    0.7,
				Memory: 0.3,
			},
		)

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if attributor == nil {

		t.Fatal(
			"expected attributor",
		)
	}
}

func TestCostAttributor_ShouldRejectInvalidCluster(
	t *testing.T,
) {

	attributor :=
		NewCostAttributor()

	_, err :=
		attributor.Attribute(
			nil,
			ClusterCapacity{
				CPUCapacityMillicores: 0,

				MemoryCapacityBytes: 1024,

				MonthlyCostUSD: 100,
			},
		)

	if err == nil {

		t.Fatal(
			"expected error",
		)
	}
}
