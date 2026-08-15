package aws

import (
	"context"
	"testing"

	"cloud-efficiency-engine/internal/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type capacityEC2Mock struct {
}

func (
	m *capacityEC2Mock,
) DescribeInstances(
	ctx context.Context,
	params *ec2.DescribeInstancesInput,
	optFns ...func(*ec2.Options),
) (
	*ec2.DescribeInstancesOutput,
	error,
) {

	return &ec2.DescribeInstancesOutput{
		Reservations: []ec2types.Reservation{
			{
				Instances: []ec2types.Instance{
					{
						InstanceId: aws.String(
							"i-123",
						),

						InstanceType: ec2types.InstanceTypeM5Large,
					},

					{
						InstanceId: aws.String(
							"i-456",
						),

						InstanceType: ec2types.InstanceTypeM5Large,
					},
				},
			},
		},
	}, nil
}

func (
	m *capacityEC2Mock,
) DescribeInstanceTypes(
	ctx context.Context,
	params *ec2.DescribeInstanceTypesInput,
	optFns ...func(*ec2.Options),
) (
	*ec2.DescribeInstanceTypesOutput,
	error,
) {

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

func TestCapacityProvider_ShouldCalculateClusterCapacity(
	t *testing.T,
) {

	provider :=
		NewCapacityProvider(
			&capacityEC2Mock{},
		)

	result, err :=
		provider.GetCapacity(
			context.Background(),
			domain.AnalysisContext{
				Provider: domain.CloudProviderAWS,

				Environment: "production",

				Region: "us-east-1",

				ClusterName: "production-cluster",
			},
		)

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if result.CPUCapacityMillicores !=
		4000 {

		t.Fatalf(
			"expected CPU capacity 4000m, got %d",
			result.CPUCapacityMillicores,
		)
	}

	expectedMemory :=
		int64(
			16 *
				1024 *
				1024 *
				1024,
		)

	if result.MemoryCapacityBytes !=
		expectedMemory {

		t.Fatalf(
			"expected memory capacity %d, got %d",
			expectedMemory,
			result.MemoryCapacityBytes,
		)
	}
}

func TestCapacityProvider_ShouldRejectMissingRegion(
	t *testing.T,
) {

	provider :=
		NewCapacityProvider(
			&capacityEC2Mock{},
		)

	_, err :=
		provider.GetCapacity(
			context.Background(),
			domain.AnalysisContext{
				Provider: domain.CloudProviderAWS,
			},
		)

	if err == nil {

		t.Fatal(
			"expected error",
		)
	}
}
