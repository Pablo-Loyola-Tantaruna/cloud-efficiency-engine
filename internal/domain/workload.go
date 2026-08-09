package domain

type WorkloadMetrics struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Replicas  int    `json:"replicas"`

	CPURequestMillicores int64 `json:"cpuRequestMillicores"`
	CPUUsageMillicores   int64 `json:"cpuUsageMillicores"`

	MemoryRequestBytes int64 `json:"memoryRequestBytes"`
	MemoryUsageBytes   int64 `json:"memoryUsageBytes"`
}
