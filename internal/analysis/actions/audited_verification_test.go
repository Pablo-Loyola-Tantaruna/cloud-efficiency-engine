package actions

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

func TestAuditedVerificationService_ShouldPersistAndAuditDrift(t *testing.T) {
	verification := newVerificationTestService(fakeStateReader{state: domain.ObservedState{CurrentValue: 7}}, time.Date(2026, 8, 16, 20, 15, 0, 0, time.UTC))
	results := NewInMemoryVerificationResultRepository()
	auditRepository := NewInMemoryAuditEventRepository()
	audit := NewAuditService(auditRepository, "system")
	service := NewAuditedVerificationService(verification, results, audit)

	result, err := service.Verify(context.Background(), verificationExecution(), verificationAction())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.VerificationStatusDrift {
		t.Fatalf("expected DRIFT, got %q", result.Status)
	}

	stored, ok := results.GetByExecutionID("exec-1")
	if !ok || stored.ID != result.ID {
		t.Fatal("expected persisted verification result")
	}
	events := auditRepository.ListByExecution("exec-1")
	if len(events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(events))
	}
	if events[0].EventType != domain.AuditVerificationDrift {
		t.Fatalf("expected VERIFICATION_DRIFT, got %q", events[0].EventType)
	}
}
