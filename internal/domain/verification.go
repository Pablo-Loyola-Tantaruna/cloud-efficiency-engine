package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type VerificationStatus string

const (
	VerificationStatusVerified VerificationStatus = "VERIFIED"
	VerificationStatusDrift    VerificationStatus = "DRIFT"
	VerificationStatusFailed   VerificationStatus = "FAILED"
)

type ObservedState struct {
	CurrentValue int64 `json:"currentValue"`
}

type StateReader interface {
	ReadState(ctx context.Context, action Action) (ObservedState, error)
}

type VerificationResult struct {
	ID            string             `json:"id"`
	PlanID        string             `json:"planId"`
	ActionID      string             `json:"actionId"`
	ExecutionID   string             `json:"executionId"`
	Attempt       int                `json:"attempt"`
	Provider      CloudProvider      `json:"provider"`
	Cluster       string             `json:"cluster"`
	Status        VerificationStatus `json:"status"`
	ExpectedValue int64              `json:"expectedValue"`
	ActualValue   int64              `json:"actualValue"`
	VerifiedAt    time.Time          `json:"verifiedAt"`
	Message       string             `json:"message"`
	Error         string             `json:"error,omitempty"`
}

func (r VerificationResult) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("verification result id must not be empty")
	}
	if strings.TrimSpace(r.PlanID) == "" {
		return fmt.Errorf("verification plan id must not be empty")
	}
	if strings.TrimSpace(r.ActionID) == "" {
		return fmt.Errorf("verification action id must not be empty")
	}
	if strings.TrimSpace(r.ExecutionID) == "" {
		return fmt.Errorf("verification execution id must not be empty")
	}
	if r.Attempt <= 0 {
		return fmt.Errorf("verification attempt must be greater than zero")
	}
	if !r.Provider.IsValid() {
		return fmt.Errorf("verification provider must be valid")
	}
	if strings.TrimSpace(r.Cluster) == "" {
		return fmt.Errorf("verification cluster must not be empty")
	}
	if !validVerificationStatus(r.Status) {
		return fmt.Errorf("unsupported verification status %q", r.Status)
	}
	if r.ExpectedValue <= 0 || r.ActualValue <= 0 {
		return fmt.Errorf("verification values must be greater than zero")
	}
	if r.VerifiedAt.IsZero() {
		return fmt.Errorf("verification timestamp must not be zero")
	}
	if r.Status == VerificationStatusFailed && strings.TrimSpace(r.Error) == "" {
		return fmt.Errorf("failed verification must contain an error")
	}
	if strings.TrimSpace(r.Message) == "" {
		return fmt.Errorf("verification message must not be empty")
	}
	return nil
}

func validVerificationStatus(status VerificationStatus) bool {
	switch status {
	case VerificationStatusVerified, VerificationStatusDrift, VerificationStatusFailed:
		return true
	default:
		return false
	}
}
