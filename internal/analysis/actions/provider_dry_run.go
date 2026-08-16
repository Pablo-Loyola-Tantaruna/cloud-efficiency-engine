package actions

import (
	"fmt"

	"cloud-efficiency-engine/internal/domain"
)

// ProviderDryRunAdapter renders a provider-specific change preview without
// contacting or mutating the cloud provider.
type ProviderDryRunAdapter interface {
	Provider() domain.CloudProvider
	Render(operation ExecutionOperation) (ProviderChange, error)
}

type ProviderChange struct {
	Provider     domain.CloudProvider `json:"provider"`
	ActionID     string               `json:"actionId"`
	Operation    string               `json:"operation"`
	Target       string               `json:"target"`
	CurrentValue int64                `json:"currentValue"`
	DesiredValue int64                `json:"desiredValue"`
	Command      string               `json:"command"`
	DryRun       bool                 `json:"dryRun"`
}

func RenderProviderDryRun(operation ExecutionOperation) (ProviderChange, error) {
	adapters := []ProviderDryRunAdapter{
		AWSDryRunAdapter{},
		AzureDryRunAdapter{},
		GCPDryRunAdapter{},
	}
	for _, adapter := range adapters {
		if adapter.Provider() == operation.Provider {
			return adapter.Render(operation)
		}
	}
	return ProviderChange{}, fmt.Errorf("unsupported provider %q", operation.Provider)
}

func renderWorkloadTarget(operation ExecutionOperation) string {
	if operation.Workload == "" {
		return operation.Cluster
	}
	return operation.Cluster + "/" + operation.Workload
}

type AWSDryRunAdapter struct{}

func (AWSDryRunAdapter) Provider() domain.CloudProvider { return domain.CloudProviderAWS }

func (AWSDryRunAdapter) Render(operation ExecutionOperation) (ProviderChange, error) {
	if operation.NodeGroup == "" && operation.Type == domain.ActionReduceNodeGroup {
		return ProviderChange{}, fmt.Errorf("AWS node group action requires node group")
	}

	change := ProviderChange{
		Provider:     domain.CloudProviderAWS,
		ActionID:     operation.ActionID,
		CurrentValue: operation.CurrentValue,
		DesiredValue: operation.DesiredValue,
		DryRun:       true,
	}

	switch operation.Type {
	case domain.ActionReduceNodeGroup:
		change.Operation = "UPDATE_EKS_NODE_GROUP_DESIRED_SIZE"
		change.Target = operation.Cluster + "/" + operation.NodeGroup
		change.Command = fmt.Sprintf("aws eks update-nodegroup-config --cluster-name %q --nodegroup-name %q --scaling-config desiredSize=%d", operation.Cluster, operation.NodeGroup, operation.DesiredValue)
	case domain.ActionRightsizeWorkloadCPU:
		change.Operation = "UPDATE_KUBERNETES_CPU_REQUEST"
		change.Target = renderWorkloadTarget(operation)
		change.Command = fmt.Sprintf("kubectl -n <namespace> set resources <workload> --requests=cpu=%dm", operation.DesiredValue)
	case domain.ActionRightsizeWorkloadMemory:
		change.Operation = "UPDATE_KUBERNETES_MEMORY_REQUEST"
		change.Target = renderWorkloadTarget(operation)
		change.Command = fmt.Sprintf("kubectl -n <namespace> set resources <workload> --requests=memory=<desired-bytes>")
	default:
		return ProviderChange{}, fmt.Errorf("unsupported AWS action type %q", operation.Type)
	}
	return change, nil
}

type AzureDryRunAdapter struct{}

func (AzureDryRunAdapter) Provider() domain.CloudProvider { return domain.CloudProviderAzure }

func (AzureDryRunAdapter) Render(operation ExecutionOperation) (ProviderChange, error) {
	if operation.NodeGroup == "" && operation.Type == domain.ActionReduceNodeGroup {
		return ProviderChange{}, fmt.Errorf("Azure node pool action requires node pool")
	}
	change := ProviderChange{
		Provider:     domain.CloudProviderAzure,
		ActionID:     operation.ActionID,
		CurrentValue: operation.CurrentValue,
		DesiredValue: operation.DesiredValue,
		DryRun:       true,
	}
	switch operation.Type {
	case domain.ActionReduceNodeGroup:
		change.Operation = "UPDATE_AKS_NODE_POOL_COUNT"
		change.Target = operation.Cluster + "/" + operation.NodeGroup
		change.Command = fmt.Sprintf("az aks nodepool update --resource-group <resource-group> --cluster-name %q --name %q --node-count %d", operation.Cluster, operation.NodeGroup, operation.DesiredValue)
	case domain.ActionRightsizeWorkloadCPU:
		change.Operation = "UPDATE_KUBERNETES_CPU_REQUEST"
		change.Target = renderWorkloadTarget(operation)
		change.Command = fmt.Sprintf("kubectl -n <namespace> set resources <workload> --requests=cpu=%dm", operation.DesiredValue)
	case domain.ActionRightsizeWorkloadMemory:
		change.Operation = "UPDATE_KUBERNETES_MEMORY_REQUEST"
		change.Target = renderWorkloadTarget(operation)
		change.Command = "kubectl -n <namespace> set resources <workload> --requests=memory=<desired-bytes>"
	default:
		return ProviderChange{}, fmt.Errorf("unsupported Azure action type %q", operation.Type)
	}
	return change, nil
}

type GCPDryRunAdapter struct{}

func (GCPDryRunAdapter) Provider() domain.CloudProvider { return domain.CloudProviderGCP }

func (GCPDryRunAdapter) Render(operation ExecutionOperation) (ProviderChange, error) {
	if operation.NodeGroup == "" && operation.Type == domain.ActionReduceNodeGroup {
		return ProviderChange{}, fmt.Errorf("GCP node pool action requires node pool")
	}
	change := ProviderChange{
		Provider:     domain.CloudProviderGCP,
		ActionID:     operation.ActionID,
		CurrentValue: operation.CurrentValue,
		DesiredValue: operation.DesiredValue,
		DryRun:       true,
	}
	switch operation.Type {
	case domain.ActionReduceNodeGroup:
		change.Operation = "RESIZE_GKE_NODE_POOL"
		change.Target = operation.Cluster + "/" + operation.NodeGroup
		change.Command = fmt.Sprintf("gcloud container clusters resize %q --node-pool=%q --num-nodes=%d --region=<region>", operation.Cluster, operation.NodeGroup, operation.DesiredValue)
	case domain.ActionRightsizeWorkloadCPU:
		change.Operation = "UPDATE_KUBERNETES_CPU_REQUEST"
		change.Target = renderWorkloadTarget(operation)
		change.Command = fmt.Sprintf("kubectl -n <namespace> set resources <workload> --requests=cpu=%dm", operation.DesiredValue)
	case domain.ActionRightsizeWorkloadMemory:
		change.Operation = "UPDATE_KUBERNETES_MEMORY_REQUEST"
		change.Target = renderWorkloadTarget(operation)
		change.Command = "kubectl -n <namespace> set resources <workload> --requests=memory=<desired-bytes>"
	default:
		return ProviderChange{}, fmt.Errorf("unsupported GCP action type %q", operation.Type)
	}
	return change, nil
}

func RenderExecutionProviderChanges(plan ExecutionPlan) ([]ProviderChange, error) {
	if plan.Mode != ExecutionModeDryRun {
		return nil, fmt.Errorf("execution plan mode must be %q", ExecutionModeDryRun)
	}
	changes := make([]ProviderChange, 0, len(plan.Operations))
	for _, operation := range plan.Operations {
		change, err := RenderProviderDryRun(operation)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}
