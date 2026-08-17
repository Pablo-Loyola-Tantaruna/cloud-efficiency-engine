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

	// ContainerName is populated when the metrics source can identify a unique
	// target container. It remains empty for multi-container workloads where
	// the aggregate metrics cannot safely identify a mutation target.
	ContainerName string `json:"containerName,omitempty"`

	Replicas int `json:"replicas"`

	// NodeGroup is populated when the metrics source can identify a unique
	// node group / node pool for the workload. It remains empty when placement
	// is unavailable or ambiguous.
	NodeGroup string `json:"nodeGroup,omitempty"`

	CPURequestMillicores int64 `json:"cpuRequestMillicores"`
	CPUUsageMillicores   int64 `json:"cpuUsageMillicores"`

	MemoryRequestBytes int64 `json:"memoryRequestBytes"`
	MemoryUsageBytes   int64 `json:"memoryUsageBytes"`
}
