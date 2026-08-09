package analysis

import (
	"cloud-efficiency-engine/internal/domain"
	"context"

	"cloud-efficiency-engine/internal/analysis/rules"
	"cloud-efficiency-engine/internal/metrics"
)

type Analyzer struct {
	provider metrics.Provider
	rules    []rules.Rule
}

func NewAnalyzer(
	provider metrics.Provider,
	rules []rules.Rule,
) *Analyzer {
	return &Analyzer{
		provider: provider,
		rules:    rules,
	}
}

func (a *Analyzer) Analyze(ctx context.Context) (AnalysisResult, error) {
	workloads, err := a.provider.GetWorkloads(ctx)
	if err != nil {
		return AnalysisResult{}, err
	}

	result := AnalysisResult{
		WorkloadsAnalyzed: len(workloads),
		Recommendations:   make([]domain.Recommendation, 0),
	}

	for _, workload := range workloads {
		for _, rule := range a.rules {
			recommendation := rule.Evaluate(workload)

			if recommendation != nil {
				result.Recommendations = append(
					result.Recommendations,
					*recommendation,
				)
			}
		}
	}

	return result, nil
}
