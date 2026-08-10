package analysis

import (
	"context"

	"cloud-efficiency-engine/internal/analysis/rules"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
)

type Analyzer struct {
	provider   metrics.Provider
	rules      []rules.Rule
	calculator *cost.Calculator
}

func NewAnalyzer(
	provider metrics.Provider,
	rules []rules.Rule,
	calculator *cost.Calculator,
) *Analyzer {

	return &Analyzer{
		provider:   provider,
		rules:      rules,
		calculator: calculator,
	}
}

func (a *Analyzer) Analyze(
	ctx context.Context,
) (AnalysisResult, error) {

	workloads, err := a.provider.GetWorkloads(ctx)

	if err != nil {
		return AnalysisResult{}, err
	}

	result := AnalysisResult{
		WorkloadsAnalyzed: len(workloads),
		Recommendations:   make([]domain.Recommendation, 0),
		CostEstimates:     make(map[string]cost.CostEstimate),
	}

	for _, workload := range workloads {

		workloadRecommendations := make(
			[]domain.Recommendation,
			0,
		)

		for _, rule := range a.rules {

			recommendation := rule.Evaluate(workload)

			if recommendation == nil {
				continue
			}

			workloadRecommendations = append(
				workloadRecommendations,
				*recommendation,
			)

			result.Recommendations = append(
				result.Recommendations,
				*recommendation,
			)
		}

		estimate := a.calculator.Estimate(
			workload,
			workloadRecommendations,
		)

		key := workload.Namespace + "/" + workload.Name

		result.CostEstimates[key] = estimate
	}

	return result, nil
}
