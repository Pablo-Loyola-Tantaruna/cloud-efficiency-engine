package actions

import "cloud-efficiency-engine/internal/domain"

type ActionPlanRepository interface {
	GetActionPlanByID(id string) (domain.ActionPlan, bool, error)
	CreateActionPlanIfAbsent(plan domain.ActionPlan) (bool, error)
	UpdateActionPlan(plan domain.ActionPlan) error
}

type ActionApprovalRepository interface {
	GetApprovalByPlanID(planID string) (domain.ActionApproval, bool, error)
	SaveApproval(approval domain.ActionApproval) error
}

type RecoveryActionRepository interface {
	GetRecoveryByID(id string) (domain.RecoveryAction, bool, error)
	ListRecoveryByPlan(planID string) ([]domain.RecoveryAction, error)
	SaveRecovery(action domain.RecoveryAction) error
}
