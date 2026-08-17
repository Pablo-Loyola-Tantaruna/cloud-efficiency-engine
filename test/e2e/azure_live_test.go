//go:build live_e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
	azureprovider "cloud-efficiency-engine/internal/providers/azure"
)

func TestLiveAzureAKSFinOpsLifecycle(t *testing.T) {
	requireLiveMutation(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	subscriptionID := requiredEnv(t, "AZURE_E2E_SUBSCRIPTION_ID")
	resourceGroup := requiredEnv(t, "AZURE_E2E_RESOURCE_GROUP")
	cluster := requiredEnv(t, "AZURE_E2E_CLUSTER")
	nodePool := requiredEnv(t, "AZURE_E2E_NODE_POOL")

	clients, err := azureprovider.NewClients(subscriptionID)
	if err != nil {
		t.Fatalf("create Azure clients: %v", err)
	}

	client := azureprovider.NewARMNodePoolClient(clients.AgentPools)
	reader := azureprovider.NewAKSStateReader(client, resourceGroup)
	executor := azureprovider.NewAKSExecutor(client, resourceGroup)

	original, err := client.GetNodePool(ctx, resourceGroup, cluster, nodePool)
	if err != nil {
		t.Fatalf("read Azure AKS node count: %v", err)
	}
	if original <= 1 {
		t.Skipf("Azure AKS node pool %s has desired size %d; refusing to scale below 1", nodePool, original)
	}
	desired := original - 1

	plan, action := newReduceNodeGroupPlan(domain.CloudProviderAzure, cluster, nodePool, original, desired)
	defer restoreNodeGroup(context.Background(), t, domain.CloudProviderAzure, cluster, nodePool, desired, original, executor, reader)

	record, verification, metrics := executeAndVerify(t, ctx, plan, action, executor, reader)
	if record.Status != domain.ExecutionStatusSucceeded {
		t.Fatalf("expected successful Azure execution, got %s", record.Status)
	}
	if metrics == nil {
		t.Fatal("runtime metrics collector was not created")
	}
	if verification.ActualValue != desired {
		t.Fatalf("Azure verification expected %d, got %d", desired, verification.ActualValue)
	}

	executeIdempotentAndVerify(t, ctx, plan, action, executor, reader)
}
