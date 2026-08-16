package actions

import (
	"fmt"
	"sort"

	"cloud-efficiency-engine/internal/domain"
)

type ExecutionOperation struct {
	ActionID       string               `json:"actionId"`
	Type           domain.ActionType    `json:"type"`
	Provider       domain.CloudProvider `json:"provider"`
	Cluster        string               `json:"cluster"`
	NodeGroup      string               `json:"nodeGroup,omitempty"`
	Workload       string               `json:"workload,omitempty"`
	CurrentValue   int64                `json:"currentValue"`
	DesiredValue   int64                `json:"desiredValue"`
	MonthlySavings float64              `json:"monthlySavingsUsd"`
}

type ExecutionPlan struct {
	PlanID       string               `json:"planId"`
	Mode         string               `json:"mode"`
	Provider     domain.CloudProvider `json:"provider"`
	Cluster      string               `json:"cluster"`
	Operations   []ExecutionOperation `json:"operations"`
	TotalSavings float64              `json:"totalMonthlySavingsUsd"`
}

const ExecutionModeDryRun = "DRY_RUN"

func BuildDryRunExecution(plan domain.ActionPlan) (ExecutionPlan, error) {
	if err := plan.Validate(); err != nil {
		return ExecutionPlan{}, err
	}
	if plan.Status != domain.ActionPlanReadyToApply {
		return ExecutionPlan{}, fmt.Errorf("action plan %q must be READY_TO_APPLY for execution preview", plan.ID)
	}

	operations := make([]ExecutionOperation, 0, len(plan.Actions))
	for _, action := range plan.Actions {
		operations = append(operations, ExecutionOperation{
			ActionID:       action.ID,
			Type:           action.Type,
			Provider:       action.Provider,
			Cluster:        action.Cluster,
			NodeGroup:      action.NodeGroup,
			Workload:       action.Workload,
			CurrentValue:   action.CurrentValue,
			DesiredValue:   action.DesiredValue,
			MonthlySavings: action.MonthlySavingsUSD,
		})
	}

	sort.Slice(operations, func(i, j int) bool {
		return operations[i].ActionID < operations[j].ActionID
	})

	return ExecutionPlan{
		PlanID:       plan.ID,
		Mode:         ExecutionModeDryRun,
		Provider:     plan.Provider,
		Cluster:      plan.Cluster,
		Operations:   operations,
		TotalSavings: plan.TotalMonthlySavingsUSD,
	}, nil
}
