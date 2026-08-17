package postgres

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

func TestPostgresRepositories_Integration_ShouldPersistLifecycle(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	pool, err := NewPool(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	if err := ApplyMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE TABLE verification_results, audit_events, recovery_actions, execution_records, action_approvals, action_plans RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}

	repositories, err := NewRepositories(pool)
	if err != nil {
		t.Fatal(err)
	}

	action := domain.Action{
		ID:                   "action-pg-1",
		Type:                 domain.ActionReduceNodeGroup,
		Provider:             domain.CloudProviderAWS,
		Cluster:              "eks-prod",
		NodeGroup:            "workers",
		CurrentValue:         8,
		DesiredValue:         6,
		MonthlySavingsUSD:    100,
		AnnualizedSavingsUSD: 1200,
		Risk:                 domain.ActionRiskMedium,
		RequiresApproval:     true,
	}
	plan := domain.ActionPlan{
		ID:                        "plan-pg-1",
		Provider:                  domain.CloudProviderAWS,
		Cluster:                   "eks-prod",
		Status:                    domain.ActionPlanPreview,
		Actions:                   []domain.Action{action},
		TotalMonthlySavingsUSD:    100,
		TotalAnnualizedSavingsUSD: 1200,
		RequiresApproval:          true,
	}

	created, err := repositories.ActionPlan.CreateActionPlanIfAbsent(plan)
	if err != nil || !created {
		t.Fatalf("expected plan creation, created=%v err=%v", created, err)
	}
	created, err = repositories.ActionPlan.CreateActionPlanIfAbsent(plan)
	if err != nil || created {
		t.Fatalf("expected idempotent plan creation, created=%v err=%v", created, err)
	}

	storedPlan, found, err := repositories.ActionPlan.GetActionPlanByID(plan.ID)
	if err != nil || !found || storedPlan.ID != plan.ID || len(storedPlan.Actions) != 1 {
		t.Fatalf("unexpected stored plan: found=%v err=%v plan=%+v", found, err, storedPlan)
	}

	approval := domain.ActionApproval{
		PlanID:     plan.ID,
		ApprovedBy: "integration-test",
		ApprovedAt: time.Now().UTC(),
		Comment:    "approved for sandbox",
	}
	if err := repositories.Approval.SaveApproval(approval); err != nil {
		t.Fatal(err)
	}
	storedApproval, found, err := repositories.Approval.GetApprovalByPlanID(plan.ID)
	if err != nil || !found || storedApproval.ApprovedBy != approval.ApprovedBy {
		t.Fatalf("unexpected approval: found=%v err=%v approval=%+v", found, err, storedApproval)
	}

	now := time.Now().UTC()
	execution := domain.ExecutionRecord{
		ID:             "exec-pg-1",
		IdempotencyKey: "plan-pg-1:action-pg-1",
		PlanID:         plan.ID,
		ActionID:       action.ID,
		Provider:       action.Provider,
		Cluster:        action.Cluster,
		Status:         domain.ExecutionStatusPending,
		Attempt:        1,
		CurrentValue:   8,
		DesiredValue:   6,
		StartedAt:      now,
	}
	created, err = repositories.Execution.CreateIfAbsent(execution)
	if err != nil || !created {
		t.Fatalf("expected execution creation, created=%v err=%v", created, err)
	}
	created, err = repositories.Execution.CreateIfAbsent(execution)
	if err != nil || created {
		t.Fatalf("expected execution idempotency, created=%v err=%v", created, err)
	}

	running := execution
	running.Status = domain.ExecutionStatusRunning
	if err := repositories.Execution.Update(running); err != nil {
		t.Fatal(err)
	}
	completedAt := time.Now().UTC()
	succeeded := running
	succeeded.Status = domain.ExecutionStatusSucceeded
	succeeded.CompletedAt = &completedAt
	succeeded.Result = "updated desired size"
	if err := repositories.Execution.Update(succeeded); err != nil {
		t.Fatal(err)
	}
	storedExecution, found := repositories.Execution.GetByIdempotencyKey(execution.IdempotencyKey)
	if !found || storedExecution.Status != domain.ExecutionStatusSucceeded {
		t.Fatalf("unexpected execution state: found=%v record=%+v", found, storedExecution)
	}
	attempts := repositories.Execution.ListByIdempotencyKey(execution.IdempotencyKey)
	if len(attempts) != 1 || attempts[0].Attempt != 1 {
		t.Fatalf("unexpected attempts: %+v", attempts)
	}

	event := domain.AuditEvent{
		ID:          "audit-pg-1",
		PlanID:      plan.ID,
		ActionID:    action.ID,
		ExecutionID: execution.ID,
		Attempt:     1,
		EventType:   domain.AuditExecutionSucceeded,
		Actor:       "integration-test",
		Timestamp:   time.Now().UTC(),
		Provider:    action.Provider,
		Cluster:     action.Cluster,
		NewState:    string(domain.ExecutionStatusSucceeded),
		Message:     "execution succeeded",
		Metadata:    map[string]string{"source": "postgres-test"},
	}
	if err := repositories.Audit.Append(event); err != nil {
		t.Fatal(err)
	}
	if events := repositories.Audit.ListByExecution(execution.ID); len(events) != 1 || events[0].ID != event.ID {
		t.Fatalf("unexpected audit events: %+v", events)
	}

	verification := domain.VerificationResult{
		ID:            "verification-pg-1",
		PlanID:        plan.ID,
		ActionID:      action.ID,
		ExecutionID:   execution.ID,
		Attempt:       1,
		Provider:      action.Provider,
		Cluster:       action.Cluster,
		Status:        domain.VerificationStatusVerified,
		ExpectedValue: 6,
		ActualValue:   6,
		VerifiedAt:    time.Now().UTC(),
		Message:       "desired size verified",
	}
	if err := repositories.Verification.Save(verification); err != nil {
		t.Fatal(err)
	}
	storedVerification, found := repositories.Verification.GetByExecutionID(execution.ID)
	if !found || storedVerification.Status != domain.VerificationStatusVerified {
		t.Fatalf("unexpected verification: found=%v result=%+v", found, storedVerification)
	}

	recovery := domain.RecoveryAction{
		ID:               "recovery-pg-1",
		PlanID:           plan.ID,
		ActionID:         action.ID,
		ExecutionID:      execution.ID,
		Provider:         action.Provider,
		Cluster:          action.Cluster,
		Resource:         action.NodeGroup,
		FromValue:        6,
		ToValue:          8,
		Reason:           "compensate drift",
		Status:           domain.RecoveryReady,
		RequiresApproval: true,
		CreatedAt:        time.Now().UTC(),
	}
	if err := repositories.Recovery.SaveRecovery(recovery); err != nil {
		t.Fatal(err)
	}
	storedRecovery, found, err := repositories.Recovery.GetRecoveryByID(recovery.ID)
	if err != nil || !found || storedRecovery.Status != domain.RecoveryReady {
		t.Fatalf("unexpected recovery: found=%v err=%v action=%+v", found, err, storedRecovery)
	}
}

func TestPostgresExecutionRepository_Integration_ShouldCreateAttemptOnlyOnce(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	pool, err := NewPool(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := ApplyMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE TABLE verification_results, audit_events, recovery_actions, execution_records, action_approvals, action_plans RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}

	repositories, err := NewRepositories(pool)
	if err != nil {
		t.Fatal(err)
	}
	action := domain.Action{
		ID:                   "action-concurrent",
		Type:                 domain.ActionReduceNodeGroup,
		Provider:             domain.CloudProviderAWS,
		Cluster:              "eks-concurrent",
		NodeGroup:            "workers",
		CurrentValue:         8,
		DesiredValue:         6,
		MonthlySavingsUSD:    100,
		AnnualizedSavingsUSD: 1200,
		Risk:                 domain.ActionRiskMedium,
		RequiresApproval:     true,
	}
	plan := domain.ActionPlan{
		ID:                        "plan-concurrent",
		Provider:                  domain.CloudProviderAWS,
		Cluster:                   "eks-concurrent",
		Status:                    domain.ActionPlanPreview,
		Actions:                   []domain.Action{action},
		TotalMonthlySavingsUSD:    100,
		TotalAnnualizedSavingsUSD: 1200,
		RequiresApproval:          true,
	}
	if _, err := repositories.ActionPlan.CreateActionPlanIfAbsent(plan); err != nil {
		t.Fatal(err)
	}

	record := domain.ExecutionRecord{
		ID:             "exec-concurrent-pg",
		IdempotencyKey: "plan-concurrent:action-concurrent",
		PlanID:         plan.ID,
		ActionID:       action.ID,
		Provider:       domain.CloudProviderAWS,
		Cluster:        "eks-concurrent",
		Status:         domain.ExecutionStatusPending,
		Attempt:        1,
		CurrentValue:   8,
		DesiredValue:   6,
		StartedAt:      time.Now().UTC(),
	}

	var createdCount int32
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			created, err := repositories.Execution.CreateIfAbsent(record)
			if err != nil {
				t.Errorf("create execution: %v", err)
				return
			}
			if created {
				atomic.AddInt32(&createdCount, 1)
			}
		}()
	}
	wg.Wait()
	if createdCount != 1 {
		t.Fatalf("expected exactly one creator, got %d", createdCount)
	}
	attempts := repositories.Execution.ListByIdempotencyKey(record.IdempotencyKey)
	if len(attempts) != 1 {
		t.Fatalf("expected exactly one persisted attempt, got %d", len(attempts))
	}
}
