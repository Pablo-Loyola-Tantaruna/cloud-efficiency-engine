package aws

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type fakeEKSNodeGroupClient struct {
	describeValue int64
	describeErr   error
	updateID      string
	updateErr     error
	waitErr       error
	calls         int
	cluster       string
	nodeGroup     string
	desired       int64
	token         string
	waitedID      string
}

func (f *fakeEKSNodeGroupClient) DescribeDesiredSize(_ context.Context, cluster, nodeGroup string) (int64, error) {
	f.cluster = cluster
	f.nodeGroup = nodeGroup
	return f.describeValue, f.describeErr
}

func (f *fakeEKSNodeGroupClient) UpdateDesiredSize(_ context.Context, cluster, nodeGroup string, desired int64, token string) (string, error) {
	f.calls++
	f.cluster = cluster
	f.nodeGroup = nodeGroup
	f.desired = desired
	f.token = token
	if f.updateErr != nil {
		return "", f.updateErr
	}
	return f.updateID, nil
}

func (f *fakeEKSNodeGroupClient) WaitForUpdate(_ context.Context, cluster, nodeGroup, updateID string) error {
	f.cluster = cluster
	f.nodeGroup = nodeGroup
	f.waitedID = updateID
	return f.waitErr
}

var fixedTime = time.Date(2026, 8, 16, 20, 0, 0, 0, time.UTC)

func validExecution() domain.ExecutionRecord {
	return domain.ExecutionRecord{
		ID:             "exec-1",
		IdempotencyKey: "plan-1:action-1",
		PlanID:         "plan-1",
		ActionID:       "action-1",
		Provider:       domain.CloudProviderAWS,
		Cluster:        "production",
		Status:         domain.ExecutionStatusRunning,
		Attempt:        1,
		CurrentValue:   8,
		DesiredValue:   6,
		StartedAt:      fixedTime,
	}
}

func validAction() domain.Action {
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

func TestEKSExecutor_ShouldUpdateDesiredSizeAndWait(t *testing.T) {
	client := &fakeEKSNodeGroupClient{describeValue: 8, updateID: "update-1"}
	executor := NewEKSExecutor(client)

	result, err := executor.Execute(context.Background(), validAction(), validExecution())
	if err != nil {
		t.Fatal(err)
	}
	if client.calls != 1 || client.cluster != "production" || client.nodeGroup != "workers" || client.desired != 6 {
		t.Fatalf("unexpected update call: %+v", client)
	}
	if client.token != "plan-1:action-1" || client.waitedID != "update-1" {
		t.Fatalf("expected idempotency token and wait id, got token=%q update=%q", client.token, client.waitedID)
	}
	if result.Status != domain.ExecutionResultSucceeded {
		t.Fatalf("expected success, got %q", result.Status)
	}
	if result.ExecutionID != "exec-1" || result.BeforeValue != 8 || result.DesiredValue != 6 {
		t.Fatalf("unexpected execution result: %+v", result)
	}
}

func TestEKSExecutor_ShouldRejectPreExecutionDrift(t *testing.T) {
	client := &fakeEKSNodeGroupClient{describeValue: 7, updateID: "update-1"}
	executor := NewEKSExecutor(client)

	if _, err := executor.Execute(context.Background(), validAction(), validExecution()); err == nil {
		t.Fatal("expected pre-execution drift error")
	}
	if client.calls != 0 {
		t.Fatal("must not update a drifted node group")
	}
}

func TestEKSExecutor_ShouldPropagateDescribeFailure(t *testing.T) {
	providerErr := errors.New("access denied")
	executor := NewEKSExecutor(&fakeEKSNodeGroupClient{describeErr: providerErr})

	if _, err := executor.Execute(context.Background(), validAction(), validExecution()); !errors.Is(err, providerErr) {
		t.Fatalf("expected describe error, got %v", err)
	}
}

func TestEKSExecutor_ShouldPropagateUpdateFailure(t *testing.T) {
	providerErr := errors.New("resource in use")
	executor := NewEKSExecutor(&fakeEKSNodeGroupClient{describeValue: 8, updateErr: providerErr})

	if _, err := executor.Execute(context.Background(), validAction(), validExecution()); !errors.Is(err, providerErr) {
		t.Fatalf("expected update error, got %v", err)
	}
}

func TestEKSExecutor_ShouldPropagateUpdateWaitFailure(t *testing.T) {
	providerErr := errors.New("update failed")
	client := &fakeEKSNodeGroupClient{describeValue: 8, updateID: "update-1", waitErr: providerErr}
	executor := NewEKSExecutor(client)

	if _, err := executor.Execute(context.Background(), validAction(), validExecution()); !errors.Is(err, providerErr) {
		t.Fatalf("expected wait error, got %v", err)
	}
}

func TestEKSExecutor_ShouldRejectEmptyUpdateID(t *testing.T) {
	client := &fakeEKSNodeGroupClient{describeValue: 8}
	executor := NewEKSExecutor(client)

	if _, err := executor.Execute(context.Background(), validAction(), validExecution()); err == nil {
		t.Fatal("expected empty update id error")
	}
}

func TestEKSExecutor_ShouldRejectNonRunningExecution(t *testing.T) {
	executor := NewEKSExecutor(&fakeEKSNodeGroupClient{describeValue: 8, updateID: "update-1"})
	execution := validExecution()
	execution.Status = domain.ExecutionStatusSucceeded

	if _, err := executor.Execute(context.Background(), validAction(), execution); err == nil {
		t.Fatal("expected lifecycle error")
	}
}

func TestEKSExecutor_ShouldRejectUnsupportedAction(t *testing.T) {
	executor := NewEKSExecutor(&fakeEKSNodeGroupClient{describeValue: 8, updateID: "update-1"})
	action := validAction()
	action.Type = domain.ActionRightsizeWorkloadCPU

	if _, err := executor.Execute(context.Background(), action, validExecution()); err == nil {
		t.Fatal("expected unsupported action error")
	}
}

func TestEKSExecutor_ShouldRejectMismatchedValues(t *testing.T) {
	executor := NewEKSExecutor(&fakeEKSNodeGroupClient{describeValue: 8, updateID: "update-1"})
	action := validAction()
	action.DesiredValue = 5

	if _, err := executor.Execute(context.Background(), action, validExecution()); err == nil {
		t.Fatal("expected value mismatch error")
	}
}
