package analysis

import (
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type AnalysisOptions struct {
	Namespace string

	Start time.Time

	End time.Time

	Step time.Duration

	Context domain.AnalysisContext
}
