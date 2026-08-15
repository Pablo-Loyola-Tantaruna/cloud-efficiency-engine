package kubernetes

import (
	"context"
	"time"

	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics/providers"
)

type MetricsSource struct {
	provider *providers.PrometheusProvider
}

func NewMetricsSource(
	prometheusURL string,
) *MetricsSource {

	return &MetricsSource{
		provider: providers.NewPrometheusProvider(
			prometheusURL,
			nil,
		),
	}
}

func (s *MetricsSource) GetWorkloads(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
) ([]domain.WorkloadMetrics, error) {

	// The Prometheus provider already performs the
	// normalization required by the Kubernetes metrics contract.
	return s.provider.GetWorkloads(
		ctx,
		namespace,
	)
}

func (s *MetricsSource) GetWorkloadHistory(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {

	return s.provider.GetWorkloadHistory(
		ctx,
		namespace,
		start,
		end,
		step,
	)
}
