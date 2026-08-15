package aws

import (
	"context"
	"fmt"

	"cloud-efficiency-engine/internal/analysis/capacity"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

type CapacityProvider struct {
	nodeInventory    *NodeInventory
	capacityResolver *NodeCapacityResolver
}

func NewCapacityProvider(
	ec2Client EC2API,
) *CapacityProvider {

	return &CapacityProvider{
		nodeInventory: NewNodeInventory(
			ec2Client,
		),

		capacityResolver: NewNodeCapacityResolver(
			ec2Client,
		),
	}
}

func (
	p *CapacityProvider,
) GetCapacity(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (cost.ClusterCapacity, error) {

	if p == nil {
		return cost.ClusterCapacity{},
			fmt.Errorf(
				"AWS capacity provider is not configured",
			)
	}

	if p.nodeInventory == nil {
		return cost.ClusterCapacity{},
			fmt.Errorf(
				"AWS node inventory is not configured",
			)
	}

	if p.capacityResolver == nil {
		return cost.ClusterCapacity{},
			fmt.Errorf(
				"AWS capacity resolver is not configured",
			)
	}

	nodes, err :=
		p.nodeInventory.ListRunningNodes(
			ctx,
			analysisContext,
		)

	if err != nil {
		return cost.ClusterCapacity{},
			err
	}

	if len(nodes) == 0 {
		return cost.ClusterCapacity{},
			fmt.Errorf(
				"no running AWS nodes found",
			)
	}

	instanceTypes :=
		make(
			[]string,
			0,
			len(nodes),
		)

	for _, node := range nodes {

		if node.InstanceType == "" {
			continue
		}

		instanceTypes =
			append(
				instanceTypes,
				node.InstanceType,
			)
	}

	if len(instanceTypes) == 0 {
		return cost.ClusterCapacity{},
			fmt.Errorf(
				"no AWS instance types found",
			)
	}

	capacities, err :=
		p.capacityResolver.ResolveMany(
			ctx,
			instanceTypes,
		)

	if err != nil {
		return cost.ClusterCapacity{},
			err
	}

	var (
		cpuCapacityMillicores int64

		memoryCapacityBytes int64
	)

	for _, node := range nodes {

		capacity, ok :=
			capacities[node.InstanceType]

		if !ok {
			return cost.ClusterCapacity{},
				fmt.Errorf(
					"capacity not found for AWS instance type %q",
					node.InstanceType,
				)
		}

		cpuCapacityMillicores +=
			capacity.VCPUs *
				1000

		memoryCapacityBytes +=
			capacity.MemoryBytes
	}

	if cpuCapacityMillicores <= 0 {
		return cost.ClusterCapacity{},
			fmt.Errorf(
				"AWS cluster CPU capacity must be greater than zero",
			)
	}

	if memoryCapacityBytes <= 0 {
		return cost.ClusterCapacity{},
			fmt.Errorf(
				"AWS cluster memory capacity must be greater than zero",
			)
	}

	return cost.ClusterCapacity{
		CPUCapacityMillicores: cpuCapacityMillicores,

		MemoryCapacityBytes: memoryCapacityBytes,
	}, nil
}

var _ capacity.Provider = (*CapacityProvider)(nil)
