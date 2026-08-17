package actions

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

func BuildIdempotencyKey(planID, actionID string) string {
	return strings.TrimSpace(planID) + ":" + strings.TrimSpace(actionID)
}

func NewExecutionRecord(plan domain.ActionPlan, action domain.Action, now time.Time) (domain.ExecutionRecord, error) {
	if err := plan.Validate(); err != nil {
		return domain.ExecutionRecord{}, err
	}
	if err := action.Validate(); err != nil {
		return domain.ExecutionRecord{}, err
	}
	if plan.Status != domain.ActionPlanReadyToApply {
		return domain.ExecutionRecord{}, fmt.Errorf("action plan %q must be READY_TO_APPLY", plan.ID)
	}
	if action.Provider != plan.Provider || action.Cluster != plan.Cluster {
		return domain.ExecutionRecord{}, fmt.Errorf("action %q does not belong to plan %q", action.ID, plan.ID)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	return newExecutionRecord(plan, action, 1, now), nil
}

func newExecutionRecord(plan domain.ActionPlan, action domain.Action, attempt int, now time.Time) domain.ExecutionRecord {
	key := BuildIdempotencyKey(plan.ID, action.ID)
	return domain.ExecutionRecord{
		ID:             executionRecordID(key, attempt),
		IdempotencyKey: key,
		PlanID:         plan.ID,
		ActionID:       action.ID,
		Provider:       plan.Provider,
		Cluster:        plan.Cluster,
		Attempt:        attempt,
		Status:         domain.ExecutionStatusPending,
		CurrentValue:   action.CurrentValue,
		DesiredValue:   action.DesiredValue,
		StartedAt:      now,
	}
}

func RetryExecutionRecord(previous domain.ExecutionRecord, now time.Time) (domain.ExecutionRecord, error) {
	if err := previous.Validate(); err != nil {
		return domain.ExecutionRecord{}, err
	}
	if previous.Status != domain.ExecutionStatusFailed {
		return domain.ExecutionRecord{}, fmt.Errorf("execution %q must be FAILED before retry", previous.ID)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	return domain.ExecutionRecord{
		ID:             executionRecordID(previous.IdempotencyKey, previous.Attempt+1),
		IdempotencyKey: previous.IdempotencyKey,
		PlanID:         previous.PlanID,
		ActionID:       previous.ActionID,
		Provider:       previous.Provider,
		Cluster:        previous.Cluster,
		Attempt:        previous.Attempt + 1,
		Status:         domain.ExecutionStatusPending,
		CurrentValue:   previous.CurrentValue,
		DesiredValue:   previous.DesiredValue,
		StartedAt:      now,
	}, nil
}

func TransitionExecution(record domain.ExecutionRecord, target domain.ExecutionStatus, now time.Time, executionErr error) (domain.ExecutionRecord, error) {
	if err := record.Validate(); err != nil {
		return domain.ExecutionRecord{}, err
	}
	if !domain.CanTransitionExecution(record.Status, target) {
		return domain.ExecutionRecord{}, fmt.Errorf("invalid execution transition: %s -> %s", record.Status, target)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	record.Status = target
	record.CompletedAt = nil
	record.Result = ""
	record.Error = ""

	if target == domain.ExecutionStatusSucceeded || target == domain.ExecutionStatusFailed || target == domain.ExecutionStatusSkipped {
		record.CompletedAt = &now
	}
	if target == domain.ExecutionStatusFailed {
		if executionErr == nil {
			return domain.ExecutionRecord{}, fmt.Errorf("failed execution requires an error")
		}
		record.Error = strings.TrimSpace(executionErr.Error())
	}

	if err := record.Validate(); err != nil {
		return domain.ExecutionRecord{}, err
	}
	return record, nil
}

func executionRecordID(key string, attempt int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", key, attempt)))
	return "exec-" + hex.EncodeToString(sum[:8])
}
