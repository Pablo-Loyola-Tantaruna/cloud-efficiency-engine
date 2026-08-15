package billing

import (
	"context"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type CostQuery struct {
	Start time.Time

	End time.Time

	Service string
}

type CostPeriod struct {
	Start time.Time `json:"start"`

	End time.Time `json:"end"`

	Service string `json:"service"`

	AmountUSD float64 `json:"amountUsd"`

	Unit string `json:"unit"`

	Estimated bool `json:"estimated"`
}

type CostReport struct {
	Provider domain.CloudProvider `json:"provider"`

	Start time.Time `json:"start"`

	End time.Time `json:"end"`

	Currency string `json:"currency"`

	TotalUSD float64 `json:"totalUsd"`

	Estimated bool `json:"estimated"`

	Periods []CostPeriod `json:"periods"`
}

type Provider interface {
	GetCost(
		ctx context.Context,
		query CostQuery,
	) (CostReport, error)
}

type ContextAwareProvider interface {
	GetCostWithContext(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
		query CostQuery,
	) (CostReport, error)
}
