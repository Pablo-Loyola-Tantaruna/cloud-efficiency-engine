package domain

import "sort"

type ExecutionHistory struct {
	PlanID        string            `json:"planId"`
	ActionID      string            `json:"actionId"`
	Latest        ExecutionRecord   `json:"latest"`
	Attempts      []ExecutionRecord `json:"attempts"`
	TotalAttempts int               `json:"totalAttempts"`
}

func NewExecutionHistory(records []ExecutionRecord) ExecutionHistory {
	ordered := append([]ExecutionRecord(nil), records...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Attempt == ordered[j].Attempt {
			return ordered[i].StartedAt.Before(ordered[j].StartedAt)
		}
		return ordered[i].Attempt < ordered[j].Attempt
	})
	history := ExecutionHistory{Attempts: ordered, TotalAttempts: len(ordered)}
	if len(ordered) > 0 {
		history.PlanID = ordered[0].PlanID
		history.ActionID = ordered[0].ActionID
		history.Latest = ordered[len(ordered)-1]
	}
	return history
}
