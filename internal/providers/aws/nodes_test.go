package aws

import (
	"context"
	"testing"

	"cloud-efficiency-engine/internal/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type ec2ClientMock struct {
	describeInstancesCalled bool

	describeTypesCalled bool
}

func (m *ec2ClientMock) DescribeInstances(
	ctx context.Context,
	params *ec2.DescribeInstancesInput,
	optFns ...func(*ec2.Options),
) (*ec2.DescribeInstancesOutput, error) {

	m.describeInstancesCalled = true

	return &ec2.DescribeInstancesOutput{
		Reservations: []ec2types.Reservation{
			{
				Instances: []ec2types.Instance{
					{
						InstanceId: aws.String(
							"i-123456789",
						),

						InstanceType: ec2types.InstanceTypeM5Large,
					},
				},
			},
		},
	}, nil
}

func (m *ec2ClientMock) DescribeInstanceTypes(
	ctx context.Context,
	params *ec2.DescribeInstanceTypesInput,
	optFns ...func(*ec2.Options),
) (*ec2.DescribeInstanceTypesOutput, error) {

	m.describeTypesCalled = true

	return &ec2.DescribeInstanceTypesOutput{
		InstanceTypes: []ec2types.InstanceTypeInfo{
			{
				InstanceType: ec2types.InstanceTypeM5Large,

				VCpuInfo: &ec2types.VCpuInfo{
					DefaultVCpus: aws.Int32(2),
				},

				MemoryInfo: &ec2types.MemoryInfo{
					SizeInMiB: aws.Int64(8192),
				},
			},
		},
	}, nil
}

func TestNodeInventory_ShouldListRunningNodes(
	t *testing.T,
) {

	client :=
		&ec2ClientMock{}

	inventory :=
		NewNodeInventory(
			client,
		)

	nodes, err :=
		inventory.ListRunningNodes(
			context.Background(),
			domain.AnalysisContext{
				Provider: domain.CloudProviderAWS,

				Region: "us-east-1",
			},
		)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if len(nodes) != 1 {
		t.Fatalf(
			"expected 1 node, got %d",
			len(nodes),
		)
	}

	if nodes[0].InstanceType !=
		string(
			ec2types.InstanceTypeM5Large,
		) {

		t.Fatalf(
			"unexpected instance type: %s",
			nodes[0].InstanceType,
		)
	}
}
