package kubernetes

import (
	"context"
	"fmt"
	"strings"

	"cloud-efficiency-engine/internal/domain"
	coreV1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "k8s.io/client-go/kubernetes"
)

type WorkloadExecutor struct {
	client kubeclient.Interface
}

func NewWorkloadExecutor(client kubeclient.Interface) *WorkloadExecutor {
	return &WorkloadExecutor{client: client}
}

func (e *WorkloadExecutor) Execute(ctx context.Context, action domain.Action, execution domain.ExecutionRecord) (domain.ExecutionResult, error) {
	if e == nil || e.client == nil {
		return domain.ExecutionResult{}, fmt.Errorf("Kubernetes workload client must not be nil")
	}
	if action.Type != domain.ActionRightsizeWorkloadCPU && action.Type != domain.ActionRightsizeWorkloadMemory {
		return domain.ExecutionResult{}, fmt.Errorf("Kubernetes workload executor does not support action type %q", action.Type)
	}
	if action.Cluster != execution.Cluster || action.Provider != execution.Provider || action.ID != execution.ActionID {
		return domain.ExecutionResult{}, fmt.Errorf("workload action %q does not match execution %q", action.ID, execution.ID)
	}
	if action.WorkloadType == domain.WorkloadUnknown || action.WorkloadType == domain.WorkloadJob {
		return domain.ExecutionResult{}, fmt.Errorf("workload type %q is not supported for mutation", action.WorkloadType)
	}
	namespace, name, err := parseWorkloadReference(action.Workload)
	if err != nil {
		return domain.ExecutionResult{}, err
	}

	current, err := e.readValue(ctx, action)
	if err != nil {
		return domain.ExecutionResult{}, err
	}
	if current == action.DesiredValue {
		return successResult(execution, fmt.Sprintf("workload %s is already at desired value %d", action.Workload, action.DesiredValue)), nil
	}
	if current != action.CurrentValue {
		return domain.ExecutionResult{}, fmt.Errorf("workload %s drift detected before execution: expected current value %d, observed %d", action.Workload, action.CurrentValue, current)
	}

	var updated bool
	switch action.WorkloadType {
	case domain.WorkloadDeployment:
		obj, getErr := e.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			return domain.ExecutionResult{}, getErr
		}
		updated, err = updatePodTemplate(&obj.Spec.Template, action)
		if err == nil && updated {
			_, err = e.client.AppsV1().Deployments(namespace).Update(ctx, obj, metav1.UpdateOptions{})
		}
	case domain.WorkloadStatefulSet:
		obj, getErr := e.client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			return domain.ExecutionResult{}, getErr
		}
		updated, err = updatePodTemplate(&obj.Spec.Template, action)
		if err == nil && updated {
			_, err = e.client.AppsV1().StatefulSets(namespace).Update(ctx, obj, metav1.UpdateOptions{})
		}
	case domain.WorkloadDaemonSet:
		obj, getErr := e.client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			return domain.ExecutionResult{}, getErr
		}
		updated, err = updatePodTemplate(&obj.Spec.Template, action)
		if err == nil && updated {
			_, err = e.client.AppsV1().DaemonSets(namespace).Update(ctx, obj, metav1.UpdateOptions{})
		}
	default:
		return domain.ExecutionResult{}, fmt.Errorf("unsupported workload type %q", action.WorkloadType)
	}
	if err != nil {
		return domain.ExecutionResult{}, err
	}
	if !updated {
		return domain.ExecutionResult{}, fmt.Errorf("no container resource change was applied to workload %s", action.Workload)
	}

	observed, err := e.readValue(ctx, action)
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("read workload %s after update: %w", action.Workload, err)
	}
	if observed != action.DesiredValue {
		return domain.ExecutionResult{}, fmt.Errorf("workload %s did not reach desired value %d, observed %d", action.Workload, action.DesiredValue, observed)
	}
	return successResult(execution, fmt.Sprintf("workload %s %s request changed from %d to %d", action.Workload, action.Type, action.CurrentValue, action.DesiredValue)), nil
}

func (e *WorkloadExecutor) ReadState(ctx context.Context, action domain.Action) (domain.ObservedState, error) {
	value, err := e.readValue(ctx, action)
	if err != nil {
		return domain.ObservedState{}, err
	}
	return domain.ObservedState{CurrentValue: value}, nil
}

func (e *WorkloadExecutor) readValue(ctx context.Context, action domain.Action) (int64, error) {
	if action.WorkloadType == domain.WorkloadJob {
		return 0, fmt.Errorf("workload type Job is not supported for rightsizing")
	}
	namespace, name, err := parseWorkloadReference(action.Workload)
	if err != nil {
		return 0, err
	}
	var template coreV1.PodTemplateSpec
	switch action.WorkloadType {
	case domain.WorkloadDeployment:
		obj, getErr := e.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			return 0, getErr
		}
		template = obj.Spec.Template
	case domain.WorkloadStatefulSet:
		obj, getErr := e.client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			return 0, getErr
		}
		template = obj.Spec.Template
	case domain.WorkloadDaemonSet:
		obj, getErr := e.client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			return 0, getErr
		}
		template = obj.Spec.Template
	default:
		return 0, fmt.Errorf("unsupported workload type %q", action.WorkloadType)
	}
	container, err := findTargetContainer(template.Spec.Containers, action.Container)
	if err != nil {
		return 0, err
	}
	return requestedValue(container.Resources.Requests, action.Type)
}

func updatePodTemplate(template *coreV1.PodTemplateSpec, action domain.Action) (bool, error) {
	container, err := findTargetContainer(template.Spec.Containers, action.Container)
	if err != nil {
		return false, err
	}
	if container.Resources.Requests == nil {
		container.Resources.Requests = coreV1.ResourceList{}
	}
	switch action.Type {
	case domain.ActionRightsizeWorkloadCPU:
		container.Resources.Requests[coreV1.ResourceCPU] = *apiresource.NewMilliQuantity(action.DesiredValue, apiresource.DecimalSI)
	case domain.ActionRightsizeWorkloadMemory:
		container.Resources.Requests[coreV1.ResourceMemory] = *apiresource.NewQuantity(action.DesiredValue, apiresource.BinarySI)
	default:
		return false, fmt.Errorf("unsupported workload action type %q", action.Type)
	}
	return true, nil
}

func findTargetContainer(containers []coreV1.Container, target string) (*coreV1.Container, error) {
	target = strings.TrimSpace(target)
	if target != "" {
		for index := range containers {
			if containers[index].Name == target {
				return &containers[index], nil
			}
		}
		return nil, fmt.Errorf("container %q not found", target)
	}
	if len(containers) != 1 {
		return nil, fmt.Errorf("workload has %d containers; an explicit container target is required", len(containers))
	}
	return &containers[0], nil
}

func requestedValue(requests coreV1.ResourceList, actionType domain.ActionType) (int64, error) {
	var resourceName coreV1.ResourceName
	switch actionType {
	case domain.ActionRightsizeWorkloadCPU:
		resourceName = coreV1.ResourceCPU
	case domain.ActionRightsizeWorkloadMemory:
		resourceName = coreV1.ResourceMemory
	default:
		return 0, fmt.Errorf("unsupported workload action type %q", actionType)
	}
	quantity, ok := requests[resourceName]
	if !ok {
		return 0, fmt.Errorf("workload has no %s request", resourceName)
	}
	if resourceName == coreV1.ResourceCPU {
		return quantity.MilliValue(), nil
	}
	return quantity.Value(), nil
}

func parseWorkloadReference(reference string) (string, string, error) {
	parts := strings.Split(strings.Trim(reference, "/"), "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("workload must use namespace/name format")
	}
	return parts[0], parts[1], nil
}

func successResult(execution domain.ExecutionRecord, message string) domain.ExecutionResult {
	return domain.ExecutionResult{
		ExecutionID: execution.ID,
		ActionID:    execution.ActionID,
		Provider:    execution.Provider,
		Cluster:     execution.Cluster,
		Status:      domain.ExecutionResultSucceeded,
		Message:     message,
	}
}

var _ domain.ProviderExecutor = (*WorkloadExecutor)(nil)
var _ domain.StateReader = (*WorkloadExecutor)(nil)
