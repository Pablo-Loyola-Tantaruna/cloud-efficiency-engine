package domain

import (
	"fmt"
	"strings"
	"time"
)

type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "PENDING"
	ExecutionStatusRunning   ExecutionStatus = "RUNNING"
	ExecutionStatusSucceeded ExecutionStatus = "SUCCEEDED"
	ExecutionStatusFailed    ExecutionStatus = "FAILED"
	ExecutionStatusSkipped   ExecutionStatus = "SKIPPED"
)

type ExecutionRecord struct {
	ID             string          `json:"id"`
	IdempotencyKey string          `json:"idempotencyKey"`
	PlanID         string          `json:"planId"`
	ActionID       string          `json:"actionId"`
	Provider       CloudProvider   `json:"provider"`
	Cluster        string          `json:"cluster"`
	Status         ExecutionStatus `json:"status"`
	Attempt        int             `json:"attempt"`
	CurrentValue   int64           `json:"currentValue"`
	DesiredValue   int64           `json:"desiredValue"`
	StartedAt      time.Time       `json:"startedAt"`
	CompletedAt    *time.Time      `json:"completedAt,omitempty"`
	Error          string          `json:"error,omitempty"`
	Result         string          `json:"result,omitempty"`
}

func (r ExecutionRecord) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("execution record id must not be empty")
	}
	if strings.TrimSpace(r.IdempotencyKey) == "" {
		return fmt.Errorf("idempotency key must not be empty")
	}
	if strings.TrimSpace(r.PlanID) == "" {
		return fmt.Errorf("execution plan id must not be empty")
	}
	if strings.TrimSpace(r.ActionID) == "" {
		return fmt.Errorf("execution action id must not be empty")
	}
	if !r.Provider.IsValid() {
		return fmt.Errorf("execution provider must be valid")
	}
	if strings.TrimSpace(r.Cluster) == "" {
		return fmt.Errorf("execution cluster must not be empty")
	}
	if !validExecutionStatus(r.Status) {
		return fmt.Errorf("unsupported execution status %q", r.Status)
	}
	if r.StartedAt.IsZero() {
		return fmt.Errorf("execution startedAt must not be zero")
	}
	if r.Status == ExecutionStatusSucceeded || r.Status == ExecutionStatusFailed || r.Status == ExecutionStatusSkipped {
		if r.CompletedAt != nil && r.CompletedAt.IsZero() {
			return fmt.Errorf("completedAt must not be zero when present")
		}
	}
	if r.Status == ExecutionStatusFailed && strings.TrimSpace(r.Error) == "" {
		return fmt.Errorf("failed execution must contain an error")
	}
	return nil
}

func validExecutionStatus(status ExecutionStatus) bool {
	switch status {
	case ExecutionStatusPending, ExecutionStatusRunning, ExecutionStatusSucceeded, ExecutionStatusFailed, ExecutionStatusSkipped:
		return true
	default:
		return false
	}
}

func CanTransitionExecution(from, to ExecutionStatus) bool {
	switch from {
	case ExecutionStatusPending:
		return to == ExecutionStatusRunning || to == ExecutionStatusFailed || to == ExecutionStatusSkipped
	case ExecutionStatusRunning:
		return to == ExecutionStatusSucceeded || to == ExecutionStatusFailed || to == ExecutionStatusSkipped
	case ExecutionStatusSucceeded, ExecutionStatusFailed, ExecutionStatusSkipped:
		return false
	default:
		return false
	}
}
