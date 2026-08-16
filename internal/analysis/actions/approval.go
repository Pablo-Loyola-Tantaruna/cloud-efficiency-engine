package actions

import (
	"fmt"
	"strings"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

func ApproveActionPlan(plan domain.ActionPlan, approvedBy, comment string, approvedAt time.Time) (domain.ActionPlan, domain.ActionApproval, error) {
	if err := plan.Validate(); err != nil {
		return domain.ActionPlan{}, domain.ActionApproval{}, err
	}
	if plan.Status != domain.ActionPlanPendingApproval {
		return domain.ActionPlan{}, domain.ActionApproval{}, fmt.Errorf("action plan %q must be pending approval", plan.ID)
	}
	if strings.TrimSpace(approvedBy) == "" {
		return domain.ActionPlan{}, domain.ActionApproval{}, fmt.Errorf("approved by must not be empty")
	}
	if approvedAt.IsZero() {
		approvedAt = time.Now().UTC()
	}
	plan.Status = domain.ActionPlanApproved
	approval := domain.ActionApproval{
		PlanID:     plan.ID,
		ApprovedBy: strings.TrimSpace(approvedBy),
		ApprovedAt: approvedAt.UTC(),
		Comment:    strings.TrimSpace(comment),
	}
	return plan, approval, nil
}
