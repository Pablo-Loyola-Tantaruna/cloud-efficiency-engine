package domain

import (
	"fmt"
	"strings"
	"time"
)

type AuditEventType string

const (
	AuditPlanCreated           AuditEventType = "PLAN_CREATED"
	AuditPlanSubmittedApproval AuditEventType = "PLAN_SUBMITTED_FOR_APPROVAL"
	AuditPlanApproved          AuditEventType = "PLAN_APPROVED"
	AuditPlanRejected          AuditEventType = "PLAN_REJECTED"

	AuditExecutionCreated   AuditEventType = "EXECUTION_CREATED"
	AuditExecutionStarted   AuditEventType = "EXECUTION_STARTED"
	AuditExecutionSucceeded AuditEventType = "EXECUTION_SUCCEEDED"
	AuditExecutionFailed    AuditEventType = "EXECUTION_FAILED"
	AuditExecutionRetried   AuditEventType = "EXECUTION_RETRIED"
	AuditExecutionSkipped   AuditEventType = "EXECUTION_SKIPPED"

	AuditVerificationVerified AuditEventType = "VERIFICATION_VERIFIED"
	AuditVerificationDrift    AuditEventType = "VERIFICATION_DRIFT"
	AuditVerificationFailed   AuditEventType = "VERIFICATION_FAILED"

	AuditRecoveryCreated   AuditEventType = "RECOVERY_CREATED"
	AuditRecoveryApproved  AuditEventType = "RECOVERY_APPROVED"
	AuditRecoveryExecuting AuditEventType = "RECOVERY_EXECUTING"
	AuditRecoverySucceeded AuditEventType = "RECOVERY_SUCCEEDED"
	AuditRecoveryFailed    AuditEventType = "RECOVERY_FAILED"
)

type AuditEvent struct {
	ID            string            `json:"id"`
	PlanID        string            `json:"planId"`
	ActionID      string            `json:"actionId,omitempty"`
	ExecutionID   string            `json:"executionId,omitempty"`
	Attempt       int               `json:"attempt,omitempty"`
	EventType     AuditEventType    `json:"eventType"`
	Actor         string            `json:"actor"`
	Timestamp     time.Time         `json:"timestamp"`
	Provider      CloudProvider     `json:"provider,omitempty"`
	Cluster       string            `json:"cluster,omitempty"`
	PreviousState string            `json:"previousState,omitempty"`
	NewState      string            `json:"newState,omitempty"`
	Message       string            `json:"message,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

func (e AuditEvent) Validate() error {
	if strings.TrimSpace(e.ID) == "" {
		return fmt.Errorf("audit event id must not be empty")
	}
	if strings.TrimSpace(e.PlanID) == "" {
		return fmt.Errorf("audit event plan id must not be empty")
	}
	if !validAuditEventType(e.EventType) {
		return fmt.Errorf("unsupported audit event type %q", e.EventType)
	}
	if strings.TrimSpace(e.Actor) == "" {
		return fmt.Errorf("audit event actor must not be empty")
	}
	if e.Timestamp.IsZero() {
		return fmt.Errorf("audit event timestamp must not be zero")
	}
	if e.Attempt < 0 {
		return fmt.Errorf("audit event attempt must not be negative")
	}
	if e.Provider != CloudProviderUnknown && !e.Provider.IsValid() {
		return fmt.Errorf("audit event provider must be valid")
	}
	return nil
}

func validAuditEventType(eventType AuditEventType) bool {
	switch eventType {
	case AuditPlanCreated, AuditPlanSubmittedApproval, AuditPlanApproved, AuditPlanRejected,
		AuditExecutionCreated, AuditExecutionStarted, AuditExecutionSucceeded,
		AuditExecutionFailed, AuditExecutionRetried, AuditExecutionSkipped,
		AuditVerificationVerified, AuditVerificationDrift, AuditVerificationFailed,
		AuditRecoveryCreated, AuditRecoveryApproved, AuditRecoveryExecuting,
		AuditRecoverySucceeded, AuditRecoveryFailed:
		return true
	default:
		return false
	}
}
