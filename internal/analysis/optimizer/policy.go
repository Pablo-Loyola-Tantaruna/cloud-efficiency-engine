package optimizer

type OptimizationPolicy struct {
	CPUPercentile    float64
	MemoryPercentile float64

	CPUSafetyMargin    float64
	MemorySafetyMargin float64

	MinCPURequestMillicores int64
	MinMemoryRequestBytes   int64

	CPUGranularityMillicores int64
	MemoryGranularityBytes   int64

	MinReductionPercentage float64

	MinimumSamplesForMediumConfidence int
	MinimumSamplesForHighConfidence   int
}

func DefaultOptimizationPolicy() OptimizationPolicy {
	return OptimizationPolicy{
		CPUPercentile:    95,
		MemoryPercentile: 99,

		CPUSafetyMargin:    0.20,
		MemorySafetyMargin: 0.15,

		MinCPURequestMillicores: 100,
		MinMemoryRequestBytes:   128 * 1024 * 1024,

		CPUGranularityMillicores: 50,
		MemoryGranularityBytes:   64 * 1024 * 1024,

		MinReductionPercentage: 20,

		MinimumSamplesForMediumConfidence: 300,
		MinimumSamplesForHighConfidence:   1000,
	}
}
