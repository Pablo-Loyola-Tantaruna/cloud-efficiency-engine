package actions

import (
	"errors"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

func executionTestRecord(status domain.ExecutionStatus) domain.ExecutionRecord {
	started := time.Date(2026, 8, 16, 19, 20, 0, 0, time.UTC)
	record := domain.ExecutionRecord{
		ID: "exec-1", IdempotencyKey: "plan-1:action-1", PlanID: "plan-1", ActionID: "action-1",
		Provider: domain.CloudProviderAWS, Cluster: "production", Attempt: 1,
		CurrentValue: 8, DesiredValue: 6, Status: status, StartedAt: started,
	}
	if status == domain.ExecutionStatusSucceeded || status == domain.ExecutionStatusFailed || status == domain.ExecutionStatusSkipped {
		completed := started.Add(time.Minute)
		record.CompletedAt = &completed
	}
	if status == domain.ExecutionStatusFailed {
		record.Error = "cloud timeout"
	}
	if status == domain.ExecutionStatusSkipped {
		record.Result = "already at desired capacity"
	}
	return record
}

func executionTestPlan() domain.ActionPlan {
	return domain.ActionPlan{
		ID: "plan-1", Provider: domain.CloudProviderAWS, Cluster: "production",
		Status: domain.ActionPlanReadyToApply, RequiresApproval: true,
		TotalMonthlySavingsUSD: 100, TotalAnnualizedSavingsUSD: 1200,
		Actions: []domain.Action{{
			ID: "action-1", Type: domain.ActionReduceNodeGroup, Provider: domain.CloudProviderAWS, Cluster: "production",
			CurrentValue: 8, DesiredValue: 6, MonthlySavingsUSD: 100, AnnualizedSavingsUSD: 1200,
			Risk: domain.ActionRiskMedium, RequiresApproval: true,
		}},
	}
}

func TestNewExecutionRecord_ShouldCreateDeterministicIdempotencyRecord(t *testing.T) {
	plan := executionTestPlan()
	now := time.Date(2026, 8, 16, 19, 20, 0, 0, time.UTC)
	record, err := NewExecutionRecord(plan, plan.Actions[0], now)
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != domain.ExecutionStatusPending {
		t.Fatalf("expected pending status, got %q", record.Status)
	}
	if record.IdempotencyKey != "plan-1:action-1" {
		t.Fatalf("unexpected idempotency key %q", record.IdempotencyKey)
	}
	if record.Attempt != 1 || record.CurrentValue != 8 || record.DesiredValue != 6 {
		t.Fatalf("unexpected execution metadata: %+v", record)
	}
	recordAgain, err := NewExecutionRecord(plan, plan.Actions[0], now)
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != recordAgain.ID {
		t.Fatalf("expected deterministic execution id, got %q and %q", record.ID, recordAgain.ID)
	}
}

func TestNewExecutionRecord_ShouldRejectPlanNotReady(t *testing.T) {
	plan := executionTestPlan()
	plan.Status = domain.ActionPlanApproved
	if _, err := NewExecutionRecord(plan, plan.Actions[0], time.Time{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestTransitionExecution_ShouldCompleteSuccess(t *testing.T) {
	record := executionTestRecord(domain.ExecutionStatusPending)
	now := record.StartedAt.Add(time.Minute)
	running, err := TransitionExecution(record, domain.ExecutionStatusRunning, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	done, err := TransitionExecution(running, domain.ExecutionStatusSucceeded, now.Add(time.Minute), nil)
	if err != nil {
		t.Fatal(err)
	}
	if done.CompletedAt == nil || done.Error != "" {
		t.Fatalf("unexpected completed record: %+v", done)
	}
}

func TestTransitionExecution_ShouldRequireErrorOnFailure(t *testing.T) {
	record := executionTestRecord(domain.ExecutionStatusRunning)
	_, err := TransitionExecution(record, domain.ExecutionStatusFailed, time.Now(), nil)
	if err == nil {
		t.Fatal("expected error when failure has no cause")
	}
	failed, err := TransitionExecution(record, domain.ExecutionStatusFailed, time.Now(), errors.New("cloud timeout"))
	if err != nil {
		t.Fatal(err)
	}
	if failed.Error != "cloud timeout" {
		t.Fatalf("unexpected error %q", failed.Error)
	}
}

func TestTransitionExecution_ShouldRejectTerminalTransition(t *testing.T) {
	record := executionTestRecord(domain.ExecutionStatusSucceeded)
	if _, err := TransitionExecution(record, domain.ExecutionStatusRunning, time.Now(), nil); err == nil {
		t.Fatal("expected terminal transition error")
	}
}

func TestTransitionExecution_ShouldAllowSkipFromRunning(t *testing.T) {
	record := executionTestRecord(domain.ExecutionStatusRunning)
	skipped, err := TransitionExecution(record, domain.ExecutionStatusSkipped, time.Now(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if skipped.CompletedAt == nil {
		t.Fatal("expected completion timestamp")
	}
}

func TestRetryExecutionRecord_ShouldIncrementAttempt(t *testing.T) {
	failed := executionTestRecord(domain.ExecutionStatusFailed)
	retry, err := RetryExecutionRecord(failed, failed.StartedAt.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if retry.Attempt != 2 {
		t.Fatalf("expected attempt 2, got %d", retry.Attempt)
	}
	if retry.IdempotencyKey != failed.IdempotencyKey {
		t.Fatal("expected stable idempotency key")
	}
	if retry.ID == failed.ID {
		t.Fatal("expected distinct execution id")
	}
	if retry.Status != domain.ExecutionStatusPending {
		t.Fatalf("expected pending retry, got %q", retry.Status)
	}
}

func TestRetryExecutionRecord_ShouldRejectNonFailed(t *testing.T) {
	if _, err := RetryExecutionRecord(executionTestRecord(domain.ExecutionStatusRunning), time.Now()); err == nil {
		t.Fatal("expected retry rejection")
	}
}
