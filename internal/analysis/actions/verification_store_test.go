package actions

import (
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

func TestInMemoryVerificationResultRepository_ShouldPersistAndQuery(t *testing.T) {
	repository := NewInMemoryVerificationResultRepository()
	result := domain.VerificationResult{
		ID:            "verify-1",
		PlanID:        "plan-1",
		ActionID:      "action-1",
		ExecutionID:   "exec-1",
		Attempt:       1,
		Provider:      domain.CloudProviderAWS,
		Cluster:       "production",
		Status:        domain.VerificationStatusVerified,
		ExpectedValue: 6,
		ActualValue:   6,
		VerifiedAt:    time.Date(2026, 8, 16, 20, 10, 0, 0, time.UTC),
		Message:       "verified desired value 6",
	}
	if err := repository.Save(result); err != nil {
		t.Fatal(err)
	}

	got, ok := repository.GetByExecutionID("exec-1")
	if !ok {
		t.Fatal("expected verification result")
	}
	if got.ID != result.ID {
		t.Fatalf("expected id %q, got %q", result.ID, got.ID)
	}

	results := repository.ListByPlan("plan-1")
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
}

func TestInMemoryVerificationResultRepository_ShouldRejectConflictingExecution(t *testing.T) {
	repository := NewInMemoryVerificationResultRepository()
	base := domain.VerificationResult{
		ID:            "verify-1",
		PlanID:        "plan-1",
		ActionID:      "action-1",
		ExecutionID:   "exec-1",
		Attempt:       1,
		Provider:      domain.CloudProviderAWS,
		Cluster:       "production",
		Status:        domain.VerificationStatusVerified,
		ExpectedValue: 6,
		ActualValue:   6,
		VerifiedAt:    time.Now().UTC(),
		Message:       "verified",
	}
	if err := repository.Save(base); err != nil {
		t.Fatal(err)
	}
	conflict := base
	conflict.ID = "verify-2"
	if err := repository.Save(conflict); err == nil {
		t.Fatal("expected conflicting verification error")
	}
}
