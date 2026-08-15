package providers

import (
	"context"
	"time"

	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/pricing"
)

type MetricsSource interface {
	GetWorkloads(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
		namespace string,
	) ([]domain.WorkloadMetrics, error)

	GetWorkloadHistory(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
		namespace string,
		start time.Time,
		end time.Time,
		step time.Duration,
	) ([]domain.WorkloadHistory, error)
}

type PricingSource interface {
	GetPricing(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
	) (pricing.ResourcePricing, error)
}

type CapacitySource interface {
	GetCapacity(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
	) (int64, int64, error)
}
