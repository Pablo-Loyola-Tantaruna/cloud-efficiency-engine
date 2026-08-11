package analysis

import (
	"context"
	"time"

	"cloud-efficiency-engine/internal/analysis/optimizer"
	"cloud-efficiency-engine/internal/analysis/resolver"
	"cloud-efficiency-engine/internal/analysis/rules"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/pricing"
)

type Engine struct {
	provider           metrics.Provider
	historicalProvider metrics.HistoricalProvider

	pricingProvider pricing.Provider

	rules []rules.Rule

	optimizer *optimizer.Engine
	resolver  *resolver.Resolver

	costCalculator *cost.Calculator
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
	}
}

func (e *Engine) Analyze(
	ctx context.Context,
	options AnalysisOptions,
) (*AnalysisReport, error) {

	workloads, err :=
		e.provider.GetWorkloads(ctx)

	if err != nil {
		return nil, err
	}

	histories :=
		make(
			[]domain.WorkloadHistory,
			0,
		)

	if e.historicalProvider != nil {

		histories, err =
			e.historicalProvider.GetWorkloadHistory(
				ctx,
				options.Start,
				options.End,
				options.Step,
			)

		if err != nil {
			return nil, err
		}
	}

	var prices pricing.ResourcePricing

	if e.pricingProvider != nil {

		prices, err =
			e.pricingProvider.GetPricing(ctx)

		if err != nil {
			return nil, err
		}
	}

	report :=
		&AnalysisReport{
			GeneratedAt: time.Now().UTC(),

			Workloads: make(
				[]WorkloadAnalysis,
				0,
				len(workloads),
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

	e.calculateSummary(report)

	return report, nil
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

	if len(histories) == 0 {
		return result
	}

	history, err :=
		MatchHistory(
			workload,
			histories,
		)

	if err != nil {
		return result
	}

	result.History =
		history

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

	if e.optimizer != nil {

		cpuRecommendation, err :=
			e.optimizer.OptimizeCPU(
				workload,
				*history,
			)

		if err != nil {
			return result
		}

		if cpuRecommendation != nil {

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

		if err != nil {
			return result
		}

		if memoryRecommendation != nil {

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

	result.Status =
		WorkloadAnalysisStatusAnalyzed

	return result
}

func (e *Engine) calculateSummary(
	report *AnalysisReport,
) {

	report.Summary.TotalWorkloads =
		len(report.Workloads)

	for _, workload := range report.Workloads {

		if len(workload.Recommendations) > 0 {

			report.Summary.
				OptimizableWorkloads++
		}

		if workload.Cost == nil {
			continue
		}

		report.Summary.CurrentMonthlyCostUSD +=
			workload.Cost.CurrentMonthlyCostUSD

		report.Summary.OptimizedMonthlyCostUSD +=
			workload.Cost.OptimizedMonthlyCostUSD

		report.Summary.PotentialSavingsUSD +=
			workload.Cost.PotentialSavingsUSD
	}

	if report.Summary.CurrentMonthlyCostUSD > 0 {

		report.Summary.SavingsPercentage =
			report.Summary.PotentialSavingsUSD /
				report.Summary.CurrentMonthlyCostUSD *
				100
	}
}
