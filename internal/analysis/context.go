package analysis

import "cloud-efficiency-engine/internal/domain"

type AnalysisContext struct {
	Workloads []domain.WorkloadMetrics

	Histories []domain.WorkloadHistory
}
