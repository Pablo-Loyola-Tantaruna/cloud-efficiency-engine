package domain

import "fmt"

type ActionType string

const (
	ActionReduceNodeGroup         ActionType = "REDUCE_NODE_GROUP"
	ActionRightsizeWorkloadCPU    ActionType = "RIGHTSIZE_WORKLOAD_CPU"
	ActionRightsizeWorkloadMemory ActionType = "RIGHTSIZE_WORKLOAD_MEMORY"
)

type ActionRisk string

const (
	ActionRiskLow    ActionRisk = "LOW"
	ActionRiskMedium ActionRisk = "MEDIUM"
	ActionRiskHigh   ActionRisk = "HIGH"
)

type Action struct {
	ID        string        `json:"id"`
	Type      ActionType    `json:"type"`
	Provider  CloudProvider `json:"provider"`
	Cluster   string        `json:"cluster"`
	NodeGroup string        `json:"nodeGroup,omitempty"`
	Workload  string        `json:"workload,omitempty"`

	CurrentValue int64 `json:"currentValue"`
	DesiredValue int64 `json:"desiredValue"`

	MonthlySavingsUSD    float64 `json:"monthlySavingsUsd"`
	AnnualizedSavingsUSD float64 `json:"annualizedSavingsUsd"`

	Risk             ActionRisk `json:"risk"`
	RequiresApproval bool       `json:"requiresApproval"`
}

func (a Action) Validate() error {
	if a.ID == "" {
		return fmt.Errorf("action id must not be empty")
	}
	if a.Provider == CloudProviderUnknown {
		return fmt.Errorf("action provider must be known")
	}
	if a.Cluster == "" {
		return fmt.Errorf("action cluster must not be empty")
	}
	if a.CurrentValue <= 0 || a.DesiredValue <= 0 {
		return fmt.Errorf("action values must be greater than zero")
	}
	if a.DesiredValue >= a.CurrentValue {
		return fmt.Errorf("action desired value must be lower than current value")
	}
	if a.MonthlySavingsUSD <= 0 || a.AnnualizedSavingsUSD <= 0 {
		return fmt.Errorf("action savings must be greater than zero")
	}
	if !a.RequiresApproval {
		return fmt.Errorf("action requires explicit approval")
	}
	return nil
}
