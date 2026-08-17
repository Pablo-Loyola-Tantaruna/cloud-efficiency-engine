//go:build live_e2e && aws_sdk_v2

package e2e

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
	awsprovider "cloud-efficiency-engine/internal/providers/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
)

func TestLiveAWSEKSFinOpsLifecycle(t *testing.T) {
	requireLiveMutation(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	region := requiredEnv(t, "AWS_E2E_REGION")
	cluster := requiredEnv(t, "AWS_E2E_CLUSTER")
	nodeGroup := requiredEnv(t, "AWS_E2E_NODE_GROUP")

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		t.Fatalf("load AWS config: %v", err)
	}

	client := awsprovider.NewEKSSDKNodeGroupClient(eks.NewFromConfig(cfg), 20*time.Minute)
	reader := awsprovider.NewEKSStateReader(client)
	executor := awsprovider.NewEKSExecutor(client)

	original, err := client.DescribeDesiredSize(ctx, cluster, nodeGroup)
	if err != nil {
		t.Fatalf("read AWS EKS desired size: %v", err)
	}
	if original <= 1 {
		t.Skipf("AWS EKS node group %s has desired size %d; refusing to scale below 1", nodeGroup, original)
	}
	desired := original - 1

	plan, action := newReduceNodeGroupPlan(domain.CloudProviderAWS, cluster, nodeGroup, original, desired)
	defer restoreNodeGroup(context.Background(), t, domain.CloudProviderAWS, cluster, nodeGroup, desired, original, executor, reader)

	record, verification, metrics := executeAndVerify(t, ctx, plan, action, executor, reader)
	if record.Status != domain.ExecutionStatusSucceeded {
		t.Fatalf("expected successful AWS execution, got %s", record.Status)
	}
	if metrics == nil {
		t.Fatal("runtime metrics collector was not created")
	}
	if verification.ActualValue != desired {
		t.Fatalf("AWS verification expected %d, got %d", desired, verification.ActualValue)
	}

	executeIdempotentAndVerify(t, ctx, plan, action, executor, reader)
}
