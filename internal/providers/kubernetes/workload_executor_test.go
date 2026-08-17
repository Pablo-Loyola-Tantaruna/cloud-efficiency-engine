package kubernetes

import (
	"context"
	"strings"
	"testing"

	"cloud-efficiency-engine/internal/domain"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeClientFake "k8s.io/client-go/kubernetes/fake"
)

func TestWorkloadExecutor_ShouldResizeDeploymentCPU(t *testing.T) {
	client := kubeClientFake.NewSimpleClientset(sampleDeployment("app", "api", "500m", "512Mi"))
	executor := NewWorkloadExecutor(client)
	action := sampleAction(domain.ActionRightsizeWorkloadCPU, 500, 250)

	result, err := executor.Execute(context.Background(), action, sampleExecution(action))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Status != domain.ExecutionResultSucceeded {
		t.Fatalf("expected success result, got %s", result.Status)
	}

	deployment, err := client.AppsV1().Deployments("app").Get(context.Background(), "api", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	if got := deployment.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().MilliValue(); got != 250 {
		t.Fatalf("expected cpu request 250m, got %d", got)
	}
}

func TestWorkloadExecutor_ShouldResizeStatefulSetMemory(t *testing.T) {
	client := kubeClientFake.NewSimpleClientset(sampleStatefulSet("app", "db", "250m", "2Gi"))
	executor := NewWorkloadExecutor(client)
	action := sampleActionForWorkloadRef("app/db", domain.WorkloadStatefulSet, domain.ActionRightsizeWorkloadMemory, 2*1024*1024*1024, 1024*1024*1024)

	result, err := executor.Execute(context.Background(), action, sampleExecution(action))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Status != domain.ExecutionResultSucceeded {
		t.Fatalf("expected success result, got %s", result.Status)
	}

	statefulSet, err := client.AppsV1().StatefulSets("app").Get(context.Background(), "db", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get statefulset: %v", err)
	}
	if got := statefulSet.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().Value(); got != 1024*1024*1024 {
		t.Fatalf("expected memory request 1Gi, got %d", got)
	}
}

func TestWorkloadExecutor_ShouldRejectPreExecutionDrift(t *testing.T) {
	client := kubeClientFake.NewSimpleClientset(sampleDeployment("app", "api", "750m", "512Mi"))
	executor := NewWorkloadExecutor(client)
	action := sampleAction(domain.ActionRightsizeWorkloadCPU, 500, 250)

	_, err := executor.Execute(context.Background(), action, sampleExecution(action))
	if err == nil || !strings.Contains(err.Error(), "drift detected") {
		t.Fatalf("expected pre-execution drift error, got %v", err)
	}
}

func TestWorkloadExecutor_ShouldTreatAlreadyDesiredValueAsIdempotent(t *testing.T) {
	client := kubeClientFake.NewSimpleClientset(sampleDeployment("app", "api", "250m", "512Mi"))
	executor := NewWorkloadExecutor(client)
	action := sampleAction(domain.ActionRightsizeWorkloadCPU, 500, 250)

	result, err := executor.Execute(context.Background(), action, sampleExecution(action))
	if err != nil {
		t.Fatalf("expected idempotent success, got %v", err)
	}
	if !strings.Contains(result.Message, "already at desired value") {
		t.Fatalf("expected idempotent message, got %q", result.Message)
	}
}

func TestWorkloadExecutor_ShouldRequireExplicitContainerForMultiContainerWorkload(t *testing.T) {
	deployment := sampleDeployment("app", "api", "500m", "512Mi")
	deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, coreV1.Container{
		Name: "sidecar",
		Resources: coreV1.ResourceRequirements{Requests: coreV1.ResourceList{
			coreV1.ResourceCPU: resource.MustParse("100m"),
		}},
	})
	client := kubeClientFake.NewSimpleClientset(deployment)
	executor := NewWorkloadExecutor(client)
	action := sampleAction(domain.ActionRightsizeWorkloadCPU, 500, 250)

	_, err := executor.Execute(context.Background(), action, sampleExecution(action))
	if err == nil || !strings.Contains(err.Error(), "explicit container target") {
		t.Fatalf("expected explicit container error, got %v", err)
	}
}

func TestWorkloadExecutor_ShouldRejectJobs(t *testing.T) {
	client := kubeClientFake.NewSimpleClientset()
	executor := NewWorkloadExecutor(client)
	action := sampleActionForWorkload(domain.WorkloadJob, domain.ActionRightsizeWorkloadCPU, 500, 250)

	_, err := executor.Execute(context.Background(), action, sampleExecution(action))
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("expected unsupported job error, got %v", err)
	}
}

func sampleAction(actionType domain.ActionType, current, desired int64) domain.Action {
	return sampleActionForWorkloadRef("app/api", domain.WorkloadDeployment, actionType, current, desired)
}

func sampleActionForWorkload(workloadType domain.WorkloadType, actionType domain.ActionType, current, desired int64) domain.Action {
	return sampleActionForWorkloadRef("app/api", workloadType, actionType, current, desired)
}

func sampleActionForWorkloadRef(workloadRef string, workloadType domain.WorkloadType, actionType domain.ActionType, current, desired int64) domain.Action {
	return domain.Action{
		ID:                   "action-1",
		Type:                 actionType,
		Provider:             domain.CloudProviderAWS,
		Cluster:              "cluster-1",
		Workload:             workloadRef,
		WorkloadType:         workloadType,
		CurrentValue:         current,
		DesiredValue:         desired,
		MonthlySavingsUSD:    10,
		AnnualizedSavingsUSD: 120,
		Risk:                 domain.ActionRiskLow,
		RequiresApproval:     true,
	}
}

func sampleExecution(action domain.Action) domain.ExecutionRecord {
	return domain.ExecutionRecord{
		ID:             "exec-1",
		ActionID:       action.ID,
		Provider:       action.Provider,
		Cluster:        action.Cluster,
		PlanID:         "plan-1",
		IdempotencyKey: "plan-1:" + action.ID,
		Status:         domain.ExecutionStatusRunning,
		Attempt:        1,
		CurrentValue:   action.CurrentValue,
		DesiredValue:   action.DesiredValue,
	}
}

func sampleDeployment(namespace, name, cpu, memory string) *appsV1.Deployment {
	return &appsV1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Spec: appsV1.DeploymentSpec{Template: coreV1.PodTemplateSpec{
			Spec: coreV1.PodSpec{Containers: []coreV1.Container{{
				Name: "api",
				Resources: coreV1.ResourceRequirements{Requests: coreV1.ResourceList{
					coreV1.ResourceCPU:    resource.MustParse(cpu),
					coreV1.ResourceMemory: resource.MustParse(memory),
				}},
			}}},
		}},
	}
}

func sampleStatefulSet(namespace, name, cpu, memory string) *appsV1.StatefulSet {
	return &appsV1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Spec: appsV1.StatefulSetSpec{Template: coreV1.PodTemplateSpec{
			Spec: coreV1.PodSpec{Containers: []coreV1.Container{{
				Name: "api",
				Resources: coreV1.ResourceRequirements{Requests: coreV1.ResourceList{
					coreV1.ResourceCPU:    resource.MustParse(cpu),
					coreV1.ResourceMemory: resource.MustParse(memory),
				}},
			}}},
		}},
	}
}
