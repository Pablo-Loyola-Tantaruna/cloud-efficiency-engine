package actions

import (
	"context"
	"testing"

	"cloud-efficiency-engine/internal/domain"
	kubeprovider "cloud-efficiency-engine/internal/providers/kubernetes"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeClientFake "k8s.io/client-go/kubernetes/fake"
)

func TestExecutionEngine_ShouldExecuteAndVerifyWorkloadRightsizing(t *testing.T) {
	client := kubeClientFake.NewSimpleClientset(&appsV1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: "app", Name: "api"},
		Spec: appsV1.DeploymentSpec{Template: coreV1.PodTemplateSpec{
			Spec: coreV1.PodSpec{Containers: []coreV1.Container{{
				Name: "api",
				Resources: coreV1.ResourceRequirements{Requests: coreV1.ResourceList{
					coreV1.ResourceCPU: resource.MustParse("500m"),
				}},
			}}},
		}},
	})

	executor := kubeprovider.NewWorkloadExecutor(client)
	store := NewInMemoryExecutionRecordStore()
	executionService := NewExecutionService(store)
	resolver := &stubExecutorResolver{executor: executor}
	verifier := NewVerificationService(executor)
	engine := NewExecutionEngine(executionService, resolver, verifier)

	action := domain.Action{
		ID:                   "action-cpu",
		Type:                 domain.ActionRightsizeWorkloadCPU,
		Provider:             domain.CloudProviderAWS,
		Cluster:              "cluster-1",
		Workload:             "app/api",
		WorkloadType:         domain.WorkloadDeployment,
		CurrentValue:         500,
		DesiredValue:         250,
		MonthlySavingsUSD:    10,
		AnnualizedSavingsUSD: 120,
		Risk:                 domain.ActionRiskLow,
		RequiresApproval:     true,
	}
	plan := domain.ActionPlan{
		ID:                        "plan-cpu",
		TenantID:                  "tenant-1",
		Provider:                  domain.CloudProviderAWS,
		Cluster:                   "cluster-1",
		Status:                    domain.ActionPlanReadyToApply,
		Actions:                   []domain.Action{action},
		TotalMonthlySavingsUSD:    10,
		TotalAnnualizedSavingsUSD: 120,
		RequiresApproval:          true,
	}

	record, verification, err := engine.Execute(context.Background(), plan, action)
	if err != nil {
		t.Fatalf("expected execution to succeed, got %v", err)
	}
	if record.Status != domain.ExecutionStatusSucceeded {
		t.Fatalf("expected succeeded execution, got %s", record.Status)
	}
	if verification == nil || verification.Status != domain.VerificationStatusVerified {
		t.Fatalf("expected verified execution, got %#v", verification)
	}

	deployment, err := client.AppsV1().Deployments("app").Get(context.Background(), "api", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get deployment after execution: %v", err)
	}
	if got := deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().MilliValue(); got != 250 {
		t.Fatalf("expected cpu request 250m after execution, got %d", got)
	}
}

type stubExecutorResolver struct {
	executor domain.ProviderExecutor
}

func (r *stubExecutorResolver) Resolve(provider domain.CloudProvider) (domain.ProviderExecutor, error) {
	return r.executor, nil
}
