package rules

import (
	"cloud-efficiency-engine/internal/domain"
)

type Rule interface {
	Evaluate(workload domain.WorkloadMetrics) *domain.Recommendation
}
