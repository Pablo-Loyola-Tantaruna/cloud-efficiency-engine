package actions

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

func BuildRecoveryAction(execution domain.ExecutionRecord, action domain.Action, verification domain.VerificationResult, now time.Time) (domain.RecoveryAction, error) {
	if err := execution.Validate(); err != nil {
		return domain.RecoveryAction{}, err
	}
	if err := action.Validate(); err != nil {
		return domain.RecoveryAction{}, err
	}
	if err := verification.Validate(); err != nil {
		return domain.RecoveryAction{}, err
	}
	if action.ID != execution.ActionID || verification.ExecutionID != execution.ID {
		return domain.RecoveryAction{}, fmt.Errorf("recovery inputs do not belong to the same execution")
	}
	if verification.Status != domain.VerificationStatusDrift {
		return domain.RecoveryAction{}, fmt.Errorf("recovery requires verification DRIFT, got %s", verification.Status)
	}
	if verification.ActualValue <= 0 {
		return domain.RecoveryAction{}, fmt.Errorf("recovery requires a positive actual value")
	}
	if verification.ActualValue == action.CurrentValue {
		return domain.RecoveryAction{}, fmt.Errorf("verification drift does not provide a compensating delta")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	id := recoveryID(execution.ID, verification.ActualValue, action.CurrentValue)
	recovery := domain.RecoveryAction{
		ID:               id,
		PlanID:           execution.PlanID,
		ActionID:         execution.ActionID,
		ExecutionID:      execution.ID,
		Provider:         execution.Provider,
		Cluster:          execution.Cluster,
		Resource:         action.NodeGroup,
		FromValue:        verification.ActualValue,
		ToValue:          action.CurrentValue,
		Reason:           fmt.Sprintf("compensate execution %q after verification drift: expected %d, observed %d", execution.ID, action.DesiredValue, verification.ActualValue),
		Status:           domain.RecoveryReady,
		RequiresApproval: true,
		CreatedAt:        now,
	}
	if err := recovery.Validate(); err != nil {
		return domain.RecoveryAction{}, err
	}
	return recovery, nil
}

func recoveryID(executionID string, from, to int64) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%d", executionID, from, to)))
	return "recovery-" + hex.EncodeToString(sum[:8])
}
