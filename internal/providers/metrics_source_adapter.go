package providers

import (
	"context"
	"time"

	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
)

type MetricsSourceAdapter struct {
	provider           metrics.Provider
	historicalProvider metrics.HistoricalProvider
}

func NewMetricsSourceAdapter(
	provider metrics.Provider,
	historicalProvider metrics.HistoricalProvider,
) *MetricsSourceAdapter {

	return &MetricsSourceAdapter{
		provider: provider,

		historicalProvider: historicalProvider,
	}
}

func (a *MetricsSourceAdapter) GetWorkloads(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
) ([]domain.WorkloadMetrics, error) {

	contextAwareProvider, ok :=
		a.provider.(metrics.ContextAwareProvider)

	if ok {

		return contextAwareProvider.
			GetWorkloadsWithContext(
				ctx,
				analysisContext,
				namespace,
			)
	}

	return a.provider.GetWorkloads(
		ctx,
		namespace,
	)
}

func (a *MetricsSourceAdapter) GetWorkloadHistory(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {

	contextAwareProvider, ok :=
		a.historicalProvider.(metrics.ContextAwareHistoricalProvider)

	if ok {

		return contextAwareProvider.
			GetWorkloadHistoryWithContext(
				ctx,
				analysisContext,
				namespace,
				start,
				end,
				step,
			)
	}

	return a.historicalProvider.
		GetWorkloadHistory(
			ctx,
			namespace,
			start,
			end,
			step,
		)
}
