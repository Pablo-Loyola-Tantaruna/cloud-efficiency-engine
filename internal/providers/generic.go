package providers

import (
	"context"
	"fmt"
	"time"

	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/pricing"
)

type GenericProvider struct {
	metricsSource MetricsSource
	pricingSource PricingSource
}

func NewGenericProvider(
	metricsSource MetricsSource,
	pricingSource PricingSource,
) *GenericProvider {

	return &GenericProvider{
		metricsSource: metricsSource,
		pricingSource: pricingSource,
	}
}

func (p *GenericProvider) GetWorkloads(
	ctx context.Context,
	namespace string,
) ([]domain.WorkloadMetrics, error) {

	return p.GetWorkloadsWithContext(
		ctx,
		domain.AnalysisContext{},
		namespace,
	)
}

func (p *GenericProvider) GetWorkloadsWithContext(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
) ([]domain.WorkloadMetrics, error) {

	if p.metricsSource == nil {

		return nil,
			fmt.Errorf(
				"metrics source is not configured",
			)
	}

	return p.metricsSource.GetWorkloads(
		ctx,
		domain.NormalizeAnalysisContext(
			analysisContext,
		),
		namespace,
	)
}

func (p *GenericProvider) GetWorkloadHistory(
	ctx context.Context,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {

	return p.GetWorkloadHistoryWithContext(
		ctx,
		domain.AnalysisContext{},
		namespace,
		start,
		end,
		step,
	)
}

func (p *GenericProvider) GetWorkloadHistoryWithContext(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {

	if p.metricsSource == nil {

		return nil,
			fmt.Errorf(
				"metrics source is not configured",
			)
	}

	return p.metricsSource.GetWorkloadHistory(
		ctx,
		domain.NormalizeAnalysisContext(
			analysisContext,
		),
		namespace,
		start,
		end,
		step,
	)
}

func (p *GenericProvider) GetPricing(
	ctx context.Context,
) (pricing.ResourcePricing, error) {

	return p.GetPricingWithContext(
		ctx,
		domain.AnalysisContext{},
	)
}

func (p *GenericProvider) GetPricingWithContext(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (pricing.ResourcePricing, error) {

	if p.pricingSource == nil {

		return pricing.ResourcePricing{},
			fmt.Errorf(
				"pricing source is not configured",
			)
	}

	return p.pricingSource.GetPricing(
		ctx,
		domain.NormalizeAnalysisContext(
			analysisContext,
		),
	)
}

var _ metrics.Provider = (*GenericProvider)(nil)

var _ metrics.ContextAwareProvider = (*GenericProvider)(nil)

var _ metrics.HistoricalProvider = (*GenericProvider)(nil)

var _ metrics.ContextAwareHistoricalProvider = (*GenericProvider)(nil)

var _ pricing.Provider = (*GenericProvider)(nil)

var _ pricing.ContextAwareProvider = (*GenericProvider)(nil)
