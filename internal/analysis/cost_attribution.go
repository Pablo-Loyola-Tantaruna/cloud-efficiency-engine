package analysis

import "sort"

type NamespaceCostBreakdown struct {
	Namespace string `json:"namespace"`

	WorkloadCount int `json:"workloadCount"`

	OptimizableWorkloads int `json:"optimizableWorkloads"`

	CurrentMonthlyCostUSD float64 `json:"currentMonthlyCostUsd"`

	OptimizedMonthlyCostUSD float64 `json:"optimizedMonthlyCostUsd"`

	PotentialSavingsUSD float64 `json:"potentialSavingsUsd"`

	SavingsPercentage float64 `json:"savingsPercentage"`
}

func buildNamespaceCostBreakdown(
	workloads []WorkloadAnalysis,
) []NamespaceCostBreakdown {

	breakdownByNamespace :=
		make(
			map[string]*NamespaceCostBreakdown,
		)

	for _, workload := range workloads {

		namespace :=
			workload.Workload.Namespace

		if namespace == "" {
			namespace = "unknown"
		}

		breakdown :=
			breakdownByNamespace[namespace]

		if breakdown == nil {

			breakdown =
				&NamespaceCostBreakdown{
					Namespace: namespace,
				}

			breakdownByNamespace[namespace] = breakdown
		}

		breakdown.WorkloadCount++

		if len(
			workload.Recommendations,
		) > 0 {

			breakdown.
				OptimizableWorkloads++
		}

		if workload.Cost == nil {
			continue
		}

		breakdown.
			CurrentMonthlyCostUSD +=
			workload.Cost.
				CurrentMonthlyCostUSD

		breakdown.
			OptimizedMonthlyCostUSD +=
			workload.Cost.
				OptimizedMonthlyCostUSD

		breakdown.
			PotentialSavingsUSD +=
			workload.Cost.
				PotentialSavingsUSD
	}

	result :=
		make(
			[]NamespaceCostBreakdown,
			0,
			len(breakdownByNamespace),
		)

	for _, breakdown := range breakdownByNamespace {

		if breakdown.
			CurrentMonthlyCostUSD > 0 {

			breakdown.
				SavingsPercentage =
				breakdown.
					PotentialSavingsUSD /
					breakdown.
						CurrentMonthlyCostUSD *
					100
		}

		breakdown.
			CurrentMonthlyCostUSD =
			roundCost(
				breakdown.
					CurrentMonthlyCostUSD,
			)

		breakdown.
			OptimizedMonthlyCostUSD =
			roundCost(
				breakdown.
					OptimizedMonthlyCostUSD,
			)

		breakdown.
			PotentialSavingsUSD =
			roundCost(
				breakdown.
					PotentialSavingsUSD,
			)

		breakdown.
			SavingsPercentage =
			roundCost(
				breakdown.
					SavingsPercentage,
			)

		result =
			append(
				result,
				*breakdown,
			)
	}

	sort.Slice(
		result,
		func(i, j int) bool {

			if result[i].
				PotentialSavingsUSD ==
				result[j].
					PotentialSavingsUSD {

				return result[i].
					Namespace <
					result[j].
						Namespace
			}

			return result[i].
				PotentialSavingsUSD >
				result[j].
					PotentialSavingsUSD
		},
	)

	return result
}

func roundCost(
	value float64,
) float64 {

	return float64(
		int64(
			value*100+0.5,
		),
	) / 100
}
