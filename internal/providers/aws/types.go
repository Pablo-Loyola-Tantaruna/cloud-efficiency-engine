package aws

import "time"

type WorkloadResource struct {
	Namespace string

	Name string

	Type string

	Replicas int

	CPURequestMillicores int64

	CPUUsageMillicores int64

	MemoryRequestBytes int64

	MemoryUsageBytes int64
}

type WorkloadSample struct {
	Timestamp time.Time

	CPUUsageMillicores float64

	MemoryUsageBytes float64
}

type ResourcePrice struct {
	CPUPerCoreHour float64

	MemoryPerGBHour float64
}
