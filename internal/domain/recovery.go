package domain

import (
	"fmt"
	"strings"
	"time"
)

type RecoveryStatus string

const (
	RecoveryReady       RecoveryStatus = "READY"
	RecoveryExecuting   RecoveryStatus = "EXECUTING"
	RecoverySucceeded   RecoveryStatus = "SUCCEEDED"
	RecoveryFailed      RecoveryStatus = "FAILED"
	RecoveryUnavailable RecoveryStatus = "UNAVAILABLE"
)

type RecoveryAction struct {
	ID               string         `json:"id"`
	PlanID           string         `json:"planId"`
	ActionID         string         `json:"actionId"`
	ExecutionID      string         `json:"executionId"`
	Provider         CloudProvider  `json:"provider"`
	Cluster          string         `json:"cluster"`
	Resource         string         `json:"resource"`
	FromValue        int64          `json:"fromValue"`
	ToValue          int64          `json:"toValue"`
	Reason           string         `json:"reason"`
	Status           RecoveryStatus `json:"status"`
	RequiresApproval bool           `json:"requiresApproval"`
	CreatedAt        time.Time      `json:"createdAt"`
}

func (r RecoveryAction) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("recovery action id must not be empty")
	}
	if strings.TrimSpace(r.PlanID) == "" {
		return fmt.Errorf("recovery plan id must not be empty")
	}
	if strings.TrimSpace(r.ActionID) == "" {
		return fmt.Errorf("recovery action id reference must not be empty")
	}
	if strings.TrimSpace(r.ExecutionID) == "" {
		return fmt.Errorf("recovery execution id must not be empty")
	}
	if !r.Provider.IsValid() {
		return fmt.Errorf("recovery provider must be valid")
	}
	if strings.TrimSpace(r.Cluster) == "" {
		return fmt.Errorf("recovery cluster must not be empty")
	}
	if strings.TrimSpace(r.Resource) == "" {
		return fmt.Errorf("recovery resource must not be empty")
	}
	if r.FromValue <= 0 || r.ToValue <= 0 {
		return fmt.Errorf("recovery values must be greater than zero")
	}
	if r.FromValue == r.ToValue {
		return fmt.Errorf("recovery values must differ")
	}
	if strings.TrimSpace(r.Reason) == "" {
		return fmt.Errorf("recovery reason must not be empty")
	}
	if !validRecoveryStatus(r.Status) {
		return fmt.Errorf("unsupported recovery status %q", r.Status)
	}
	if r.CreatedAt.IsZero() {
		return fmt.Errorf("recovery createdAt must not be zero")
	}
	if !r.RequiresApproval {
		return fmt.Errorf("recovery requires explicit approval")
	}
	return nil
}

func validRecoveryStatus(status RecoveryStatus) bool {
	switch status {
	case RecoveryReady, RecoveryExecuting, RecoverySucceeded, RecoveryFailed, RecoveryUnavailable:
		return true
	default:
		return false
	}
}
