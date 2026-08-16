package domain

import (
	"context"
	"fmt"
	"strings"
)

type ExecutionResultStatus string

const (
	ExecutionResultSucceeded ExecutionResultStatus = "SUCCEEDED"
	ExecutionResultFailed    ExecutionResultStatus = "FAILED"
)

type ExecutionResult struct {
	Status       ExecutionResultStatus
	ExecutionID  string
	Provider     CloudProvider
	Cluster      string
	ActionID     string
	BeforeValue  int64
	DesiredValue int64
	Message      string
	Error        string
}

func (r ExecutionResult) Validate() error {
	if !validExecutionResultStatus(r.Status) {
		return fmt.Errorf("unsupported execution result status %q", r.Status)
	}
	if strings.TrimSpace(r.ExecutionID) == "" {
		return fmt.Errorf("execution result execution id must not be empty")
	}
	if strings.TrimSpace(r.ActionID) == "" {
		return fmt.Errorf("execution result action id must not be empty")
	}
	if !r.Provider.IsValid() {
		return fmt.Errorf("execution result provider must be valid")
	}
	if strings.TrimSpace(r.Cluster) == "" {
		return fmt.Errorf("execution result cluster must not be empty")
	}
	if r.BeforeValue <= 0 || r.DesiredValue <= 0 {
		return fmt.Errorf("execution result values must be greater than zero")
	}
	if r.Status == ExecutionResultFailed && strings.TrimSpace(r.Error) == "" {
		return fmt.Errorf("failed execution result must contain an error")
	}
	return nil
}

func validExecutionResultStatus(status ExecutionResultStatus) bool {
	switch status {
	case ExecutionResultSucceeded, ExecutionResultFailed:
		return true
	default:
		return false
	}
}

// ProviderExecutor is the only contract required by the execution orchestration layer.
// Provider implementations translate the domain action into real provider API calls.
type ProviderExecutor interface {
	Execute(ctx context.Context, action Action, execution ExecutionRecord) (ExecutionResult, error)
}

// ExecutorResolver isolates provider selection from orchestration.
type ExecutorResolver interface {
	Resolve(provider CloudProvider) (ProviderExecutor, error)
}
