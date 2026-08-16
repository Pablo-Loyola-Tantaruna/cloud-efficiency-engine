package actions

import (
	"fmt"

	"cloud-efficiency-engine/internal/domain"
)

func TransitionActionPlan(plan domain.ActionPlan, target domain.ActionPlanStatus) (domain.ActionPlan, error) {
	if err := plan.Validate(); err != nil {
		return domain.ActionPlan{}, err
	}
	if !domain.CanTransitionActionPlan(plan.Status, target) {
		return domain.ActionPlan{}, fmt.Errorf("invalid action plan transition: %s -> %s", plan.Status, target)
	}

	plan.Status = target
	return plan, nil
}
