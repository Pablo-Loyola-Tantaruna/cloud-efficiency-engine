package azure

import (
	"context"
	"fmt"
	"strings"

	"cloud-efficiency-engine/internal/domain"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v9"
)

// AKSNodePoolClient isolates Azure SDK details from the action engine.
type AKSNodePoolClient interface {
	GetNodePool(ctx context.Context, resourceGroup, cluster, nodePool string) (int64, error)
	SetNodePoolSize(ctx context.Context, resourceGroup, cluster, nodePool string, desired int64) error
}

type AKSExecutor struct {
	client        AKSNodePoolClient
	resourceGroup string
}

func NewAKSExecutor(client AKSNodePoolClient, resourceGroup string) *AKSExecutor {
	return &AKSExecutor{client: client, resourceGroup: strings.TrimSpace(resourceGroup)}
}

func (e *AKSExecutor) Execute(ctx context.Context, action domain.Action, execution domain.ExecutionRecord) (domain.ExecutionResult, error) {
	if e == nil || e.client == nil {
		return domain.ExecutionResult{}, fmt.Errorf("azure aks executor client must not be nil")
	}
	if e.resourceGroup == "" {
		return domain.ExecutionResult{}, fmt.Errorf("azure aks resource group must not be empty")
	}
	if execution.Status != domain.ExecutionStatusRunning {
		return domain.ExecutionResult{}, fmt.Errorf("execution %q must be RUNNING", execution.ID)
	}
	if action.Type != domain.ActionReduceNodeGroup {
		return domain.ExecutionResult{}, fmt.Errorf("azure aks executor does not support action type %q", action.Type)
	}
	if action.Provider != domain.CloudProviderAzure || execution.Provider != domain.CloudProviderAzure {
		return domain.ExecutionResult{}, fmt.Errorf("azure aks executor requires Azure provider")
	}
	if action.Cluster != execution.Cluster || action.ID != execution.ActionID {
		return domain.ExecutionResult{}, fmt.Errorf("action %q does not match execution %q", action.ID, execution.ID)
	}
	if strings.TrimSpace(action.NodeGroup) == "" {
		return domain.ExecutionResult{}, fmt.Errorf("node pool must not be empty")
	}
	if action.CurrentValue != execution.CurrentValue || action.DesiredValue != execution.DesiredValue {
		return domain.ExecutionResult{}, fmt.Errorf("action values do not match execution values")
	}

	observedBefore, err := e.client.GetNodePool(ctx, e.resourceGroup, action.Cluster, action.NodeGroup)
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("describe AKS node pool before execution: %w", err)
	}
	if observedBefore != action.CurrentValue {
		return domain.ExecutionResult{}, fmt.Errorf("AKS node pool %q drifted before execution: expected %d, observed %d", action.NodeGroup, action.CurrentValue, observedBefore)
	}

	if err := e.client.SetNodePoolSize(ctx, e.resourceGroup, action.Cluster, action.NodeGroup, action.DesiredValue); err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("update AKS node pool %q: %w", action.NodeGroup, err)
	}

	result := domain.ExecutionResult{
		Status:       domain.ExecutionResultSucceeded,
		ExecutionID:  execution.ID,
		Provider:     domain.CloudProviderAzure,
		Cluster:      action.Cluster,
		ActionID:     action.ID,
		BeforeValue:  observedBefore,
		DesiredValue: action.DesiredValue,
		Message:      fmt.Sprintf("AKS node pool %q desired size updated from %d to %d", action.NodeGroup, observedBefore, action.DesiredValue),
	}
	if err := result.Validate(); err != nil {
		return domain.ExecutionResult{}, err
	}
	return result, nil
}

// ARMNodePoolClient adapts the Azure SDK v9 AgentPoolsClient to AKSNodePoolClient.
type ARMNodePoolClient struct {
	client *armcontainerservice.AgentPoolsClient
}

func NewARMNodePoolClient(client *armcontainerservice.AgentPoolsClient) *ARMNodePoolClient {
	return &ARMNodePoolClient{client: client}
}

func (c *ARMNodePoolClient) GetNodePool(ctx context.Context, resourceGroup, cluster, nodePool string) (int64, error) {
	if c == nil || c.client == nil {
		return 0, fmt.Errorf("Azure agent pools client is not configured")
	}
	response, err := c.client.Get(ctx, resourceGroup, cluster, nodePool, nil)
	if err != nil {
		return 0, err
	}
	if response.Properties == nil || response.Properties.Count == nil {
		return 0, fmt.Errorf("Azure AKS node pool %q returned no node count", nodePool)
	}
	return int64(*response.Properties.Count), nil
}

func (c *ARMNodePoolClient) SetNodePoolSize(ctx context.Context, resourceGroup, cluster, nodePool string, desired int64) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("Azure agent pools client is not configured")
	}
	if desired <= 0 {
		return fmt.Errorf("desired node count must be greater than zero")
	}
	current, err := c.GetNodePool(ctx, resourceGroup, cluster, nodePool)
	if err != nil {
		return err
	}
	response, err := c.client.BeginCreateOrUpdate(ctx, resourceGroup, cluster, nodePool, armcontainerservice.AgentPool{
		Properties: &armcontainerservice.ManagedClusterAgentPoolProfileProperties{
			Count: to.Ptr(int32(desired)),
		},
	}, nil)
	if err != nil {
		return err
	}
	if response == nil {
		return fmt.Errorf("Azure AKS node pool %q returned no operation poller", nodePool)
	}
	if _, err := response.PollUntilDone(ctx, nil); err != nil {
		return err
	}
	if desired == current {
		return nil
	}
	return nil
}

var _ domain.ProviderExecutor = (*AKSExecutor)(nil)
var _ AKSNodePoolClient = (*ARMNodePoolClient)(nil)
