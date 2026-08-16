package actions

import (
	"errors"
	"sync"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

func executionReadyPlan() domain.ActionPlan {
	return domain.ActionPlan{
		ID: "plan-1", Provider: domain.CloudProviderAWS, Cluster: "production", Status: domain.ActionPlanReadyToApply,
		TotalMonthlySavingsUSD: 100, TotalAnnualizedSavingsUSD: 1200, RequiresApproval: true,
		Actions: []domain.Action{{ID: "action-1", Type: domain.ActionReduceNodeGroup, Provider: domain.CloudProviderAWS, Cluster: "production", NodeGroup: "workers", CurrentValue: 8, DesiredValue: 6, MonthlySavingsUSD: 100, AnnualizedSavingsUSD: 1200, Risk: domain.ActionRiskMedium, RequiresApproval: true}},
	}
}

func TestStartExecutionRecord_ShouldBeIdempotent(t *testing.T) {
	store := NewInMemoryExecutionRecordStore()
	plan := executionReadyPlan()
	now := time.Date(2026, 8, 16, 19, 30, 0, 0, time.UTC)
	first, created, err := StartExecutionRecord(store, plan, "action-1", now)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected first execution record to be created")
	}
	second, created, err := StartExecutionRecord(store, plan, "action-1", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("expected second request to reuse existing execution record")
	}
	if second.ID != first.ID || second.IdempotencyKey != first.IdempotencyKey || second.Attempt != 1 {
		t.Fatal("expected identical execution identity")
	}
	if !second.StartedAt.Equal(first.StartedAt) {
		t.Fatal("expected original execution timestamp to be preserved")
	}
}

func TestExecutionRecordLifecycle_ShouldTrackSuccess(t *testing.T) {
	store := NewInMemoryExecutionRecordStore()
	record, _, err := StartExecutionRecord(store, executionReadyPlan(), "action-1", time.Unix(1, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	record, err = MarkExecutionRunning(store, record)
	if err != nil {
		t.Fatal(err)
	}
	completedAt := record.StartedAt.Add(time.Minute)
	record, err = FinishExecution(store, record, domain.ExecutionStatusSucceeded, nil, completedAt)
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != domain.ExecutionStatusSucceeded || record.CompletedAt == nil || !record.CompletedAt.Equal(completedAt) {
		t.Fatalf("unexpected success record: %+v", record)
	}
}

func TestExecutionRecordLifecycle_ShouldTrackFailureAndRetry(t *testing.T) {
	store := NewInMemoryExecutionRecordStore()
	service := NewExecutionService(store)
	record, created, err := service.Start(executionReadyPlan(), executionReadyPlan().Actions[0])
	if err != nil || !created {
		t.Fatalf("start failed: %v", err)
	}
	record, err = service.MarkRunning(record)
	if err != nil {
		t.Fatal(err)
	}
	failed, err := service.Fail(record, errors.New("provider timeout"))
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != domain.ExecutionStatusFailed {
		t.Fatalf("expected FAILED, got %q", failed.Status)
	}
	retry, created, err := service.Retry(executionReadyPlan(), executionReadyPlan().Actions[0])
	if err != nil || !created {
		t.Fatalf("retry failed: %v", err)
	}
	if retry.Attempt != 2 || retry.Status != domain.ExecutionStatusPending {
		t.Fatalf("unexpected retry: %+v", retry)
	}
}

func TestExecutionService_ShouldNeverRetrySucceededExecution(t *testing.T) {
	store := NewInMemoryExecutionRecordStore()
	service := NewExecutionService(store)
	plan := executionReadyPlan()
	record, _, err := service.Start(plan, plan.Actions[0])
	if err != nil {
		t.Fatal(err)
	}
	record, err = service.MarkRunning(record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Complete(record, "scaled from 8 to 6"); err != nil {
		t.Fatal(err)
	}
	if service.IsAlreadyExecuted(plan.ID, plan.Actions[0].ID) == false {
		t.Fatal("expected execution to be terminally successful")
	}
	if _, _, err = service.Retry(plan, plan.Actions[0]); err == nil {
		t.Fatal("expected retry rejection")
	}
}

func TestExecutionStore_ShouldCreateOnlyOneConcurrentAttempt(t *testing.T) {
	store := NewInMemoryExecutionRecordStore()
	plan := executionReadyPlan()
	const workers = 32
	results := make(chan bool, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			_, created, err := StartExecutionRecord(store, plan, "action-1", time.Now().UTC())
			results <- created
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	createdCount := 0
	for created := range results {
		if created {
			createdCount++
		}
	}
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if createdCount != 1 {
		t.Fatalf("expected exactly one creator, got %d", createdCount)
	}
}

func TestStartExecutionRecord_ShouldRejectUnreadyPlan(t *testing.T) {
	plan := executionReadyPlan()
	plan.Status = domain.ActionPlanApproved
	store := NewInMemoryExecutionRecordStore()
	if _, _, err := StartExecutionRecord(store, plan, "action-1", time.Now().UTC()); err == nil {
		t.Fatal("expected unready plan error")
	}
}

func TestFinishExecution_ShouldRejectNonTerminalStatus(t *testing.T) {
	store := NewInMemoryExecutionRecordStore()
	record, _, err := StartExecutionRecord(store, executionReadyPlan(), "action-1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = MarkExecutionRunning(store, record); err != nil {
		t.Fatal(err)
	}
	if _, err = FinishExecution(store, record, domain.ExecutionStatusRunning, nil, time.Now().UTC()); err == nil {
		t.Fatal("expected terminal status validation error")
	}
}
