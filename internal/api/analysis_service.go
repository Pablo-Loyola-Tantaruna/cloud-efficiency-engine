package api

import (
	"context"

	"cloud-efficiency-engine/internal/analysis"
)

type AnalysisService interface {
	Analyze(
		ctx context.Context,
		options analysis.AnalysisOptions,
	) (*analysis.AnalysisReport, error)
}
