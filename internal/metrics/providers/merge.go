package providers

import (
	"fmt"
	"strings"

	"cloud-efficiency-engine/internal/domain"
)

type workloadMergeData struct {
	metrics            domain.WorkloadMetrics
	nodeGroupSeen      bool
	nodeGroupAmbiguous bool
}

func mergeMetrics(cpuRequests []prometheusResult, cpuUsage []prometheusResult, memoryRequests []prometheusResult, memoryUsage []prometheusResult, replicas []prometheusResult) ([]domain.WorkloadMetrics, error) {
	workloads := make(map[string]*workloadMergeData)

	getWorkload := func(metric map[string]string) *workloadMergeData {
		namespace := metric["namespace"]
		name := metric["workload"]
		key := namespace + "/" + name
		if workloads[key] == nil {
			workloads[key] = &workloadMergeData{metrics: domain.WorkloadMetrics{Namespace: namespace, Name: name}}
		}
		applyNodeGroupLabel(workloads[key], metric)
		return workloads[key]
	}

	for _, item := range cpuRequests {
		value, err := parsePrometheusValue(item.Value)
		if err != nil {
			return nil, fmt.Errorf("parse CPU request for workload: %w", err)
		}
		getWorkload(item.Metric).metrics.CPURequestMillicores = int64(value)
	}
	for _, item := range cpuUsage {
		value, err := parsePrometheusValue(item.Value)
		if err != nil {
			return nil, fmt.Errorf("parse CPU usage for workload: %w", err)
		}
		getWorkload(item.Metric).metrics.CPUUsageMillicores = int64(value)
	}
	for _, item := range memoryRequests {
		value, err := parsePrometheusValue(item.Value)
		if err != nil {
			return nil, fmt.Errorf("parse memory request for workload: %w", err)
		}
		getWorkload(item.Metric).metrics.MemoryRequestBytes = int64(value)
	}
	for _, item := range memoryUsage {
		value, err := parsePrometheusValue(item.Value)
		if err != nil {
			return nil, fmt.Errorf("parse memory usage for workload: %w", err)
		}
		getWorkload(item.Metric).metrics.MemoryUsageBytes = int64(value)
	}
	for _, item := range replicas {
		value, err := parsePrometheusValue(item.Value)
		if err != nil {
			return nil, fmt.Errorf("parse replicas for workload: %w", err)
		}
		workload := getWorkload(item.Metric)
		workload.metrics.Replicas = int(value)
		workload.metrics.Type = domain.WorkloadDeployment
	}

	result := make([]domain.WorkloadMetrics, 0, len(workloads))
	for _, item := range workloads {
		if item.nodeGroupAmbiguous {
			item.metrics.NodeGroup = ""
		}
		result = append(result, item.metrics)
	}
	return result, nil
}

func applyNodeGroupLabel(workload *workloadMergeData, metric map[string]string) {
	if workload == nil || workload.nodeGroupAmbiguous {
		return
	}

	candidate := ""
	for _, key := range []string{"node_group", "nodegroup", "node_pool", "nodepool", "eks_nodegroup", "aks_nodepool", "gke_nodepool"} {
		candidate = strings.TrimSpace(metric[key])
		if candidate != "" {
			break
		}
	}
	if candidate == "" {
		return
	}

	if !workload.nodeGroupSeen {
		workload.metrics.NodeGroup = candidate
		workload.nodeGroupSeen = true
		return
	}
	if !strings.EqualFold(workload.metrics.NodeGroup, candidate) {
		workload.metrics.NodeGroup = ""
		workload.nodeGroupAmbiguous = true
	}
}
