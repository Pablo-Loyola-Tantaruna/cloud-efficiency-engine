package actions

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type fakeProviderExecutor struct {
	result domain.ExecutionResult
	err    error
	calls  int
}

func (f *fakeProviderExecutor) Execute(_ context.Context, _ domain.Action, execution domain.ExecutionRecord) (domain.ExecutionResult, error) {
	f.calls++
	result := f.result
	if result.ExecutionID == "" {
		result.ExecutionID = execution.ID
	}
	if result.ActionID == "" {
		result.ActionID = execution.ActionID
	}
	if result.Provider == "" || result.Provider == domain.CloudProviderUnknown {
		result.Provider = execution.Provider
	}
	if result.Cluster == "" {
		result.Cluster = execution.Cluster
	}
	return result, f.err
}

type fakeExecutorResolver struct {
	executor domain.ProviderExecutor
	err      error
}

func (r *fakeExecutorResolver) Resolve(_ domain.CloudProvider) (domain.ProviderExecutor, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.executor, nil
}

func engineReadyPlan() domain.ActionPlan {
	return domain.ActionPlan{
		ID:                        "plan-engine-1",
		Provider:                  domain.CloudProviderAWS,
		Cluster:                   "production",
		Status:                    domain.ActionPlanReadyToApply,
		TotalMonthlySavingsUSD:    100,
		TotalAnnualizedSavingsUSD: 1200,
		RequiresApproval:          true,
		Actions: []domain.Action{{
			ID:                   "action-engine-1",
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
		}},
	}
}

func TestExecutionEngine_ShouldExecuteAndVerify(t *testing.T) {
	plan := engineReadyPlan()
	action := plan.Actions[0]
	repo := NewInMemoryExecutionRecordStore()
	executions := NewExecutionService(repo)
	createdAt := time.Date(2026, 8, 17, 1, 0, 0, 0, time.UTC)
	executions.now = func() time.Time { return createdAt }

	executor := &fakeProviderExecutor{result: domain.ExecutionResult{Status: domain.ExecutionResultSucceeded, BeforeValue: 8, DesiredValue: 6, Message: "applied"}}
	resolver := &fakeExecutorResolver{executor: executor}
	reader := &fakeStateReader{state: domain.ObservedState{CurrentValue: 6}}
	verifier := NewVerificationService(reader)
	verifier.now = func() time.Time { return createdAt.Add(time.Minute) }

	engine := NewExecutionEngine(executions, resolver, verifier)
	record, verification, err := engine.Execute(context.Background(), plan, action)
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != domain.ExecutionStatusSucceeded {
		t.Fatalf("expected SUCCEEDED, got %q", record.Status)
	}
	if verification == nil || verification.Status != domain.VerificationStatusVerified {
		t.Fatalf("expected VERIFIED result, got %#v", verification)
	}
	if executor.calls != 1 {
		t.Fatalf("expected one provider execution, got %d", executor.calls)
	}
}

func TestExecutionEngine_ShouldFailWhenProviderExecutionFails(t *testing.T) {
	plan := engineReadyPlan()
	repo := NewInMemoryExecutionRecordStore()
	executions := NewExecutionService(repo)
	executor := &fakeProviderExecutor{err: errors.New("provider timeout")}
	resolver := &fakeExecutorResolver{executor: executor}
	engine := NewExecutionEngine(executions, resolver, nil)

	record, _, err := engine.Execute(context.Background(), plan, plan.Actions[0])
	if err == nil {
		t.Fatal("expected execution error")
	}
	if record.Status != domain.ExecutionStatusFailed {
		t.Fatalf("expected FAILED, got %q", record.Status)
	}
}

func TestExecutionEngine_ShouldRejectMismatchedExecutorResult(t *testing.T) {
	plan := engineReadyPlan()
	repo := NewInMemoryExecutionRecordStore()
	executions := NewExecutionService(repo)
	executor := &fakeProviderExecutor{result: domain.ExecutionResult{
		Status:      domain.ExecutionResultSucceeded,
		ExecutionID: "wrong-execution",
		Message:     "applied",
	}}
	resolver := &fakeExecutorResolver{executor: executor}
	engine := NewExecutionEngine(executions, resolver, nil)

	record, _, err := engine.Execute(context.Background(), plan, plan.Actions[0])
	if err == nil {
		t.Fatal("expected mismatch error")
	}
	if record.Status != domain.ExecutionStatusFailed {
		t.Fatalf("expected FAILED, got %q", record.Status)
	}
}
