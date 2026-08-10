package providers

import (
	"fmt"

	"cloud-efficiency-engine/internal/domain"
)

func mergeMetrics(
	cpuRequests []prometheusResult,
	cpuUsage []prometheusResult,
	memoryRequests []prometheusResult,
	memoryUsage []prometheusResult,
) ([]domain.WorkloadMetrics, error) {

	type workloadData struct {
		metrics domain.WorkloadMetrics
	}

	workloads := make(map[string]*workloadData)

	getWorkload := func(
		metric map[string]string,
	) *workloadData {

		namespace := metric["namespace"]
		name := metric["workload"]

		key := namespace + "/" + name

		if workloads[key] == nil {
			workloads[key] = &workloadData{
				metrics: domain.WorkloadMetrics{
					Namespace: namespace,
					Name:      name,
				},
			}
		}

		return workloads[key]
	}

	for _, item := range cpuRequests {

		value, err := parsePrometheusValue(item.Value)

		if err != nil {
			return nil, fmt.Errorf(
				"parse CPU request for workload: %w",
				err,
			)
		}

		getWorkload(item.Metric).
			metrics.
			CPURequestMillicores = int64(value)
	}

	for _, item := range cpuUsage {

		value, err := parsePrometheusValue(item.Value)

		if err != nil {
			return nil, fmt.Errorf(
				"parse CPU usage for workload: %w",
				err,
			)
		}

		getWorkload(item.Metric).
			metrics.
			CPUUsageMillicores = int64(value)
	}

	for _, item := range memoryRequests {

		value, err := parsePrometheusValue(item.Value)

		if err != nil {
			return nil, fmt.Errorf(
				"parse memory request for workload: %w",
				err,
			)
		}

		getWorkload(item.Metric).
			metrics.
			MemoryRequestBytes = int64(value)
	}

	for _, item := range memoryUsage {

		value, err := parsePrometheusValue(item.Value)

		if err != nil {
			return nil, fmt.Errorf(
				"parse memory usage for workload: %w",
				err,
			)
		}

		getWorkload(item.Metric).
			metrics.
			MemoryUsageBytes = int64(value)
	}

	result := make(
		[]domain.WorkloadMetrics,
		0,
		len(workloads),
	)

	for _, item := range workloads {
		result = append(
			result,
			item.metrics,
		)
	}

	return result, nil
}
