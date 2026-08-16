package domain

import "fmt"

type ActionPlanStatus string

const (
	ActionPlanPreview         ActionPlanStatus = "PREVIEW"
	ActionPlanPendingApproval ActionPlanStatus = "PENDING_APPROVAL"
	ActionPlanApproved        ActionPlanStatus = "APPROVED"
	ActionPlanReadyToApply    ActionPlanStatus = "READY_TO_APPLY"
	ActionPlanApplied         ActionPlanStatus = "APPLIED"
	ActionPlanFailed          ActionPlanStatus = "FAILED"
)

type ActionPlan struct {
	ID                        string           `json:"id"`
	Provider                  CloudProvider    `json:"provider"`
	Cluster                   string           `json:"cluster"`
	Status                    ActionPlanStatus `json:"status"`
	Actions                   []Action         `json:"actions"`
	TotalMonthlySavingsUSD    float64          `json:"totalMonthlySavingsUsd"`
	TotalAnnualizedSavingsUSD float64          `json:"totalAnnualizedSavingsUsd"`
	RequiresApproval          bool             `json:"requiresApproval"`
}

func (p ActionPlan) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("action plan id must not be empty")
	}
	if !p.Provider.IsValid() {
		return fmt.Errorf("action plan provider must be valid")
	}
	if p.Cluster == "" {
		return fmt.Errorf("action plan cluster must not be empty")
	}
	if p.Status == "" {
		return fmt.Errorf("action plan status must not be empty")
	}
	if !validActionPlanStatus(p.Status) {
		return fmt.Errorf("unsupported action plan status %q", p.Status)
	}
	if len(p.Actions) == 0 {
		return fmt.Errorf("action plan must contain at least one action")
	}
	for _, action := range p.Actions {
		if err := action.Validate(); err != nil {
			return fmt.Errorf("invalid action %q: %w", action.ID, err)
		}
	}
	if p.TotalMonthlySavingsUSD <= 0 || p.TotalAnnualizedSavingsUSD <= 0 {
		return fmt.Errorf("action plan savings must be greater than zero")
	}
	if !p.RequiresApproval {
		return fmt.Errorf("action plan requires explicit approval")
	}
	return nil
}

func validActionPlanStatus(status ActionPlanStatus) bool {
	switch status {
	case ActionPlanPreview, ActionPlanPendingApproval, ActionPlanApproved,
		ActionPlanReadyToApply, ActionPlanApplied, ActionPlanFailed:
		return true
	default:
		return false
	}
}

func CanTransitionActionPlan(from, to ActionPlanStatus) bool {
	switch from {
	case ActionPlanPreview:
		return to == ActionPlanPendingApproval
	case ActionPlanPendingApproval:
		return to == ActionPlanApproved || to == ActionPlanFailed
	case ActionPlanApproved:
		return to == ActionPlanReadyToApply || to == ActionPlanFailed
	case ActionPlanReadyToApply:
		return to == ActionPlanApplied || to == ActionPlanFailed
	case ActionPlanApplied, ActionPlanFailed:
		return false
	default:
		return false
	}
}
