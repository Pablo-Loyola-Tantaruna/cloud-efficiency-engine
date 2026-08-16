package gcp

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/providers"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/container/apiv1/containerpb"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/iterator"
)

type GKEClusterReader interface {
	GetCluster(
		ctx context.Context,
		projectID string,
		location string,
		clusterName string,
	) (*containerpb.Cluster, error)
}

type ComputeMachineTypeReader interface {
	GetMachineType(
		ctx context.Context,
		projectID string,
		zone string,
		machineType string,
	) (*computepb.MachineType, error)
}

type ManagedInstanceGroupReader interface {
	CountRunningInstances(
		ctx context.Context,
		projectID string,
		zone string,
		instanceGroupManager string,
	) (int64, error)
}

type GKECapacitySource struct {
	clusters      GKEClusterReader
	machineTypes  ComputeMachineTypeReader
	managedGroups ManagedInstanceGroupReader
}

func NewGKECapacitySource(
	clusters GKEClusterReader,
	machineTypes ComputeMachineTypeReader,
	managedGroups ManagedInstanceGroupReader,
) (*GKECapacitySource, error) {
	if clusters == nil {
		return nil, fmt.Errorf("GCP GKE cluster reader must not be nil")
	}
	if machineTypes == nil {
		return nil, fmt.Errorf("GCP machine type reader must not be nil")
	}
	if managedGroups == nil {
		return nil, fmt.Errorf("GCP managed instance group reader must not be nil")
	}
	return &GKECapacitySource{
		clusters:      clusters,
		machineTypes:  machineTypes,
		managedGroups: managedGroups,
	}, nil
}

func (s *GKECapacitySource) loadClusterCapacity(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (int64, int64, int64, error) {
	if s == nil || s.clusters == nil || s.machineTypes == nil || s.managedGroups == nil {
		return 0, 0, 0, fmt.Errorf("GCP GKE capacity source is not configured")
	}

	projectID := strings.TrimSpace(analysisContext.AccountID)
	location := strings.TrimSpace(analysisContext.Region)
	clusterName := strings.TrimSpace(analysisContext.ClusterName)
	if projectID == "" {
		return 0, 0, 0, fmt.Errorf("GCP project ID must not be empty in analysis context accountId")
	}
	if location == "" {
		return 0, 0, 0, fmt.Errorf("GCP GKE location must not be empty in analysis context region")
	}
	if clusterName == "" {
		return 0, 0, 0, fmt.Errorf("GCP GKE cluster name must not be empty")
	}

	cluster, err := s.clusters.GetCluster(ctx, projectID, location, clusterName)
	if err != nil {
		return 0, 0, 0, err
	}
	if cluster == nil {
		return 0, 0, 0, fmt.Errorf("GCP GKE cluster %q was not found", clusterName)
	}

	var cpuMillicores int64
	var memoryBytes int64
	var nodeCount int64

	for _, pool := range cluster.GetNodePools() {
		if pool == nil {
			continue
		}

		config := pool.GetConfig()
		if config == nil {
			return 0, 0, 0, fmt.Errorf("GCP GKE node pool %q has no config", pool.GetName())
		}

		machineType := strings.TrimSpace(config.GetMachineType())
		if machineType == "" {
			return 0, 0, 0, fmt.Errorf("GCP GKE node pool %q has no machine type", pool.GetName())
		}
		machineType = lastPathSegment(machineType)

		instanceGroupURLs := pool.GetInstanceGroupUrls()
		if len(instanceGroupURLs) == 0 {
			return 0, 0, 0, fmt.Errorf("GCP GKE node pool %q has no managed instance groups", pool.GetName())
		}

		var poolNodes int64
		zone := ""
		for _, instanceGroupURL := range instanceGroupURLs {
			groupZone, manager, err := parseZonalInstanceGroupURL(instanceGroupURL)
			if err != nil {
				return 0, 0, 0, fmt.Errorf("GCP GKE node pool %q: %w", pool.GetName(), err)
			}
			if zone == "" {
				zone = groupZone
			}
			count, err := s.managedGroups.CountRunningInstances(ctx, projectID, groupZone, manager)
			if err != nil {
				return 0, 0, 0, err
			}
			poolNodes += count
		}

		if poolNodes <= 0 {
			continue
		}

		machine, err := s.machineTypes.GetMachineType(ctx, projectID, zone, machineType)
		if err != nil {
			return 0, 0, 0, err
		}
		if machine == nil || machine.GetGuestCpus() <= 0 || machine.GetMemoryMb() <= 0 {
			return 0, 0, 0, fmt.Errorf("GCP machine type %q has incomplete capacity metadata", machineType)
		}

		cpuMillicores += poolNodes * int64(machine.GetGuestCpus()) * 1000
		memoryBytes += poolNodes * int64(machine.GetMemoryMb()) * 1024 * 1024
		nodeCount += poolNodes
	}

	if cpuMillicores == 0 || memoryBytes == 0 || nodeCount == 0 {
		return 0, 0, 0, fmt.Errorf("GCP GKE cluster %q has no active node capacity", clusterName)
	}
	return cpuMillicores, memoryBytes, nodeCount, nil
}

func (s *GKECapacitySource) GetCapacity(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (int64, int64, error) {
	cpuMillicores, memoryBytes, _, err := s.loadClusterCapacity(ctx, analysisContext)
	return cpuMillicores, memoryBytes, err
}

func (s *GKECapacitySource) GetNodeCount(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (int64, error) {
	_, _, nodeCount, err := s.loadClusterCapacity(ctx, analysisContext)
	return nodeCount, err
}

func lastPathSegment(value string) string {
	parsed, err := url.Parse(value)
	if err == nil && parsed.Path != "" {
		segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		if len(segments) > 0 {
			return segments[len(segments)-1]
		}
	}
	value = strings.TrimRight(value, "/")
	if index := strings.LastIndex(value, "/"); index >= 0 {
		return value[index+1:]
	}
	return value
}

func parseZonalInstanceGroupURL(value string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", "", fmt.Errorf("parse instance group URL: %w", err)
	}
	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	zoneIndex := indexOf(segments, "zones")
	managerIndex := indexOf(segments, "instanceGroupManagers")
	if zoneIndex < 0 || managerIndex < 0 || zoneIndex+1 >= len(segments) || managerIndex+1 >= len(segments) {
		return "", "", fmt.Errorf("instance group URL must contain a zone and instance group manager")
	}
	return segments[zoneIndex+1], segments[managerIndex+1], nil
}

func indexOf(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}

type ClusterManagerAPI interface {
	GetCluster(
		ctx context.Context,
		request *containerpb.GetClusterRequest,
		opts ...gax.CallOption,
	) (*containerpb.Cluster, error)
}

type GKEClusterManagerClientReader struct {
	client ClusterManagerAPI
}

func NewGKEClusterManagerClientReader(client ClusterManagerAPI) *GKEClusterManagerClientReader {
	return &GKEClusterManagerClientReader{client: client}
}

func (r *GKEClusterManagerClientReader) GetCluster(
	ctx context.Context,
	projectID string,
	location string,
	clusterName string,
) (*containerpb.Cluster, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("GCP GKE cluster client is not configured")
	}
	return r.client.GetCluster(ctx, &containerpb.GetClusterRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/clusters/%s", projectID, location, clusterName),
	})
}

type GCPMachineTypeReader struct {
	client *compute.MachineTypesClient
}

func NewGCPMachineTypeReader(client *compute.MachineTypesClient) *GCPMachineTypeReader {
	return &GCPMachineTypeReader{client: client}
}

func (r *GCPMachineTypeReader) GetMachineType(
	ctx context.Context,
	projectID string,
	zone string,
	machineType string,
) (*computepb.MachineType, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("GCP machine types client is not configured")
	}
	return r.client.Get(ctx, &computepb.GetMachineTypeRequest{
		Project:     projectID,
		Zone:        zone,
		MachineType: machineType,
	})
}

type GCPManagedInstanceGroupReader struct {
	client *compute.InstanceGroupManagersClient
}

func NewGCPManagedInstanceGroupReader(client *compute.InstanceGroupManagersClient) *GCPManagedInstanceGroupReader {
	return &GCPManagedInstanceGroupReader{client: client}
}

func (r *GCPManagedInstanceGroupReader) CountRunningInstances(
	ctx context.Context,
	projectID string,
	zone string,
	instanceGroupManager string,
) (int64, error) {
	if r == nil || r.client == nil {
		return 0, fmt.Errorf("GCP instance group managers client is not configured")
	}

	it := r.client.ListManagedInstances(ctx, &computepb.ListManagedInstancesInstanceGroupManagersRequest{
		Project:              projectID,
		Zone:                 zone,
		InstanceGroupManager: instanceGroupManager,
	})

	var count int64
	for {
		instance, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("list GCP managed instances: %w", err)
		}
		if instance != nil && strings.EqualFold(instance.GetCurrentAction(), "NONE") {
			count++
		}
	}
	return count, nil
}

var _ GKEClusterReader = (*GKEClusterManagerClientReader)(nil)
var _ ComputeMachineTypeReader = (*GCPMachineTypeReader)(nil)
var _ ManagedInstanceGroupReader = (*GCPManagedInstanceGroupReader)(nil)
var _ providers.CapacitySource = (*GKECapacitySource)(nil)
var _ providers.NodeCountSource = (*GKECapacitySource)(nil)
