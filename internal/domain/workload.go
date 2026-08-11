package domain

type WorkloadType string

const (
	WorkloadDeployment  WorkloadType = "Deployment"
	WorkloadStatefulSet WorkloadType = "StatefulSet"
	WorkloadDaemonSet   WorkloadType = "DaemonSet"
	WorkloadJob         WorkloadType = "Job"
	WorkloadUnknown     WorkloadType = "Unknown"
)

type WorkloadMetrics struct {
	Namespace string       `json:"namespace"`
	Name      string       `json:"name"`
	Type      WorkloadType `json:"type"`

	Replicas int `json:"replicas"`

	CPURequestMillicores int64 `json:"cpuRequestMillicores"`
	CPUUsageMillicores   int64 `json:"cpuUsageMillicores"`

	MemoryRequestBytes int64 `json:"memoryRequestBytes"`
	MemoryUsageBytes   int64 `json:"memoryUsageBytes"`
}
