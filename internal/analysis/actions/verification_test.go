package actions

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type fakeStateReader struct {
	state domain.ObservedState
	err   error
}

func (f fakeStateReader) ReadState(_ context.Context, _ domain.Action) (domain.ObservedState, error) {
	return f.state, f.err
}

func verificationAction() domain.Action {
	return domain.Action{
		ID:                   "action-1",
		Type:                 domain.ActionReduceNodeGroup,
		Provider:             domain.CloudProviderAWS,
		Cluster:              "production",
		NodeGroup:            "workers",
		CurrentValue:         8,
		DesiredValue:         6,
		MonthlySavingsUSD:    100,
		AnnualizedSavingsUSD: 1200,
		Risk:                 domain.ActionRiskMedium,
		RequiresApproval:     true,
	}
}

func verificationExecution() domain.ExecutionRecord {
	completed := time.Date(2026, 8, 16, 20, 1, 0, 0, time.UTC)
	return domain.ExecutionRecord{
		ID:             "exec-1",
		IdempotencyKey: "plan-1:action-1",
		PlanID:         "plan-1",
		ActionID:       "action-1",
		Provider:       domain.CloudProviderAWS,
		Cluster:        "production",
		Status:         domain.ExecutionStatusSucceeded,
		Attempt:        1,
		CurrentValue:   8,
		DesiredValue:   6,
		StartedAt:      time.Date(2026, 8, 16, 20, 0, 0, 0, time.UTC),
		CompletedAt:    &completed,
	}
}

func newVerificationTestService(reader domain.StateReader, now time.Time) *VerificationService {
	service := NewVerificationService(reader)
	service.now = func() time.Time { return now }
	return service
}

func TestVerificationService_ShouldReturnVerified(t *testing.T) {
	service := newVerificationTestService(fakeStateReader{state: domain.ObservedState{CurrentValue: 6}}, time.Date(2026, 8, 16, 20, 5, 0, 0, time.UTC))

	result, err := service.Verify(context.Background(), verificationExecution(), verificationAction())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.VerificationStatusVerified {
		t.Fatalf("expected VERIFIED, got %q", result.Status)
	}
	if result.ExpectedValue != 6 || result.ActualValue != 6 {
		t.Fatalf("unexpected values: expected=%d actual=%d", result.ExpectedValue, result.ActualValue)
	}
	if result.ID == "" {
		t.Fatal("expected verification id")
	}
}

func TestVerificationService_ShouldReturnDrift(t *testing.T) {
	service := newVerificationTestService(fakeStateReader{state: domain.ObservedState{CurrentValue: 7}}, time.Date(2026, 8, 16, 20, 5, 0, 0, time.UTC))

	result, err := service.Verify(context.Background(), verificationExecution(), verificationAction())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.VerificationStatusDrift {
		t.Fatalf("expected DRIFT, got %q", result.Status)
	}
	if result.ActualValue != 7 {
		t.Fatalf("expected actual value 7, got %d", result.ActualValue)
	}
}

func TestVerificationService_ShouldReturnFailedWhenReaderFails(t *testing.T) {
	service := newVerificationTestService(fakeStateReader{err: errors.New("provider unavailable")}, time.Date(2026, 8, 16, 20, 5, 0, 0, time.UTC))

	result, err := service.Verify(context.Background(), verificationExecution(), verificationAction())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.VerificationStatusFailed {
		t.Fatalf("expected FAILED, got %q", result.Status)
	}
	if result.Error != "provider unavailable" {
		t.Fatalf("unexpected error %q", result.Error)
	}
}

func TestVerificationService_ShouldRejectUnsuccessfulExecution(t *testing.T) {
	execution := verificationExecution()
	execution.Status = domain.ExecutionStatusFailed
	service := newVerificationTestService(fakeStateReader{state: domain.ObservedState{CurrentValue: 6}}, time.Now().UTC())

	if _, err := service.Verify(context.Background(), execution, verificationAction()); err == nil {
		t.Fatal("expected unsuccessful execution error")
	}
}

func TestVerificationService_ShouldRejectMismatchedAction(t *testing.T) {
	action := verificationAction()
	action.ID = "different-action"
	service := newVerificationTestService(fakeStateReader{state: domain.ObservedState{CurrentValue: 6}}, time.Now().UTC())

	if _, err := service.Verify(context.Background(), verificationExecution(), action); err == nil {
		t.Fatal("expected action mismatch error")
	}
}
