//go:build live_e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
	gcpprovider "cloud-efficiency-engine/internal/providers/gcp"
)

func TestLiveGCPGKEFinOpsLifecycle(t *testing.T) {
	requireLiveMutation(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	projectID := requiredEnv(t, "GCP_E2E_PROJECT_ID")
	location := requiredEnv(t, "GCP_E2E_LOCATION")
	cluster := requiredEnv(t, "GCP_E2E_CLUSTER")
	nodePool := requiredEnv(t, "GCP_E2E_NODE_POOL")

	clients, err := gcpprovider.NewClients(ctx, projectID)
	if err != nil {
		t.Fatalf("create GCP clients: %v", err)
	}
	defer clients.Close()

	client := gcpprovider.NewGKEClusterManagerClient(clients.ClusterManager, 20*time.Minute)
	reader := gcpprovider.NewGKEStateReader(client, projectID, location)
	executor := gcpprovider.NewGKEExecutor(client, projectID, location)

	original, err := client.GetNodePool(ctx, projectID, location, cluster, nodePool)
	if err != nil {
		t.Fatalf("read GCP GKE node count: %v", err)
	}
	if original <= 1 {
		t.Skipf("GCP GKE node pool %s has desired size %d; refusing to scale below 1", nodePool, original)
	}
	desired := original - 1

	plan, action := newReduceNodeGroupPlan(domain.CloudProviderGCP, cluster, nodePool, original, desired)
	defer restoreNodeGroup(context.Background(), t, domain.CloudProviderGCP, cluster, nodePool, desired, original, executor, reader)

	record, verification, metrics := executeAndVerify(t, ctx, plan, action, executor, reader)
	if record.Status != domain.ExecutionStatusSucceeded {
		t.Fatalf("expected successful GCP execution, got %s", record.Status)
	}
	if metrics == nil {
		t.Fatal("runtime metrics collector was not created")
	}
	if verification.ActualValue != desired {
		t.Fatalf("GCP verification expected %d, got %d", desired, verification.ActualValue)
	}

	executeIdempotentAndVerify(t, ctx, plan, action, executor, reader)
}
