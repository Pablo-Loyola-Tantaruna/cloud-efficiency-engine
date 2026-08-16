package analysis

import (
	"encoding/json"
	"time"

	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

type AnalysisReport struct {
	GeneratedAt time.Time `json:"generatedAt"`

	Context domain.AnalysisContext `json:"context"`

	Summary AnalysisSummary `json:"summary"`

	NamespaceBreakdown []NamespaceCostBreakdown `json:"namespaceBreakdown"`

	TopRecommendations []domain.Recommendation `json:"topRecommendations"`

	Workloads []WorkloadAnalysis `json:"workloads"`

	Billing *billing.CostReport `json:"billing,omitempty"`

	Attribution *cost.AttributionReport `json:"attribution,omitempty"`
}

type AnalysisSummary struct {
	TotalWorkloads int `json:"totalWorkloads"`

	OptimizableWorkloads int `json:"optimizableWorkloads"`

	CurrentMonthlyCostUSD float64 `json:"currentMonthlyCostUsd"`

	OptimizedMonthlyCostUSD float64 `json:"optimizedMonthlyCostUsd"`

	PotentialSavingsUSD float64 `json:"potentialSavingsUsd"`

	SavingsPercentage float64 `json:"savingsPercentage"`

	ActualCloudCostUSD float64 `json:"actualCloudCostUsd"`

	CostVarianceUSD float64 `json:"costVarianceUsd"`
}

type WorkloadAnalysis struct {
	Workload domain.WorkloadMetrics `json:"workload"`

	Status string `json:"status"`

	History *domain.WorkloadHistory `json:"history,omitempty"`

	Recommendations []domain.Recommendation `json:"recommendations"`

	Cost *cost.CostEstimate `json:"cost,omitempty"`
}

const (
	WorkloadAnalysisStatusAnalyzed = "ANALYZED"

	WorkloadAnalysisStatusInsufficientData = "INSUFFICIENT_DATA"
)

func (report AnalysisReport) MarshalJSON() ([]byte, error) {
	type alias AnalysisReport

	copy := alias(report)
	copy.TopRecommendations = buildTopRecommendations(
		report.Workloads,
		report.Context.ClusterName,
		report.Attribution,
	)

	return json.Marshal(copy)
}
