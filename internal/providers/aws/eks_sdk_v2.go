//go:build aws_sdk_v2

package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	exk "github.com/aws/aws-sdk-go-v2/service/eks"
	exktypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
)

// NewEKSSDKNodeGroupClient adapts the AWS SDK v2 EKS client to the provider-neutral
// EKSNodeGroupClient contract. The build tag keeps the SDK boundary explicit until
// the module dependency is synchronized in go.sum.
func NewEKSSDKNodeGroupClient(client *exk.Client, maxWait time.Duration) *EKSSDKNodeGroupClient {
	if maxWait <= 0 {
		maxWait = 15 * time.Minute
	}
	return &EKSSDKNodeGroupClient{client: client, maxWait: maxWait}
}

type EKSSDKNodeGroupClient struct {
	client  *exk.Client
	maxWait time.Duration
}

func (c *EKSSDKNodeGroupClient) DescribeDesiredSize(ctx context.Context, cluster, nodeGroup string) (int64, error) {
	if c == nil || c.client == nil {
		return 0, fmt.Errorf("EKS SDK client must not be nil")
	}
	output, err := c.client.DescribeNodegroup(ctx, &exk.DescribeNodegroupInput{
		ClusterName:   aws.String(cluster),
		NodegroupName: aws.String(nodeGroup),
	})
	if err != nil {
		return 0, err
	}
	if output.Nodegroup == nil || output.Nodegroup.ScalingConfig == nil || output.Nodegroup.ScalingConfig.DesiredSize == nil {
		return 0, fmt.Errorf("EKS node group %q returned no desired size", nodeGroup)
	}
	return int64(*output.Nodegroup.ScalingConfig.DesiredSize), nil
}

func (c *EKSSDKNodeGroupClient) UpdateDesiredSize(ctx context.Context, cluster, nodeGroup string, desired int64, clientRequestToken string) (string, error) {
	if c == nil || c.client == nil {
		return "", fmt.Errorf("EKS SDK client must not be nil")
	}
	output, err := c.client.UpdateNodegroupConfig(ctx, &exk.UpdateNodegroupConfigInput{
		ClusterName:        aws.String(cluster),
		NodegroupName:      aws.String(nodeGroup),
		ClientRequestToken: aws.String(clientRequestToken),
		ScalingConfig:      &exktypes.NodegroupScalingConfig{DesiredSize: aws.Int32(int32(desired))},
	})
	if err != nil {
		return "", err
	}
	if output.Update == nil || output.Update.Id == nil {
		return "", fmt.Errorf("EKS node group %q returned no update id", nodeGroup)
	}
	return aws.ToString(output.Update.Id), nil
}

func (c *EKSSDKNodeGroupClient) WaitForUpdate(ctx context.Context, cluster, nodeGroup, updateID string) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("EKS SDK client must not be nil")
	}
	waitCtx, cancel := context.WithTimeout(ctx, c.maxWait)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		output, err := c.client.DescribeUpdate(waitCtx, &exk.DescribeUpdateInput{
			Name:          aws.String(cluster),
			NodegroupName: aws.String(nodeGroup),
			UpdateId:      aws.String(updateID),
		})
		if err != nil {
			return err
		}
		if output.Update == nil {
			return fmt.Errorf("EKS update %q returned no update payload", updateID)
		}

		switch output.Update.Status {
		case exktypes.UpdateStatusSuccessful:
			return nil
		case exktypes.UpdateStatusFailed, exktypes.UpdateStatusCancelled:
			return fmt.Errorf("EKS update %q finished with status %s", updateID, output.Update.Status)
		case exktypes.UpdateStatusInProgress:
			select {
			case <-waitCtx.Done():
				return waitCtx.Err()
			case <-ticker.C:
			}
		default:
			return fmt.Errorf("EKS update %q returned unsupported status %s", updateID, output.Update.Status)
		}
	}
}
