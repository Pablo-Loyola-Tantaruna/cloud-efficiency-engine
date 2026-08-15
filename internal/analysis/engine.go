package analysis

import (
	"cloud-efficiency-engine/internal/analysis/capacity"
	"context"
	"time"

	"cloud-efficiency-engine/internal/analysis/optimizer"
	"cloud-efficiency-engine/internal/analysis/resolver"
	"cloud-efficiency-engine/internal/analysis/rules"
	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/pricing"
	providerregistry "cloud-efficiency-engine/internal/providers"
)

type Engine struct {
	provider metrics.Provider

	historicalProvider metrics.HistoricalProvider

	pricingProvider pricing.Provider

	providerRegistry *providerregistry.Registry

	rules []rules.Rule

	optimizer *optimizer.Engine

	resolver *resolver.Resolver

	costCalculator *cost.Calculator

	costAttributor *cost.CostAttributor
}

func NewEngine(
	provider metrics.Provider,
	historicalProvider metrics.HistoricalProvider,
	pricingProvider pricing.Provider,
	rules []rules.Rule,
	optimizationPolicy optimizer.OptimizationPolicy,
	recommendationResolver *resolver.Resolver,
	costCalculator *cost.Calculator,
) *Engine {

	if recommendationResolver == nil {

		recommendationResolver =
			resolver.NewResolver()
	}

	return &Engine{
		provider: provider,

		historicalProvider: historicalProvider,

		pricingProvider: pricingProvider,

		rules: rules,

		optimizer: optimizer.NewEngine(
			optimizationPolicy,
		),

		resolver: recommendationResolver,

		costCalculator: costCalculator,

		costAttributor: cost.NewCostAttributor(),
	}
}

func NewEngineWithRegistry(
	providerRegistry *providerregistry.Registry,
	rules []rules.Rule,
	optimizationPolicy optimizer.OptimizationPolicy,
	recommendationResolver *resolver.Resolver,
	costCalculator *cost.Calculator,
) *Engine {

	if recommendationResolver == nil {

		recommendationResolver =
			resolver.NewResolver()
	}

	return &Engine{
		providerRegistry: providerRegistry,

		rules: rules,

		optimizer: optimizer.NewEngine(
			optimizationPolicy,
		),

		resolver: recommendationResolver,

		costCalculator: costCalculator,

		costAttributor: cost.NewCostAttributor(),
	}
}

func (e *Engine) Analyze(
	ctx context.Context,
	options AnalysisOptions,
) (*AnalysisReport, error) {

	analysisContext :=
		domain.NormalizeAnalysisContext(
			options.Context,
		)

	var (
		metricsProvider    metrics.Provider
		historicalProvider metrics.HistoricalProvider
		pricingProvider    pricing.Provider
		billingProvider    billing.Provider
		capacityProvider   capacity.Provider
	)

	if e.providerRegistry != nil {

		bundle, err :=
			e.providerRegistry.Resolve(
				ctx,
				analysisContext,
			)

		if err != nil {
			return nil, err
		}

		metricsProvider =
			bundle.MetricsProvider

		historicalProvider =
			bundle.HistoricalProvider

		pricingProvider =
			bundle.PricingProvider

		billingProvider =
			bundle.BillingProvider

		capacityProvider =
			bundle.CapacityProvider

	} else {

		metricsProvider =
			e.provider

		historicalProvider =
			e.historicalProvider

		pricingProvider =
			e.pricingProvider
	}

	workloads, err :=
		e.getWorkloads(
			ctx,
			analysisContext,
			options,
			metricsProvider,
		)

	if err != nil {
		return nil, err
	}

	histories :=
		make(
			[]domain.WorkloadHistory,
			0,
		)

	if historicalProvider != nil {

		histories, err =
			e.getWorkloadHistory(
				ctx,
				analysisContext,
				options,
				historicalProvider,
			)

		if err != nil {
			return nil, err
		}
	}

	var prices pricing.ResourcePricing

	if pricingProvider != nil {

		prices, err =
			e.getPricing(
				ctx,
				analysisContext,
				pricingProvider,
			)

		if err != nil {
			return nil, err
		}
	}

	var actualCost *billing.CostReport

	if billingProvider != nil {

		costReport, billingErr :=
			e.getBillingCost(
				ctx,
				analysisContext,
				options,
				billingProvider,
			)

		if billingErr != nil {
			return nil, billingErr
		}

		actualCost =
			&costReport
	}

	report :=
		&AnalysisReport{
			GeneratedAt: time.Now().UTC(),

			Context: analysisContext,

			Billing: actualCost,

			Workloads: make(
				[]WorkloadAnalysis,
				0,
				len(workloads),
			),

			NamespaceBreakdown: make(
				[]NamespaceCostBreakdown,
				0,
			),
		}

	for _, workload := range workloads {

		workloadAnalysis :=
			e.analyzeWorkload(
				workload,
				histories,
				prices,
			)

		report.Workloads =
			append(
				report.Workloads,
				workloadAnalysis,
			)
	}

	e.calculateSummary(
		report,
	)

	e.calculateBillingSummary(
		report,
	)

	if capacityProvider != nil &&
		actualCost != nil &&
		e.costAttributor != nil {

		attribution, attributionErr :=
			e.calculateAttribution(
				ctx,
				analysisContext,
				workloads,
				actualCost,
				capacityProvider,
			)

		if attributionErr != nil {
			return nil, attributionErr
		}

		report.Attribution =
			&attribution
	}

	report.NamespaceBreakdown =
		buildNamespaceCostBreakdown(
			report.Workloads,
		)

	return report, nil
}

func (e *Engine) calculateAttribution(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	workloads []domain.WorkloadMetrics,
	actualCost *billing.CostReport,
	provider capacity.Provider,
) (cost.AttributionReport, error) {

	cluster, err :=
		provider.GetCapacity(
			ctx,
			analysisContext,
		)

	if err != nil {
		return cost.AttributionReport{},
			err
	}

	monthlyCost, err :=
		cost.MonthlyizeCost(
			actualCost.TotalUSD,
			actualCost.Start,
			actualCost.End,
		)

	if err != nil {

		return cost.AttributionReport{},
			err
	}

	cluster.MonthlyCostUSD =
		monthlyCost

	return e.costAttributor.Attribute(
		workloads,
		cluster,
	)
}

func (e *Engine) getWorkloads(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	options AnalysisOptions,
	provider metrics.Provider,
) ([]domain.WorkloadMetrics, error) {

	contextAwareProvider, ok :=
		provider.(metrics.ContextAwareProvider)

	if ok {

		return contextAwareProvider.
			GetWorkloadsWithContext(
				ctx,
				analysisContext,
				options.Namespace,
			)
	}

	return provider.GetWorkloads(
		ctx,
		options.Namespace,
	)
}

func (e *Engine) getWorkloadHistory(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	options AnalysisOptions,
	provider metrics.HistoricalProvider,
) ([]domain.WorkloadHistory, error) {

	contextAwareProvider, ok :=
		provider.(metrics.ContextAwareHistoricalProvider)

	if ok {

		return contextAwareProvider.
			GetWorkloadHistoryWithContext(
				ctx,
				analysisContext,
				options.Namespace,
				options.Start,
				options.End,
				options.Step,
			)
	}

	return provider.
		GetWorkloadHistory(
			ctx,
			options.Namespace,
			options.Start,
			options.End,
			options.Step,
		)
}

func (e *Engine) getPricing(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	provider pricing.Provider,
) (pricing.ResourcePricing, error) {

	contextAwareProvider, ok :=
		provider.(pricing.ContextAwareProvider)

	if ok {

		return contextAwareProvider.
			GetPricingWithContext(
				ctx,
				analysisContext,
			)
	}

	return provider.GetPricing(
		ctx,
	)
}

func (e *Engine) analyzeWorkload(
	workload domain.WorkloadMetrics,
	histories []domain.WorkloadHistory,
	prices pricing.ResourcePricing,
) WorkloadAnalysis {

	result :=
		WorkloadAnalysis{
			Workload: workload,

			Status: WorkloadAnalysisStatusInsufficientData,

			Recommendations: make(
				[]domain.Recommendation,
				0,
			),
		}

	var history *domain.WorkloadHistory

	if len(histories) > 0 {

		matchedHistory, err :=
			MatchHistory(
				workload,
				histories,
			)

		if err == nil {

			history =
				matchedHistory

			result.History =
				matchedHistory
		}
	}

	recommendations :=
		make(
			[]domain.Recommendation,
			0,
		)

	for _, rule := range e.rules {

		recommendation :=
			rule.Evaluate(
				workload,
			)

		if recommendation == nil {
			continue
		}

		recommendations =
			append(
				recommendations,
				*recommendation,
			)
	}

	if history != nil &&
		e.optimizer != nil {

		cpuRecommendation, err :=
			e.optimizer.OptimizeCPU(
				workload,
				*history,
			)

		if err == nil &&
			cpuRecommendation != nil {

			recommendation :=
				optimizer.ToCPURecommendation(
					workload,
					cpuRecommendation,
				)

			if recommendation != nil {

				recommendations =
					append(
						recommendations,
						*recommendation,
					)
			}
		}

		memoryRecommendation, err :=
			e.optimizer.OptimizeMemory(
				workload,
				*history,
			)

		if err == nil &&
			memoryRecommendation != nil {

			recommendation :=
				optimizer.ToMemoryRecommendation(
					workload,
					memoryRecommendation,
				)

			if recommendation != nil {

				recommendations =
					append(
						recommendations,
						*recommendation,
					)
			}
		}
	}

	if e.resolver != nil {

		recommendations =
			e.resolver.Resolve(
				recommendations,
			)
	}

	result.Recommendations =
		recommendations

	if e.costCalculator != nil {

		estimate :=
			e.costCalculator.Estimate(
				workload,
				recommendations,
				prices,
			)

		result.Cost =
			&estimate
	}

	if history != nil {

		result.Status =
			WorkloadAnalysisStatusAnalyzed
	}

	return result
}

func (e *Engine) calculateSummary(
	report *AnalysisReport,
) {

	report.Summary.TotalWorkloads =
		len(
			report.Workloads,
		)

	for _, workload := range report.Workloads {

		if len(
			workload.Recommendations,
		) > 0 {

			report.Summary.
				OptimizableWorkloads++
		}

		if workload.Cost == nil {
			continue
		}

		report.Summary.
			CurrentMonthlyCostUSD +=
			workload.Cost.
				CurrentMonthlyCostUSD

		report.Summary.
			OptimizedMonthlyCostUSD +=
			workload.Cost.
				OptimizedMonthlyCostUSD

		report.Summary.
			PotentialSavingsUSD +=
			workload.Cost.
				PotentialSavingsUSD
	}

	if report.Summary.
		CurrentMonthlyCostUSD > 0 {

		report.Summary.
			SavingsPercentage =
			report.Summary.
				PotentialSavingsUSD /
				report.Summary.
					CurrentMonthlyCostUSD *
				100
	}
}

func (e *Engine) getBillingCost(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	options AnalysisOptions,
	provider billing.Provider,
) (billing.CostReport, error) {

	query :=
		billing.CostQuery{
			Start: options.Start,

			End: options.End,
		}

	contextAwareProvider, ok :=
		provider.(billing.ContextAwareProvider)

	if ok {

		return contextAwareProvider.
			GetCostWithContext(
				ctx,
				analysisContext,
				query,
			)
	}

	return provider.GetCost(
		ctx,
		query,
	)
}

func (e *Engine) calculateBillingSummary(
	report *AnalysisReport,
) {

	if report == nil ||
		report.Billing == nil {

		return
	}

	report.Summary.ActualCloudCostUSD =
		report.Billing.TotalUSD

	report.Summary.CostVarianceUSD =
		report.Billing.TotalUSD -
			report.Summary.CurrentMonthlyCostUSD
}
