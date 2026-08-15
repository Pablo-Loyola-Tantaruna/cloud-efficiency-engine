package aws

import (
	"context"

	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/domain"
)

type MetricsClient interface {
	GetWorkloads(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
	) ([]WorkloadResource, error)

	GetWorkloadHistory(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
	) (map[string][]WorkloadSample, error)
}

type PricingClient interface {
	GetPricing(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
	) (ResourcePrice, error)
}

type BillingClient interface {
	GetCost(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
		query billing.CostQuery,
	) (billing.CostReport, error)
}
