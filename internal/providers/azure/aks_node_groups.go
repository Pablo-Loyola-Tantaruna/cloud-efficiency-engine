package azure

import (
	"context"
	"fmt"
	"strings"

	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

func (s *AKSCapacitySource) GetNodeGroups(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) ([]cost.NodeGroupCapacity, error) {
	if s == nil || s.clusters == nil || s.sizes == nil {
		return nil, fmt.Errorf("Azure AKS capacity source is not configured")
	}

	clusterName := strings.TrimSpace(analysisContext.ClusterName)
	if clusterName == "" {
		return nil, fmt.Errorf("Azure AKS cluster name must not be empty")
	}

	cluster, err := s.clusters.FindCluster(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	groups := make([]cost.NodeGroupCapacity, 0, len(cluster.NodePools))
	for _, pool := range cluster.NodePools {
		if pool.NodeCount <= 0 || strings.TrimSpace(pool.Name) == "" || strings.TrimSpace(pool.VMSize) == "" {
			continue
		}

		cores, memoryBytes, err := s.sizes.GetSize(ctx, cluster.Location, pool.VMSize)
		if err != nil {
			return nil, err
		}

		groups = append(groups, cost.NodeGroupCapacity{
			Name:                  pool.Name,
			MachineType:           pool.VMSize,
			CPUCapacityMillicores: cores * pool.NodeCount * 1000,
			MemoryCapacityBytes:   memoryBytes * pool.NodeCount,
			NodeCount:             pool.NodeCount,
			PricingSource:         cost.PricingSourceEstimated,
		})
	}

	return groups, nil
}

var _ interface {
	GetNodeGroups(context.Context, domain.AnalysisContext) ([]cost.NodeGroupCapacity, error)
} = (*AKSCapacitySource)(nil)
