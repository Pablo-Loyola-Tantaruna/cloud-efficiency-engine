//go:build live_e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"

	"cloud-efficiency-engine/internal/analysis/actions"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/observability"
	providerregistry "cloud-efficiency-engine/internal/providers"
)

func requireLiveMutation(t *testing.T) {
	t.Helper()
	if os.Getenv("FINOPS_E2E_ALLOW_MUTATION") != "true" {
		t.Skip("live E2E mutation disabled; set FINOPS_E2E_ALLOW_MUTATION=true explicitly")
	}
}

func requiredEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatal("required live E2E environment variable is missing: " + name)
	}
	return value
}

func executeAndVerify(t *testing.T, ctx context.Context, plan domain.ActionPlan, action domain.Action, executor domain.ProviderExecutor, reader domain.StateReader) (domain.ExecutionRecord, *domain.VerificationResult, *observability.RuntimeMetrics) {
	t.Helper()

	store := actions.NewInMemoryExecutionRecordStore()
	executionService := actions.NewExecutionService(store)
	resolver := providerregistry.NewStaticExecutorResolver(map[domain.CloudProvider]domain.ProviderExecutor{
		action.Provider: executor,
	})
	verifier := actions.NewVerificationService(reader)
	metrics := observability.NewRuntimeMetrics(nil)
	engine := actions.NewExecutionEngine(executionService, resolver, verifier, metrics)

	record, verification, err := engine.Execute(ctx, plan, action)
	if err != nil {
		t.Fatalf("live E2E execution failed: %v", err)
	}
	if verification == nil {
		t.Fatal("live E2E execution returned no verification result")
	}
	if verification.Status != domain.VerificationStatusVerified {
		t.Fatalf("live E2E verification failed: status=%s message=%s", verification.Status, verification.Message)
	}
	if verification.ActualValue != action.DesiredValue {
		t.Fatalf("live E2E verification mismatch: expected=%d actual=%d", action.DesiredValue, verification.ActualValue)
	}

	return record, verification, metrics
}

func executeIdempotentAndVerify(t *testing.T, ctx context.Context, plan domain.ActionPlan, action domain.Action, executor domain.ProviderExecutor, reader domain.StateReader) {
	t.Helper()

	store := actions.NewInMemoryExecutionRecordStore()
	executionService := actions.NewExecutionService(store)
	resolver := providerregistry.NewStaticExecutorResolver(map[domain.CloudProvider]domain.ProviderExecutor{
		action.Provider: executor,
	})
	verifier := actions.NewVerificationService(reader)
	metrics := observability.NewRuntimeMetrics(nil)
	engine := actions.NewExecutionEngine(executionService, resolver, verifier, metrics)

	first, firstVerification, err := engine.Execute(ctx, plan, action)
	if err != nil {
		t.Fatalf("first live E2E execution failed: %v", err)
	}
	if firstVerification == nil || firstVerification.Status != domain.VerificationStatusVerified {
		t.Fatalf("first live E2E verification failed: %+v", firstVerification)
	}

	second, secondVerification, err := engine.Execute(ctx, plan, action)
	if err != nil {
		t.Fatalf("idempotent live E2E execution failed: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("idempotent execution created a different execution: first=%s second=%s", first.ID, second.ID)
	}
	if secondVerification == nil || secondVerification.Status != domain.VerificationStatusVerified {
		t.Fatalf("idempotent live E2E verification failed: %+v", secondVerification)
	}
	if second.Status != domain.ExecutionStatusSucceeded {
		t.Fatalf("idempotent execution did not remain succeeded: status=%s", second.Status)
	}
}

func newReduceNodeGroupPlan(provider domain.CloudProvider, cluster, nodeGroup string, current, desired int64) (domain.ActionPlan, domain.Action) {
	monthlySavings := 10.0
	action := domain.Action{
		ID:                   fmt.Sprintf("live-e2e-%s-action", provider),
		Type:                 domain.ActionReduceNodeGroup,
		Provider:             provider,
		Cluster:              cluster,
		NodeGroup:            nodeGroup,
		CurrentValue:         current,
		DesiredValue:         desired,
		MonthlySavingsUSD:    monthlySavings,
		AnnualizedSavingsUSD: monthlySavings * 12,
		Risk:                 domain.ActionRiskLow,
		RequiresApproval:     true,
	}
	plan := domain.ActionPlan{
		ID:                        fmt.Sprintf("live-e2e-%s-plan", provider),
		TenantID:                  "live-e2e",
		Provider:                  provider,
		Cluster:                   cluster,
		Status:                    domain.ActionPlanReadyToApply,
		Actions:                   []domain.Action{action},
		TotalMonthlySavingsUSD:    monthlySavings,
		TotalAnnualizedSavingsUSD: monthlySavings * 12,
		RequiresApproval:          true,
	}
	return plan, action
}

func restoreNodeGroup(ctx context.Context, t *testing.T, provider domain.CloudProvider, cluster, nodeGroup string, reduced, original int64, executor domain.ProviderExecutor, reader domain.StateReader) {
	t.Helper()

	action := domain.Action{
		ID:                   fmt.Sprintf("live-e2e-%s-restore-action", provider),
		Type:                 domain.ActionReduceNodeGroup,
		Provider:             provider,
		Cluster:              cluster,
		NodeGroup:            nodeGroup,
		CurrentValue:         reduced,
		DesiredValue:         original,
		MonthlySavingsUSD:    0.01,
		AnnualizedSavingsUSD: 0.12,
		Risk:                 domain.ActionRiskLow,
		RequiresApproval:     true,
	}
	plan := domain.ActionPlan{
		ID:                        fmt.Sprintf("live-e2e-%s-restore-plan", provider),
		TenantID:                  "live-e2e",
		Provider:                  provider,
		Cluster:                   cluster,
		Status:                    domain.ActionPlanReadyToApply,
		Actions:                   []domain.Action{action},
		TotalMonthlySavingsUSD:    0.01,
		TotalAnnualizedSavingsUSD: 0.12,
		RequiresApproval:          true,
	}

	store := actions.NewInMemoryExecutionRecordStore()
	executionService := actions.NewExecutionService(store)
	record, _, err := executionService.Start(plan, action)
	if err != nil {
		t.Fatalf("create restore execution record: %v", err)
	}
	record, err = executionService.MarkRunning(record)
	if err != nil {
		t.Fatalf("mark restore execution running: %v", err)
	}
	result, err := executor.Execute(ctx, action, record)
	if err != nil {
		t.Fatalf("restore %s node group: %v", provider, err)
	}
	if result.Status != domain.ExecutionResultSucceeded {
		t.Fatalf("restore %s node group returned status %s", provider, result.Status)
	}

	observed, err := reader.ReadState(ctx, action)
	if err != nil {
		t.Fatalf("verify restored %s node group: %v", provider, err)
	}
	if observed.CurrentValue != original {
		t.Fatalf("restore verification failed for %s: expected=%d actual=%d", provider, original, observed.CurrentValue)
	}
}
