package actions

import (
	"errors"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

func TestAuditService_ShouldRecordAndOrderExecutionEvents(t *testing.T) {
	repo := NewInMemoryAuditEventRepository()
	service := NewAuditService(repo, "operator@example.com")
	t1 := time.Date(2026, 8, 16, 20, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Minute)
	service.now = func() time.Time { return t1 }

	record := domain.ExecutionRecord{
		ID: "exec-1", IdempotencyKey: "plan-1:action-1", PlanID: "plan-1", ActionID: "action-1",
		Provider: domain.CloudProviderAWS, Cluster: "production", Status: domain.ExecutionStatusRunning,
		Attempt: 1, StartedAt: t1,
	}
	if err := service.RecordExecution(domain.AuditExecutionStarted, record, "execution started"); err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return t2 }
	record.Status = domain.ExecutionStatusSucceeded
	record.CompletedAt = &t2
	if err := service.RecordExecution(domain.AuditExecutionSucceeded, record, "ok"); err != nil {
		t.Fatal(err)
	}

	events := service.ListByExecution("exec-1")
	if len(events) != 2 {
		t.Fatalf("expected 2 audit events, got %d", len(events))
	}
	if events[0].EventType != domain.AuditExecutionStarted || events[1].EventType != domain.AuditExecutionSucceeded {
		t.Fatalf("unexpected event ordering: %#v", events)
	}
	if events[0].Actor != "operator@example.com" {
		t.Fatalf("unexpected actor %q", events[0].Actor)
	}
}

func TestAuditedExecutionService_ShouldCaptureRetry(t *testing.T) {
	executionRepo := NewInMemoryExecutionRecordStore()
	auditRepo := NewInMemoryAuditEventRepository()
	service := NewAuditedExecutionService(executionRepo, auditRepo, "system")
	plan := executionReadyPlan()

	record, created, err := service.Start(plan, plan.Actions[0])
	if err != nil || !created {
		t.Fatalf("start failed: created=%v err=%v", created, err)
	}
	record, err = service.MarkRunning(record)
	if err != nil {
		t.Fatal(err)
	}
	record, err = service.Fail(record, errors.New("provider timeout"))
	if err != nil {
		t.Fatal(err)
	}
	retry, created, err := service.Retry(plan, plan.Actions[0])
	if err != nil || !created {
		t.Fatalf("retry failed: created=%v err=%v", created, err)
	}
	if retry.Attempt != 2 {
		t.Fatalf("expected attempt 2, got %d", retry.Attempt)
	}

	events := service.AuditByPlan(plan.ID)
	if len(events) != 4 {
		t.Fatalf("expected created, started, failed and retried events, got %d", len(events))
	}
	if events[3].EventType != domain.AuditExecutionRetried {
		t.Fatalf("expected retry event, got %q", events[3].EventType)
	}
}
