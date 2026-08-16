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

func NewCapacityProvider(ec2Client EC2API) *CapacityProvider {
	return &CapacityProvider{
		nodeInventory:    NewNodeInventory(ec2Client),
		capacityResolver: NewNodeCapacityResolver(ec2Client),
	}
}

func (p *CapacityProvider) GetCapacity(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (cost.ClusterCapacity, error) {
	if p == nil || p.nodeInventory == nil || p.capacityResolver == nil {
		return cost.ClusterCapacity{}, fmt.Errorf("AWS capacity provider is not configured")
	}

	nodes, err := p.nodeInventory.ListRunningNodes(ctx, analysisContext)
	if err != nil {
		return cost.ClusterCapacity{}, err
	}
	if len(nodes) == 0 {
		return cost.ClusterCapacity{}, fmt.Errorf("no running AWS nodes found")
	}

	instanceTypes := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if node.InstanceType != "" {
			instanceTypes = append(instanceTypes, node.InstanceType)
		}
	}
	if len(instanceTypes) == 0 {
		return cost.ClusterCapacity{}, fmt.Errorf("no AWS instance types found")
	}

	capacities, err := p.capacityResolver.ResolveMany(ctx, instanceTypes)
	if err != nil {
		return cost.ClusterCapacity{}, err
	}

	var cpuCapacityMillicores int64
	var memoryCapacityBytes int64
	for _, node := range nodes {
		capacity, ok := capacities[node.InstanceType]
		if !ok {
			return cost.ClusterCapacity{}, fmt.Errorf("capacity not found for AWS instance type %q", node.InstanceType)
		}
		cpuCapacityMillicores += capacity.VCPUs * 1000
		memoryCapacityBytes += capacity.MemoryBytes
	}
	if cpuCapacityMillicores <= 0 || memoryCapacityBytes <= 0 {
		return cost.ClusterCapacity{}, fmt.Errorf("AWS cluster capacity must be greater than zero")
	}

	return cost.ClusterCapacity{
		CPUCapacityMillicores: cpuCapacityMillicores,
		MemoryCapacityBytes:   memoryCapacityBytes,
		NodeCount:             int64(len(nodes)),
		PricingSource:         cost.PricingSourceEstimated,
		NodeGroups:            buildAWSNodeGroups(nodes, capacities),
	}, nil
}

func (p *CapacityProvider) GetNodeGroups(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) ([]cost.NodeGroupCapacity, error) {
	if p == nil || p.nodeInventory == nil || p.capacityResolver == nil {
		return nil, fmt.Errorf("AWS capacity provider is not configured")
	}

	nodes, err := p.nodeInventory.ListRunningNodes(ctx, analysisContext)
	if err != nil {
		return nil, err
	}

	instanceTypes := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if node.InstanceType != "" {
			instanceTypes = append(instanceTypes, node.InstanceType)
		}
	}
	capacities, err := p.capacityResolver.ResolveMany(ctx, instanceTypes)
	if err != nil {
		return nil, err
	}
	return buildAWSNodeGroups(nodes, capacities), nil
}

func buildAWSNodeGroups(nodes []Node, capacities map[string]InstanceCapacity) []cost.NodeGroupCapacity {
	groups := make(map[string]cost.NodeGroupCapacity)
	for _, node := range nodes {
		if node.NodeGroup == "" {
			continue
		}
		capacity, ok := capacities[node.InstanceType]
		if !ok {
			continue
		}
		group := groups[node.NodeGroup]
		group.Name = node.NodeGroup
		group.CPUCapacityMillicores += capacity.VCPUs * 1000
		group.MemoryCapacityBytes += capacity.MemoryBytes
		group.NodeCount++
		if group.MachineType == "" {
			group.MachineType = node.InstanceType
		} else if group.MachineType != node.InstanceType {
			group.MachineType = ""
		}
		group.PricingSource = cost.PricingSourceEstimated
		groups[node.NodeGroup] = group
	}
	result := make([]cost.NodeGroupCapacity, 0, len(groups))
	for _, group := range groups {
		result = append(result, group)
	}
	return result
}

var _ capacity.Provider = (*CapacityProvider)(nil)
var _ capacity.NodeGroupProvider = (*CapacityProvider)(nil)
