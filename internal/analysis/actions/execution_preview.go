package actions

import (
	"fmt"
	"sort"
	"strings"

	"cloud-efficiency-engine/internal/domain"
)

type ExecutionPreview struct {
	PlanID                    string               `json:"planId"`
	Mode                      string               `json:"mode"`
	Provider                  domain.CloudProvider `json:"provider"`
	Cluster                   string               `json:"cluster"`
	Changes                   []ProviderChange     `json:"changes"`
	TotalMonthlySavingsUSD    float64              `json:"totalMonthlySavingsUsd"`
	TotalAnnualizedSavingsUSD float64              `json:"totalAnnualizedSavingsUsd"`
	HighestRisk               domain.ActionRisk    `json:"highestRisk"`
	ApprovalRequired          bool                 `json:"approvalRequired"`
	Diff                      string               `json:"diff"`
}

// BuildExecutionPreview combines provider-specific dry-run changes with a
// human-readable diff. It never contacts or mutates a cloud provider.
func BuildExecutionPreview(plan domain.ActionPlan) (ExecutionPreview, error) {
	execution, err := BuildDryRunExecution(plan)
	if err != nil {
		return ExecutionPreview{}, err
	}

	changes, err := RenderExecutionProviderChanges(execution)
	if err != nil {
		return ExecutionPreview{}, err
	}

	sort.Slice(changes, func(i, j int) bool { return changes[i].ActionID < changes[j].ActionID })

	highestRisk := domain.ActionRiskLow
	for _, action := range plan.Actions {
		if riskRank(action.Risk) > riskRank(highestRisk) {
			highestRisk = action.Risk
		}
	}

	return ExecutionPreview{
		PlanID:                    plan.ID,
		Mode:                      execution.Mode,
		Provider:                  execution.Provider,
		Cluster:                   execution.Cluster,
		Changes:                   changes,
		TotalMonthlySavingsUSD:    plan.TotalMonthlySavingsUSD,
		TotalAnnualizedSavingsUSD: plan.TotalAnnualizedSavingsUSD,
		HighestRisk:               highestRisk,
		ApprovalRequired:          plan.RequiresApproval,
		Diff:                      renderExecutionDiff(plan, changes),
	}, nil
}

func renderExecutionDiff(plan domain.ActionPlan, changes []ProviderChange) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "PLAN %s\n", plan.ID)
	fmt.Fprintf(&builder, "PROVIDER: %s\n", strings.ToUpper(string(plan.Provider)))
	fmt.Fprintf(&builder, "CLUSTER: %s\n", plan.Cluster)
	fmt.Fprintf(&builder, "MODE: %s\n\n", ExecutionModeDryRun)

	for index, change := range changes {
		fmt.Fprintf(&builder, "[%d] %s\n", index+1, change.Operation)
		fmt.Fprintf(&builder, "  target: %s\n", change.Target)
		fmt.Fprintf(&builder, "- value: %d\n", change.CurrentValue)
		fmt.Fprintf(&builder, "+ value: %d\n", change.DesiredValue)
		fmt.Fprintf(&builder, "  command: %s\n", change.Command)
		fmt.Fprintln(&builder)
	}

	fmt.Fprintf(&builder, "TOTAL: $%.2f/month ($%.2f/year)\n", plan.TotalMonthlySavingsUSD, plan.TotalAnnualizedSavingsUSD)
	fmt.Fprintf(&builder, "APPROVAL: %s\n", approvalLabel(plan.RequiresApproval))
	return builder.String()
}

func riskRank(risk domain.ActionRisk) int {
	switch risk {
	case domain.ActionRiskHigh:
		return 3
	case domain.ActionRiskMedium:
		return 2
	case domain.ActionRiskLow:
		return 1
	default:
		return 0
	}
}
