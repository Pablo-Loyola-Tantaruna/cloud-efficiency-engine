package aws

import (
	"context"
	"fmt"

	"cloud-efficiency-engine/internal/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type Node struct {
	InstanceID string

	InstanceType string

	Region string
}

type NodeInventory struct {
	client EC2API
}

func NewNodeInventory(
	client EC2API,
) *NodeInventory {

	return &NodeInventory{
		client: client,
	}
}

func (n *NodeInventory) ListRunningNodes(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) ([]Node, error) {

	if n.client == nil {

		return nil,
			fmt.Errorf(
				"AWS EC2 client is not configured",
			)
	}

	if analysisContext.Region == "" {

		return nil,
			fmt.Errorf(
				"AWS region must not be empty",
			)
	}

	filters :=
		[]ec2types.Filter{
			{
				Name: aws.String(
					"instance-state-name",
				),

				Values: []string{
					"running",
				},
			},
		}

	if analysisContext.ClusterName != "" {

		filters =
			append(
				filters,
				ec2types.Filter{
					Name: aws.String(
						"tag:aws:eks:cluster-name",
					),

					Values: []string{
						analysisContext.ClusterName,
					},
				},
			)
	}

	result :=
		make(
			[]Node,
			0,
		)

	var nextToken *string

	for {

		output, err :=
			n.client.DescribeInstances(
				ctx,
				&ec2.DescribeInstancesInput{
					Filters: filters,

					NextToken: nextToken,
				},
			)

		if err != nil {

			return nil,
				fmt.Errorf(
					"describe AWS nodes: %w",
					err,
				)
		}

		if output == nil {

			break
		}

		for _, reservation := range output.Reservations {

			for _, instance := range reservation.Instances {

				if instance.InstanceId == nil ||
					instance.InstanceType == "" {

					continue
				}

				if instance.InstanceLifecycle ==
					ec2types.InstanceLifecycleTypeSpot {

					continue
				}

				result =
					append(
						result,
						Node{
							InstanceID: aws.ToString(
								instance.InstanceId,
							),

							InstanceType: string(
								instance.InstanceType,
							),

							Region: analysisContext.Region,
						},
					)
			}
		}

		if output.NextToken == nil ||
			aws.ToString(
				output.NextToken,
			) == "" {

			break
		}

		nextToken =
			output.NextToken
	}

	return result, nil
}
