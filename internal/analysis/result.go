package analysis

import (
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

type AnalysisResult struct {
	WorkloadsAnalyzed int                          `json:"workloadsAnalyzed"`
	Recommendations   []domain.Recommendation      `json:"recommendations"`
	CostEstimates     map[string]cost.CostEstimate `json:"costEstimates"`
}
