package actions

import (
	"fmt"
	"sort"
	"strings"

	"cloud-efficiency-engine/internal/domain"
)

// RenderPreview creates a deterministic, human-readable dry-run representation
// of an action plan. It does not execute or mutate any cloud resource.
func RenderPreview(plan domain.ActionPlan) (string, error) {
	if err := plan.Validate(); err != nil {
		return "", err
	}

	actions := append([]domain.Action(nil), plan.Actions...)
	sort.Slice(actions, func(i, j int) bool {
		if actions[i].Type != actions[j].Type {
			return actions[i].Type < actions[j].Type
		}
		return actions[i].ID < actions[j].ID
	})

	var builder strings.Builder
	fmt.Fprintf(&builder, "PLAN %s\n\n", plan.ID)
	fmt.Fprintf(&builder, "%s / %s\n\n", strings.ToUpper(string(plan.Provider)), plan.Cluster)

	for index, action := range actions {
		fmt.Fprintf(&builder, "%d. %s\n", index+1, action.Type)
		if action.NodeGroup != "" {
			fmt.Fprintf(&builder, "   node-group: %s\n", action.NodeGroup)
		}
		if action.Workload != "" {
			fmt.Fprintf(&builder, "   workload: %s\n", action.Workload)
		}
		fmt.Fprintf(&builder, "   value: %d → %d\n", action.CurrentValue, action.DesiredValue)
		fmt.Fprintf(&builder, "   savings: $%.2f/month\n", action.MonthlySavingsUSD)
		fmt.Fprintf(&builder, "   annualized: $%.2f/year\n", action.AnnualizedSavingsUSD)
		fmt.Fprintf(&builder, "   risk: %s\n", action.Risk)
		fmt.Fprintln(&builder)
	}

	fmt.Fprintln(&builder, "TOTAL")
	fmt.Fprintf(&builder, "$%.2f/month\n", plan.TotalMonthlySavingsUSD)
	fmt.Fprintf(&builder, "$%.2f/year\n", plan.TotalAnnualizedSavingsUSD)
	fmt.Fprintln(&builder)
	fmt.Fprintf(&builder, "STATUS: %s\n", plan.Status)
	fmt.Fprintf(&builder, "APPROVAL: %s\n", approvalLabel(plan.RequiresApproval))

	return builder.String(), nil
}

func approvalLabel(required bool) string {
	if required {
		return "REQUIRED"
	}
	return "NOT REQUIRED"
}
