package analysis

import "time"

type AnalysisOptions struct {
	Namespace string
	Start     time.Time
	End       time.Time
	Step      time.Duration
}
