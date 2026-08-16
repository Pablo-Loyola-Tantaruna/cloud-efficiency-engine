package gcp

import (
	"context"
	"fmt"
	"strings"

	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
)

func (s *GKECapacitySource) GetNodeGroups(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) ([]cost.NodeGroupCapacity, error) {
	if s == nil || s.clusters == nil || s.machineTypes == nil || s.managedGroups == nil {
		return nil, fmt.Errorf("GCP GKE capacity source is not configured")
	}

	projectID := strings.TrimSpace(analysisContext.AccountID)
	location := strings.TrimSpace(analysisContext.Region)
	clusterName := strings.TrimSpace(analysisContext.ClusterName)
	if projectID == "" || location == "" || clusterName == "" {
		return nil, fmt.Errorf("GCP project, location and cluster name are required")
	}

	cluster, err := s.clusters.GetCluster(ctx, projectID, location, clusterName)
	if err != nil {
		return nil, err
	}
	if cluster == nil {
		return nil, fmt.Errorf("GCP GKE cluster %q was not found", clusterName)
	}

	groups := make([]cost.NodeGroupCapacity, 0, len(cluster.GetNodePools()))
	for _, pool := range cluster.GetNodePools() {
		if pool == nil || strings.TrimSpace(pool.GetName()) == "" {
			continue
		}

		config := pool.GetConfig()
		if config == nil {
			continue
		}
		machineType := strings.TrimSpace(config.GetMachineType())
		if machineType == "" {
			continue
		}
		machineType = lastPathSegment(machineType)

		var nodeCount int64
		zone := ""
		for _, instanceGroupURL := range pool.GetInstanceGroupUrls() {
			groupZone, manager, parseErr := parseZonalInstanceGroupURL(instanceGroupURL)
			if parseErr != nil {
				return nil, fmt.Errorf("GCP GKE node pool %q: %w", pool.GetName(), parseErr)
			}
			if zone == "" {
				zone = groupZone
			}
			count, countErr := s.managedGroups.CountRunningInstances(ctx, projectID, groupZone, manager)
			if countErr != nil {
				return nil, countErr
			}
			nodeCount += count
		}
		if nodeCount <= 0 || zone == "" {
			continue
		}

		machine, err := s.machineTypes.GetMachineType(ctx, projectID, zone, machineType)
		if err != nil {
			return nil, err
		}
		if machine == nil || machine.GetGuestCpus() <= 0 || machine.GetMemoryMb() <= 0 {
			return nil, fmt.Errorf("GCP machine type %q has incomplete capacity metadata", machineType)
		}

		groups = append(groups, cost.NodeGroupCapacity{
			Name:                  pool.GetName(),
			MachineType:           machineType,
			CPUCapacityMillicores: int64(machine.GetGuestCpus()) * nodeCount * 1000,
			MemoryCapacityBytes:   int64(machine.GetMemoryMb()) * nodeCount * 1024 * 1024,
			NodeCount:             nodeCount,
			PricingSource:         cost.PricingSourceEstimated,
		})
	}

	return groups, nil
}

var _ interface {
	GetNodeGroups(context.Context, domain.AnalysisContext) ([]cost.NodeGroupCapacity, error)
} = (*GKECapacitySource)(nil)
