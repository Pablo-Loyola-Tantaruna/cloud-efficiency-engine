package observability

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"cloud-efficiency-engine/internal/analysis"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

func TestMetricsUpdate_ShouldExportNamespaceCostMetrics(
	t *testing.T,
) {

	metrics :=
		NewMetrics()

	report :=
		&analysis.AnalysisReport{
			NamespaceBreakdown: []analysis.NamespaceCostBreakdown{
				{
					Namespace: "payments",

					WorkloadCount: 5,

					OptimizableWorkloads: 3,

					CurrentMonthlyCostUSD: 1000,

					OptimizedMonthlyCostUSD: 700,

					PotentialSavingsUSD: 300,

					SavingsPercentage: 30,
				},
			},
		}

	metrics.Update(
		report,
	)

	expected :=
		`
# HELP cee_current_monthly_cost_usd Current estimated monthly infrastructure cost in USD by namespace.
# TYPE cee_current_monthly_cost_usd gauge
cee_current_monthly_cost_usd{namespace="payments"} 1000
`

	if err :=
		testutil.GatherAndCompare(
			metrics.registry,
			strings.NewReader(
				expected,
			),
			"cee_current_monthly_cost_usd",
		); err != nil {

		t.Fatal(err)
	}
}

func TestMetricsUpdate_ShouldExportWorkloadCostMetrics(
	t *testing.T,
) {

	metrics :=
		NewMetrics()

	report :=
		&analysis.AnalysisReport{
			Workloads: []analysis.WorkloadAnalysis{
				{
					Workload: domain.WorkloadMetrics{
						Namespace: "payments",

						Name: "payments-api",
					},

					Recommendations: []domain.Recommendation{
						{
							Rule: "cpu-overprovisioning",
						},
					},

					Cost: &cost.CostEstimate{
						CurrentMonthlyCostUSD: 1000,

						OptimizedMonthlyCostUSD: 700,

						PotentialSavingsUSD: 300,

						SavingsPercentage: 30,
					},
				},
			},
		}

	metrics.Update(
		report,
	)

	expectedCost :=
		`
# HELP cee_workload_current_monthly_cost_usd Current estimated monthly infrastructure cost in USD by workload.
# TYPE cee_workload_current_monthly_cost_usd gauge
cee_workload_current_monthly_cost_usd{namespace="payments",workload="payments-api"} 1000
`

	if err :=
		testutil.GatherAndCompare(
			metrics.registry,
			strings.NewReader(
				expectedCost,
			),
			"cee_workload_current_monthly_cost_usd",
		); err != nil {

		t.Fatal(err)
	}

	expectedSavings :=
		`
# HELP cee_workload_potential_savings_usd Estimated potential monthly savings in USD by workload.
# TYPE cee_workload_potential_savings_usd gauge
cee_workload_potential_savings_usd{namespace="payments",workload="payments-api"} 300
`

	if err :=
		testutil.GatherAndCompare(
			metrics.registry,
			strings.NewReader(
				expectedSavings,
			),
			"cee_workload_potential_savings_usd",
		); err != nil {

		t.Fatal(err)
	}

	expectedOptimizable :=
		`
# HELP cee_workload_optimizable Whether a workload has one or more optimization recommendations.
# TYPE cee_workload_optimizable gauge
cee_workload_optimizable{namespace="payments",workload="payments-api"} 1
`

	if err :=
		testutil.GatherAndCompare(
			metrics.registry,
			strings.NewReader(
				expectedOptimizable,
			),
			"cee_workload_optimizable",
		); err != nil {

		t.Fatal(err)
	}
}

func TestMetricsUpdate_ShouldRemoveStaleWorkload(
	t *testing.T,
) {

	metrics :=
		NewMetrics()

	firstReport :=
		&analysis.AnalysisReport{
			Workloads: []analysis.WorkloadAnalysis{
				{
					Workload: domain.WorkloadMetrics{
						Namespace: "payments",

						Name: "payments-api",
					},

					Cost: &cost.CostEstimate{
						CurrentMonthlyCostUSD: 1000,
					},
				},
				{
					Workload: domain.WorkloadMetrics{
						Namespace: "payments",

						Name: "payments-worker",
					},

					Cost: &cost.CostEstimate{
						CurrentMonthlyCostUSD: 500,
					},
				},
			},
		}

	secondReport :=
		&analysis.AnalysisReport{
			Workloads: []analysis.WorkloadAnalysis{
				{
					Workload: domain.WorkloadMetrics{
						Namespace: "payments",

						Name: "payments-api",
					},

					Cost: &cost.CostEstimate{
						CurrentMonthlyCostUSD: 900,
					},
				},
			},
		}

	metrics.Update(
		firstReport,
	)

	metrics.Update(
		secondReport,
	)

	expected :=
		`
# HELP cee_workload_current_monthly_cost_usd Current estimated monthly infrastructure cost in USD by workload.
# TYPE cee_workload_current_monthly_cost_usd gauge
cee_workload_current_monthly_cost_usd{namespace="payments",workload="payments-api"} 900
`

	if err :=
		testutil.GatherAndCompare(
			metrics.registry,
			strings.NewReader(
				expected,
			),
			"cee_workload_current_monthly_cost_usd",
		); err != nil {

		t.Fatal(err)
	}

	stale :=
		`
# HELP cee_workload_current_monthly_cost_usd Current estimated monthly infrastructure cost in USD by workload.
# TYPE cee_workload_current_monthly_cost_usd gauge
cee_workload_current_monthly_cost_usd{namespace="payments",workload="payments-worker"} 500
`

	if err :=
		testutil.GatherAndCompare(
			metrics.registry,
			strings.NewReader(
				stale,
			),
			"cee_workload_current_monthly_cost_usd",
		); err == nil {

		t.Fatal(
			"expected payments-worker metrics to be removed",
		)
	}
}

func TestMetricsRecordSchedulerSuccess_ShouldExposeMetrics(
	t *testing.T,
) {

	metrics :=
		NewMetrics()

	metrics.RecordSchedulerSuccess(
		"payments",
		time.Now().UTC().Add(
			-2*time.Second,
		),
	)

	expected :=
		`
# HELP cee_scheduler_success_total Total number of successful scheduled analyses.
# TYPE cee_scheduler_success_total counter
cee_scheduler_success_total{namespace="payments"} 1
`

	if err :=
		testutil.GatherAndCompare(
			metrics.registry,
			strings.NewReader(
				expected,
			),
			"cee_scheduler_success_total",
		); err != nil {

		t.Fatal(err)
	}
}
