package metrics

import (
	"context"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type HistoricalProvider interface {
	GetWorkloadHistory(
		ctx context.Context,
		namespace string,
		start time.Time,
		end time.Time,
		step time.Duration,
	) ([]domain.WorkloadHistory, error)
}

type ContextAwareHistoricalProvider interface {
	GetWorkloadHistoryWithContext(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
		namespace string,
		start time.Time,
		end time.Time,
		step time.Duration,
	) ([]domain.WorkloadHistory, error)
}
