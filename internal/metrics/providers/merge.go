package providers

import (
	"fmt"
	"strings"

	"cloud-efficiency-engine/internal/domain"
)

type workloadMergeData struct {
	metrics               domain.WorkloadMetrics
	nodeGroupSeen         bool
	nodeGroupAmbiguous    bool
	containerSeen         bool
	containerAmbiguous    bool
	workloadTypeSeen      bool
	workloadTypeAmbiguous bool
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
		applyContainerLabel(workloads[key], metric)
		applyWorkloadTypeLabel(workloads[key], metric)
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
	}

	result := make([]domain.WorkloadMetrics, 0, len(workloads))
	for _, item := range workloads {
		if item.nodeGroupAmbiguous {
			item.metrics.NodeGroup = ""
		}
		if item.containerAmbiguous {
			item.metrics.ContainerName = ""
		}
		if item.workloadTypeAmbiguous {
			item.metrics.Type = domain.WorkloadUnknown
		}
		if !item.workloadTypeSeen {
			item.metrics.Type = domain.WorkloadUnknown
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

func applyContainerLabel(workload *workloadMergeData, metric map[string]string) {
	if workload == nil || workload.containerAmbiguous {
		return
	}

	candidate := strings.TrimSpace(metric["container"])
	if candidate == "" || candidate == "POD" {
		return
	}

	if !workload.containerSeen {
		workload.metrics.ContainerName = candidate
		workload.containerSeen = true
		return
	}
	if workload.metrics.ContainerName != candidate {
		workload.metrics.ContainerName = ""
		workload.containerAmbiguous = true
	}
}

func applyWorkloadTypeLabel(workload *workloadMergeData, metric map[string]string) {
	if workload == nil || workload.workloadTypeAmbiguous {
		return
	}

	candidate := ""
	for _, key := range []string{"workload_kind", "workload_type", "kind", "owner_kind"} {
		candidate = strings.TrimSpace(metric[key])
		if candidate != "" {
			break
		}
	}
	if candidate == "" {
		return
	}

	workloadType := normalizeWorkloadType(candidate)
	if !workload.workloadTypeSeen {
		workload.metrics.Type = workloadType
		workload.workloadTypeSeen = true
		return
	}
	if workload.metrics.Type != workloadType {
		workload.metrics.Type = domain.WorkloadUnknown
		workload.workloadTypeAmbiguous = true
	}
}

func normalizeWorkloadType(value string) domain.WorkloadType {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "deployment":
		return domain.WorkloadDeployment
	case "statefulset", "stateful-set":
		return domain.WorkloadStatefulSet
	case "daemonset", "daemon-set":
		return domain.WorkloadDaemonSet
	case "job":
		return domain.WorkloadJob
	default:
		return domain.WorkloadUnknown
	}
}
