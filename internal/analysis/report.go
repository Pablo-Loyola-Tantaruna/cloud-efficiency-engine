package analysis

import (
	"cloud-efficiency-engine/internal/cost"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type AnalysisReport struct {
	GeneratedAt time.Time `json:"generatedAt"`

	Summary AnalysisSummary `json:"summary"`

	Workloads []WorkloadAnalysis `json:"workloads"`
}

type AnalysisSummary struct {
	TotalWorkloads int `json:"totalWorkloads"`

	OptimizableWorkloads int `json:"optimizableWorkloads"`

	CurrentMonthlyCostUSD float64 `json:"currentMonthlyCostUsd"`

	OptimizedMonthlyCostUSD float64 `json:"optimizedMonthlyCostUsd"`

	PotentialSavingsUSD float64 `json:"potentialSavingsUsd"`

	SavingsPercentage float64 `json:"savingsPercentage"`
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
