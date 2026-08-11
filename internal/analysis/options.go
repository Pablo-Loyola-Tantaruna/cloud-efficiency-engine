package analysis

import (
	"time"
)

type AnalysisOptions struct {
	Start time.Time
	End   time.Time
	Step  time.Duration
}
