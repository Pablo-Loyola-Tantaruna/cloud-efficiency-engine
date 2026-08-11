package optimizer

import (
	"fmt"

	"cloud-efficiency-engine/internal/analysis/statistics"
	"cloud-efficiency-engine/internal/domain"
)

type Engine struct {
	policy OptimizationPolicy
}

func NewEngine(
	policy OptimizationPolicy,
) *Engine {

	return &Engine{
		policy: policy,
	}
}

func (e *Engine) Optimize(
	workload domain.WorkloadMetrics,
	history domain.WorkloadHistory,
) (OptimizationResult, error) {

	cpu, err := e.OptimizeCPU(
		workload,
		history,
	)

	if err != nil {
		return OptimizationResult{}, err
	}

	memory, err := e.OptimizeMemory(
		workload,
		history,
	)

	if err != nil {
		return OptimizationResult{}, err
	}

	return OptimizationResult{
		CPURecommendation: ToCPURecommendation(
			workload,
			cpu,
		),

		MemoryRecommendation: ToMemoryRecommendation(
			workload,
			memory,
		),
	}, nil
}

func (e *Engine) OptimizeCPU(
	workload domain.WorkloadMetrics,
	history domain.WorkloadHistory,
) (*ResourceRecommendation, error) {

	if workload.Namespace != history.Namespace ||
		workload.Name != history.Name {

		return nil, fmt.Errorf(
			"workload and history do not match",
		)
	}

	values := statistics.SampleValues(
		history.CPUUsageMillicores,
	)

	stats, err := statistics.CalculateResourceStatistics(
		values,
	)

	if err != nil {
		return nil, err
	}

	return RecommendCPU(
		workload.CPURequestMillicores,
		stats,
		e.policy,
	)
}

func (e *Engine) OptimizeMemory(
	workload domain.WorkloadMetrics,
	history domain.WorkloadHistory,
) (*ResourceRecommendation, error) {

	if workload.Namespace != history.Namespace ||
		workload.Name != history.Name {

		return nil, fmt.Errorf(
			"workload and history do not match",
		)
	}

	values := statistics.SampleValues(
		history.MemoryUsageBytes,
	)

	stats, err := statistics.CalculateResourceStatistics(
		values,
	)

	if err != nil {
		return nil, err
	}

	return RecommendMemory(
		workload.MemoryRequestBytes,
		stats,
		e.policy,
	)
}
