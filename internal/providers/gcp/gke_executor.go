package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud-efficiency-engine/internal/domain"
	container "cloud.google.com/go/container/apiv1"
	containerpb "cloud.google.com/go/container/apiv1/containerpb"
)

// GKENodePoolClient isolates GKE API details from the action engine.
type GKENodePoolClient interface {
	GetNodePool(ctx context.Context, projectID, location, cluster, nodePool string) (int64, error)
	SetNodePoolSize(ctx context.Context, projectID, location, cluster, nodePool string, desired int64) error
}

type GKEExecutor struct {
	client    GKENodePoolClient
	projectID string
	location  string
}

func NewGKEExecutor(client GKENodePoolClient, projectID, location string) *GKEExecutor {
	return &GKEExecutor{
		client:    client,
		projectID: strings.TrimSpace(projectID),
		location:  strings.TrimSpace(location),
	}
}

func (e *GKEExecutor) Execute(ctx context.Context, action domain.Action, execution domain.ExecutionRecord) (domain.ExecutionResult, error) {
	if e == nil || e.client == nil {
		return domain.ExecutionResult{}, fmt.Errorf("gcp gke executor client must not be nil")
	}
	if e.projectID == "" || e.location == "" {
		return domain.ExecutionResult{}, fmt.Errorf("gcp project ID and location must not be empty")
	}
	if execution.Status != domain.ExecutionStatusRunning {
		return domain.ExecutionResult{}, fmt.Errorf("execution %q must be RUNNING", execution.ID)
	}
	if action.Type != domain.ActionReduceNodeGroup {
		return domain.ExecutionResult{}, fmt.Errorf("gcp gke executor does not support action type %q", action.Type)
	}
	if action.Provider != domain.CloudProviderGCP || execution.Provider != domain.CloudProviderGCP {
		return domain.ExecutionResult{}, fmt.Errorf("gcp gke executor requires GCP provider")
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

	observedBefore, err := e.client.GetNodePool(ctx, e.projectID, e.location, action.Cluster, action.NodeGroup)
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("describe GKE node pool before execution: %w", err)
	}
	if observedBefore != action.CurrentValue {
		return domain.ExecutionResult{}, fmt.Errorf("GKE node pool %q drifted before execution: expected %d, observed %d", action.NodeGroup, action.CurrentValue, observedBefore)
	}

	if err := e.client.SetNodePoolSize(ctx, e.projectID, e.location, action.Cluster, action.NodeGroup, action.DesiredValue); err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("update GKE node pool %q: %w", action.NodeGroup, err)
	}

	result := domain.ExecutionResult{
		Status:       domain.ExecutionResultSucceeded,
		ExecutionID:  execution.ID,
		Provider:     domain.CloudProviderGCP,
		Cluster:      action.Cluster,
		ActionID:     action.ID,
		BeforeValue:  observedBefore,
		DesiredValue: action.DesiredValue,
		Message:      fmt.Sprintf("GKE node pool %q desired size updated from %d to %d", action.NodeGroup, observedBefore, action.DesiredValue),
	}
	if err := result.Validate(); err != nil {
		return domain.ExecutionResult{}, err
	}
	return result, nil
}

// GKEClusterManagerClient adapts the Google Cloud Container API client.
type GKEClusterManagerClient struct {
	client  *container.ClusterManagerClient
	maxWait time.Duration
}

func NewGKEClusterManagerClient(client *container.ClusterManagerClient, maxWait time.Duration) *GKEClusterManagerClient {
	if maxWait <= 0 {
		maxWait = 15 * time.Minute
	}
	return &GKEClusterManagerClient{client: client, maxWait: maxWait}
}

func (c *GKEClusterManagerClient) GetNodePool(ctx context.Context, projectID, location, cluster, nodePool string) (int64, error) {
	if c == nil || c.client == nil {
		return 0, fmt.Errorf("GCP cluster manager client is not configured")
	}
	pool, err := c.client.GetNodePool(ctx, &containerpb.GetNodePoolRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/clusters/%s/nodePools/%s", projectID, location, cluster, nodePool),
	})
	if err != nil {
		return 0, err
	}
	if pool == nil {
		return 0, fmt.Errorf("GKE node pool %q returned no payload", nodePool)
	}

	return nodePoolSize(pool)
}

func (c *GKEClusterManagerClient) SetNodePoolSize(ctx context.Context, projectID, location, cluster, nodePool string, desired int64) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("GCP cluster manager client is not configured")
	}
	if desired <= 0 {
		return fmt.Errorf("desired node count must be greater than zero")
	}

	operation, err := c.client.SetNodePoolSize(ctx, &containerpb.SetNodePoolSizeRequest{
		Name:      fmt.Sprintf("projects/%s/locations/%s/clusters/%s/nodePools/%s", projectID, location, cluster, nodePool),
		NodeCount: int32(desired),
	})
	if err != nil {
		return err
	}
	if operation == nil || operation.GetName() == "" {
		return fmt.Errorf("GKE node pool %q returned no operation name", nodePool)
	}
	return c.waitForOperation(ctx, operation.GetName())
}

func (c *GKEClusterManagerClient) waitForOperation(ctx context.Context, operationName string) error {
	waitCtx, cancel := context.WithTimeout(ctx, c.maxWait)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		operation, err := c.client.GetOperation(waitCtx, &containerpb.GetOperationRequest{Name: operationName})
		if err != nil {
			return err
		}
		if operation == nil {
			return fmt.Errorf("GKE operation %q returned no payload", operationName)
		}

		switch operation.GetStatus() {
		case containerpb.Operation_DONE:
			if operation.GetError() != nil {
				return fmt.Errorf("GKE operation %q failed: %s", operationName, operation.GetError().GetMessage())
			}
			return nil
		case containerpb.Operation_PENDING, containerpb.Operation_RUNNING, containerpb.Operation_ABORTING:
			select {
			case <-waitCtx.Done():
				return waitCtx.Err()
			case <-ticker.C:
			}
		default:
			return fmt.Errorf("GKE operation %q returned unsupported status %s", operationName, operation.GetStatus())
		}
	}
}

func nodePoolSize(pool *containerpb.NodePool) (int64, error) {
	if pool == nil {
		return 0, fmt.Errorf("GKE node pool is nil")
	}
	count := pool.GetInitialNodeCount()
	if count <= 0 {
		return 0, fmt.Errorf("GKE node pool %q returned no positive node count", pool.GetName())
	}
	return int64(count), nil
}

var _ domain.ProviderExecutor = (*GKEExecutor)(nil)
var _ GKENodePoolClient = (*GKEClusterManagerClient)(nil)
