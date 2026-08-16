package actions

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

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

	key := BuildIdempotencyKey(plan.ID, action.ID)
	record := domain.ExecutionRecord{
		ID: executionRecordID(key, 1), IdempotencyKey: key,
		PlanID: plan.ID, ActionID: action.ID, Provider: plan.Provider, Cluster: plan.Cluster,
		Status: domain.ExecutionStatusPending, Attempt: 1,
		CurrentValue: action.CurrentValue, DesiredValue: action.DesiredValue, StartedAt: now,
	}
	if err := record.Validate(); err != nil {
		return domain.ExecutionRecord{}, err
	}
	return record, nil
}

func RetryExecutionRecord(record domain.ExecutionRecord, now time.Time) (domain.ExecutionRecord, error) {
	if err := record.Validate(); err != nil {
		return domain.ExecutionRecord{}, err
	}
	if record.Status != domain.ExecutionStatusFailed {
		return domain.ExecutionRecord{}, fmt.Errorf("execution %q must be FAILED before retry", record.ID)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	attempt := record.Attempt + 1
	if attempt <= 0 {
		attempt = 1
	}
	key := record.IdempotencyKey
	if strings.TrimSpace(key) == "" {
		key = BuildIdempotencyKey(record.PlanID, record.ActionID)
	}
	retry := domain.ExecutionRecord{
		ID: executionRecordID(key, attempt), IdempotencyKey: key,
		PlanID: record.PlanID, ActionID: record.ActionID, Provider: record.Provider, Cluster: record.Cluster,
		Status: domain.ExecutionStatusPending, Attempt: attempt,
		CurrentValue: record.CurrentValue, DesiredValue: record.DesiredValue, StartedAt: now,
	}
	if err := retry.Validate(); err != nil {
		return domain.ExecutionRecord{}, err
	}
	return retry, nil
}

func TransitionExecution(record domain.ExecutionRecord, target domain.ExecutionStatus, now time.Time, executionErr error) (domain.ExecutionRecord, error) {
	if err := record.Validate(); err != nil {
		return domain.ExecutionRecord{}, err
	}
	if !domain.CanTransitionExecution(record.Status, target) {
		return domain.ExecutionRecord{}, fmt.Errorf("invalid execution transition: %s -> %s", record.Status, target)
	}
	if record.Attempt == 0 {
		record.Attempt = 1
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	record.Status = target
	if target == domain.ExecutionStatusSucceeded || target == domain.ExecutionStatusFailed || target == domain.ExecutionStatusSkipped {
		record.CompletedAt = &now
	}
	if target == domain.ExecutionStatusFailed {
		if executionErr == nil {
			return domain.ExecutionRecord{}, fmt.Errorf("failed execution requires an error")
		}
		record.Error = strings.TrimSpace(executionErr.Error())
	} else {
		record.Error = ""
	}
	if target == domain.ExecutionStatusSkipped && strings.TrimSpace(record.Result) == "" {
		record.Result = "skipped"
	}
	if err := record.Validate(); err != nil {
		return domain.ExecutionRecord{}, err
	}
	return record, nil
}

func BuildIdempotencyKey(planID, actionID string) string { return planID + ":" + actionID }

func executionRecordID(key string, attempt int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", key, attempt)))
	return "exec-" + hex.EncodeToString(sum[:8])
}

func idempotencyKey(planID, actionID string) string { return BuildIdempotencyKey(planID, actionID) }
