package actions

import (
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

var recoveryTestTime = time.Date(2026, 8, 17, 2, 0, 0, 0, time.UTC)

func recoveryTestAction() domain.Action {
	return domain.Action{
		ID: "action-recovery-1", Type: domain.ActionReduceNodeGroup,
		Provider: domain.CloudProviderAWS, Cluster: "production", NodeGroup: "workers",
		CurrentValue: 8, DesiredValue: 6, MonthlySavingsUSD: 100, AnnualizedSavingsUSD: 1200,
		Risk: domain.ActionRiskMedium, RequiresApproval: true,
	}
}

func recoveryTestExecution() domain.ExecutionRecord {
	completed := recoveryTestTime
	return domain.ExecutionRecord{
		ID: "exec-recovery-1", IdempotencyKey: "plan-recovery-1:action-recovery-1",
		PlanID: "plan-recovery-1", ActionID: "action-recovery-1", Provider: domain.CloudProviderAWS,
		Cluster: "production", Status: domain.ExecutionStatusSucceeded, Attempt: 1,
		CurrentValue: 8, DesiredValue: 6, StartedAt: recoveryTestTime.Add(-time.Minute),
		CompletedAt: &completed,
	}
}

func TestBuildRecoveryAction_ShouldCreateCompensatingActionFromDrift(t *testing.T) {
	action := recoveryTestAction()
	execution := recoveryTestExecution()
	verification := domain.VerificationResult{
		ID: "verify-recovery-1", PlanID: execution.PlanID, ActionID: action.ID, ExecutionID: execution.ID,
		Attempt: 1, Provider: execution.Provider, Cluster: execution.Cluster,
		Status: domain.VerificationStatusDrift, ExpectedValue: 6, ActualValue: 7,
		VerifiedAt: recoveryTestTime, Message: "drift",
	}

	recovery, err := BuildRecoveryAction(execution, action, verification, recoveryTestTime.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if recovery.FromValue != 7 || recovery.ToValue != 8 {
		t.Fatalf("unexpected recovery values: %+v", recovery)
	}
	if recovery.Status != domain.RecoveryReady || !recovery.RequiresApproval {
		t.Fatalf("unexpected recovery lifecycle: %+v", recovery)
	}
}

func TestBuildRecoveryAction_ShouldRejectVerifiedState(t *testing.T) {
	action := recoveryTestAction()
	execution := recoveryTestExecution()
	verification := domain.VerificationResult{
		ID: "verify-recovery-1", PlanID: execution.PlanID, ActionID: action.ID, ExecutionID: execution.ID,
		Attempt: 1, Provider: execution.Provider, Cluster: execution.Cluster,
		Status: domain.VerificationStatusVerified, ExpectedValue: 6, ActualValue: 6,
		VerifiedAt: recoveryTestTime,
	}
	if _, err := BuildRecoveryAction(execution, action, verification, recoveryTestTime); err == nil {
		t.Fatal("expected verified recovery rejection")
	}
}
