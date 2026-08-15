package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type NodeCapacityResolver struct {
	client EC2API
}

func NewNodeCapacityResolver(
	client EC2API,
) *NodeCapacityResolver {

	return &NodeCapacityResolver{
		client: client,
	}
}

type InstanceCapacity struct {
	InstanceType string

	VCPUs int64

	MemoryBytes int64
}

func (r *NodeCapacityResolver) Resolve(
	ctx context.Context,
	instanceType string,
) (InstanceCapacity, error) {

	result, err :=
		r.ResolveMany(
			ctx,
			[]string{
				instanceType,
			},
		)

	if err != nil {
		return InstanceCapacity{}, err
	}

	capacity, ok :=
		result[instanceType]

	if !ok {
		return InstanceCapacity{},
			fmt.Errorf(
				"AWS instance type %q not found",
				instanceType,
			)
	}

	return capacity, nil
}

func (r *NodeCapacityResolver) ResolveMany(
	ctx context.Context,
	instanceTypes []string,
) (map[string]InstanceCapacity, error) {

	if r.client == nil {
		return nil,
			fmt.Errorf(
				"AWS EC2 client is not configured",
			)
	}

	if len(instanceTypes) == 0 {
		return map[string]InstanceCapacity{}, nil
	}

	uniqueTypes :=
		make(
			map[string]struct{},
			len(instanceTypes),
		)

	for _, instanceType := range instanceTypes {

		if instanceType == "" {
			continue
		}

		uniqueTypes[instanceType] =
			struct{}{}
	}

	requestTypes :=
		make(
			[]ec2types.InstanceType,
			0,
			len(uniqueTypes),
		)

	for instanceType := range uniqueTypes {

		requestTypes =
			append(
				requestTypes,
				ec2types.InstanceType(
					instanceType,
				),
			)
	}

	output, err :=
		r.client.DescribeInstanceTypes(
			ctx,
			&ec2.DescribeInstanceTypesInput{
				InstanceTypes: requestTypes,
			},
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"describe AWS instance types: %w",
				err,
			)
	}

	if output == nil {
		return nil,
			fmt.Errorf(
				"AWS DescribeInstanceTypes returned an empty response",
			)
	}

	result :=
		make(
			map[string]InstanceCapacity,
			len(output.InstanceTypes),
		)

	for _, info := range output.InstanceTypes {

		instanceType :=
			string(
				info.InstanceType,
			)

		if instanceType == "" {
			continue
		}

		if info.VCpuInfo == nil ||
			info.VCpuInfo.DefaultVCpus == nil {

			return nil,
				fmt.Errorf(
					"AWS instance type %q has no vCPU information",
					instanceType,
				)
		}

		if info.MemoryInfo == nil ||
			info.MemoryInfo.SizeInMiB == nil {

			return nil,
				fmt.Errorf(
					"AWS instance type %q has no memory information",
					instanceType,
				)
		}

		result[instanceType] =
			InstanceCapacity{
				InstanceType: instanceType,

				VCPUs: int64(
					aws.ToInt32(
						info.VCpuInfo.
							DefaultVCpus,
					),
				),

				MemoryBytes: int64(
					aws.ToInt64(
						info.MemoryInfo.
							SizeInMiB,
					),
				) *
					1024 *
					1024,
			}
	}

	for instanceType := range uniqueTypes {

		if _, ok := result[instanceType]; !ok {

			return nil,
				fmt.Errorf(
					"AWS instance type %q was not returned",
					instanceType,
				)
		}
	}

	return result, nil
}
