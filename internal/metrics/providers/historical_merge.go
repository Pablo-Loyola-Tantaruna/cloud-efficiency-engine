package providers

import (
	"fmt"

	"cloud-efficiency-engine/internal/domain"
)

func mergeHistoricalMetrics(
	cpuResults []prometheusRangeResult,
	memoryResults []prometheusRangeResult,
) ([]domain.WorkloadHistory, error) {

	type workloadData struct {
		history domain.WorkloadHistory
	}

	workloads := make(
		map[string]*workloadData,
	)

	getWorkload := func(
		metric map[string]string,
	) *workloadData {

		namespace := metric["namespace"]
		name := metric["workload"]

		key := namespace + "/" + name

		if workloads[key] == nil {

			workloads[key] = &workloadData{
				history: domain.WorkloadHistory{
					Namespace: namespace,
					Name:      name,
				},
			}
		}

		return workloads[key]
	}

	for _, item := range cpuResults {

		samples, err := parsePrometheusRangeValues(
			item.Values,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"parse CPU history for workload: %w",
				err,
			)
		}

		workload := getWorkload(item.Metric)

		workload.history.CPUUsageMillicores = samples
	}

	for _, item := range memoryResults {

		samples, err := parsePrometheusRangeValues(
			item.Values,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"parse memory history for workload: %w",
				err,
			)
		}

		workload := getWorkload(item.Metric)

		workload.history.MemoryUsageBytes = samples
	}

	result := make(
		[]domain.WorkloadHistory,
		0,
		len(workloads),
	)

	for _, item := range workloads {

		result = append(
			result,
			item.history,
		)
	}

	return result, nil
}
