package actions

import (
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

func TestExecutionHistoryService_ShouldReturnOrderedAttempts(t *testing.T) {
	store := NewInMemoryExecutionRecordStore()
	now := time.Date(2026, 8, 16, 20, 0, 0, 0, time.UTC)
	for _, record := range []domain.ExecutionRecord{
		{ID: "exec-2", IdempotencyKey: "plan-1:action-1", PlanID: "plan-1", ActionID: "action-1", Provider: domain.CloudProviderAWS, Cluster: "production", Status: domain.ExecutionStatusFailed, Attempt: 2, CurrentValue: 8, DesiredValue: 6, StartedAt: now.Add(2 * time.Minute), CompletedAt: ptrTime(now.Add(3 * time.Minute)), Error: "timeout"},
		{ID: "exec-1", IdempotencyKey: "plan-1:action-1", PlanID: "plan-1", ActionID: "action-1", Provider: domain.CloudProviderAWS, Cluster: "production", Status: domain.ExecutionStatusFailed, Attempt: 1, CurrentValue: 8, DesiredValue: 6, StartedAt: now, CompletedAt: ptrTime(now.Add(time.Minute)), Error: "connection refused"},
		{ID: "exec-3", IdempotencyKey: "plan-1:action-1", PlanID: "plan-1", ActionID: "action-1", Provider: domain.CloudProviderAWS, Cluster: "production", Status: domain.ExecutionStatusSucceeded, Attempt: 3, CurrentValue: 8, DesiredValue: 6, StartedAt: now.Add(4 * time.Minute), CompletedAt: ptrTime(now.Add(5 * time.Minute)), Result: "verified"},
	} {
		if _, err := store.CreateIfAbsent(record); err != nil {
			t.Fatal(err)
		}
	}

	history, err := NewExecutionHistoryService(store).Get("plan-1", "action-1")
	if err != nil {
		t.Fatal(err)
	}
	if history.TotalAttempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", history.TotalAttempts)
	}
	if history.Attempts[0].Attempt != 1 || history.Attempts[1].Attempt != 2 || history.Attempts[2].Attempt != 3 {
		t.Fatalf("unexpected attempt order: %#v", history.Attempts)
	}
	if history.Latest.Status != domain.ExecutionStatusSucceeded {
		t.Fatalf("expected latest attempt to succeed, got %q", history.Latest.Status)
	}
}

func ptrTime(value time.Time) *time.Time { return &value }
