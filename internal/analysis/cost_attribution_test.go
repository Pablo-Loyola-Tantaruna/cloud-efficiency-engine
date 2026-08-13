package analysis

import (
	"testing"

	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

func TestBuildNamespaceCostBreakdown_ShouldAggregateWorkloads(
	t *testing.T,
) {

	// Arrange

	workloads :=
		[]WorkloadAnalysis{
			{
				Workload: domain.WorkloadMetrics{
					Namespace: "payments",
					Name:      "payments-api",
				},

				Recommendations: []domain.Recommendation{
					{
						Rule: "cpu-overprovisioning",
					},
				},

				Cost: &cost.CostEstimate{
					CurrentMonthlyCostUSD:   1000,
					OptimizedMonthlyCostUSD: 700,
					PotentialSavingsUSD:     300,
				},
			},
			{
				Workload: domain.WorkloadMetrics{
					Namespace: "payments",
					Name:      "payments-worker",
				},

				Cost: &cost.CostEstimate{
					CurrentMonthlyCostUSD:   500,
					OptimizedMonthlyCostUSD: 450,
					PotentialSavingsUSD:     50,
				},
			},
			{
				Workload: domain.WorkloadMetrics{
					Namespace: "orders",
					Name:      "orders-api",
				},

				Recommendations: []domain.Recommendation{
					{
						Rule: "memory-overprovisioning",
					},
				},

				Cost: &cost.CostEstimate{
					CurrentMonthlyCostUSD:   800,
					OptimizedMonthlyCostUSD: 600,
					PotentialSavingsUSD:     200,
				},
			},
		}

	// Act

	result :=
		buildNamespaceCostBreakdown(
			workloads,
		)

	// Assert

	if len(result) != 2 {

		t.Fatalf(
			"expected 2 namespaces, got %d",
			len(result),
		)
	}

	if result[0].Namespace !=
		"payments" {

		t.Fatalf(
			"expected payments as first namespace, got %s",
			result[0].Namespace,
		)
	}

	if result[0].WorkloadCount != 2 {

		t.Fatalf(
			"expected 2 payments workloads, got %d",
			result[0].WorkloadCount,
		)
	}

	if result[0].OptimizableWorkloads != 1 {

		t.Fatalf(
			"expected 1 optimizable payments workload, got %d",
			result[0].OptimizableWorkloads,
		)
	}

	if result[0].CurrentMonthlyCostUSD !=
		1500 {

		t.Fatalf(
			"expected payments current cost 1500, got %.2f",
			result[0].CurrentMonthlyCostUSD,
		)
	}

	if result[0].OptimizedMonthlyCostUSD !=
		1150 {

		t.Fatalf(
			"expected payments optimized cost 1150, got %.2f",
			result[0].OptimizedMonthlyCostUSD,
		)
	}

	if result[0].PotentialSavingsUSD !=
		350 {

		t.Fatalf(
			"expected payments savings 350, got %.2f",
			result[0].PotentialSavingsUSD,
		)
	}

	expectedSavingsPercentage :=
		23.33

	if result[0].SavingsPercentage !=
		expectedSavingsPercentage {

		t.Fatalf(
			"expected payments savings percentage %.2f, got %.2f",
			expectedSavingsPercentage,
			result[0].SavingsPercentage,
		)
	}
}

func TestBuildNamespaceCostBreakdown_ShouldSortByPotentialSavings(
	t *testing.T,
) {

	// Arrange

	workloads :=
		[]WorkloadAnalysis{
			{
				Workload: domain.WorkloadMetrics{
					Namespace: "small",
					Name:      "small-api",
				},

				Cost: &cost.CostEstimate{
					CurrentMonthlyCostUSD: 100,
					PotentialSavingsUSD:   10,
				},
			},
			{
				Workload: domain.WorkloadMetrics{
					Namespace: "large",
					Name:      "large-api",
				},

				Cost: &cost.CostEstimate{
					CurrentMonthlyCostUSD: 2000,
					PotentialSavingsUSD:   500,
				},
			},
			{
				Workload: domain.WorkloadMetrics{
					Namespace: "medium",
					Name:      "medium-api",
				},

				Cost: &cost.CostEstimate{
					CurrentMonthlyCostUSD: 1000,
					PotentialSavingsUSD:   100,
				},
			},
		}

	// Act

	result :=
		buildNamespaceCostBreakdown(
			workloads,
		)

	// Assert

	expectedOrder :=
		[]string{
			"large",
			"medium",
			"small",
		}

	for index, expected := range expectedOrder {

		if result[index].Namespace !=
			expected {

			t.Fatalf(
				"expected namespace %s at position %d, got %s",
				expected,
				index,
				result[index].Namespace,
			)
		}
	}
}

func TestBuildNamespaceCostBreakdown_ShouldHandleUnknownNamespace(
	t *testing.T,
) {

	// Arrange

	workloads :=
		[]WorkloadAnalysis{
			{
				Workload: domain.WorkloadMetrics{
					Name: "unknown-api",
				},

				Cost: &cost.CostEstimate{
					CurrentMonthlyCostUSD: 100,
					PotentialSavingsUSD:   20,
				},
			},
		}

	// Act

	result :=
		buildNamespaceCostBreakdown(
			workloads,
		)

	// Assert

	if len(result) != 1 {

		t.Fatalf(
			"expected 1 namespace, got %d",
			len(result),
		)
	}

	if result[0].Namespace !=
		"unknown" {

		t.Fatalf(
			"expected unknown namespace, got %s",
			result[0].Namespace,
		)
	}
}
