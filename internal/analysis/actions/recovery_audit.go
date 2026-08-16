package actions

import (
	"fmt"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

func RecordRecoveryCreated(audit *AuditService, recovery domain.RecoveryAction) error {
	if audit == nil {
		return fmt.Errorf("audit service must not be nil")
	}
	if err := recovery.Validate(); err != nil {
		return err
	}
	return audit.Record(domain.AuditEvent{
		ID: fmt.Sprintf("audit-recovery-created-%s", recovery.ID), PlanID: recovery.PlanID,
		ActionID: recovery.ActionID, ExecutionID: recovery.ExecutionID, EventType: domain.AuditRecoveryCreated,
		Provider: recovery.Provider, Cluster: recovery.Cluster, NewState: string(recovery.Status),
		Message: recovery.Reason, Timestamp: recovery.CreatedAt,
	})
}

func RecordRecoveryState(audit *AuditService, recovery domain.RecoveryAction, eventType domain.AuditEventType, previousState, message string, now time.Time) error {
	if audit == nil {
		return fmt.Errorf("audit service must not be nil")
	}
	if err := recovery.Validate(); err != nil {
		return err
	}
	return audit.Record(domain.AuditEvent{
		ID:     fmt.Sprintf("audit-recovery-%s-%s-%s", recovery.ID, eventType, now.UTC().Format(time.RFC3339Nano)),
		PlanID: recovery.PlanID, ActionID: recovery.ActionID, ExecutionID: recovery.ExecutionID,
		EventType: eventType, Provider: recovery.Provider, Cluster: recovery.Cluster,
		PreviousState: previousState, NewState: string(recovery.Status), Message: message,
		Timestamp: now.UTC(),
	})
}
