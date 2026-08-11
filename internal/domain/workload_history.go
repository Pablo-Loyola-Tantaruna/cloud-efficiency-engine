package domain

import "time"

type MetricSample struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type WorkloadHistory struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`

	CPUUsageMillicores []MetricSample `json:"cpuUsageMillicores"`
	MemoryUsageBytes   []MetricSample `json:"memoryUsageBytes"`
}
