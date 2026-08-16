package gcp

import (
	"context"
	"testing"

	"cloud-efficiency-engine/internal/domain"

	"cloud.google.com/go/compute/apiv1/computepb"
	containerpb "cloud.google.com/go/container/apiv1/containerpb"
)

type gkeClusterMock struct {
	cluster *containerpb.Cluster
}

func (m *gkeClusterMock) GetCluster(context.Context, string, string, string) (*containerpb.Cluster, error) {
	return m.cluster, nil
}

type machineTypeMock struct {
	cores  int64
	memory int64
}

func (m *machineTypeMock) GetMachineType(context.Context, string, string, string) (*computepb.MachineType, error) {
	cores := int32(m.cores)
	memoryMB := int32(m.memory / (1024 * 1024))
	return &computepb.MachineType{
		GuestCpus: &cores,
		MemoryMb:  &memoryMB,
	}, nil
}

type managedGroupMock struct {
	counts map[string]int64
}

func (m *managedGroupMock) CountRunningInstances(_ context.Context, _ string, _ string, instanceGroupManager string) (int64, error) {
	return m.counts[instanceGroupManager], nil
}

func TestGKECapacitySource_ShouldSumRunningNodePools(t *testing.T) {
	source, err := NewGKECapacitySource(
		&gkeClusterMock{cluster: &containerpb.Cluster{
			NodePools: []*containerpb.NodePool{
				{
					Name: "pool-a",
					Config: &containerpb.NodeConfig{
						MachineType: "e2-standard-4",
					},
					InstanceGroupUrls: []string{
						"https://www.googleapis.com/compute/v1/projects/project-1/zones/us-central1-a/instanceGroupManagers/pool-a-group",
					},
				},
				{
					Name: "pool-b",
					Config: &containerpb.NodeConfig{
						MachineType: "e2-standard-8",
					},
					InstanceGroupUrls: []string{
						"https://www.googleapis.com/compute/v1/projects/project-1/zones/us-central1-b/instanceGroupManagers/pool-b-group",
					},
				},
			},
		}},
		&machineTypeMock{cores: 4, memory: 16 * 1024 * 1024 * 1024},
		&managedGroupMock{counts: map[string]int64{
			"pool-a-group": 2,
			"pool-b-group": 1,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}

	cpu, memory, err := source.GetCapacity(context.Background(), domain.AnalysisContext{
		Provider:    domain.CloudProviderGCP,
		AccountID:   "project-1",
		Region:      "us-central1",
		ClusterName: "cluster-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if cpu != 12000 {
		t.Fatalf("expected 12000 millicores, got %d", cpu)
	}
	if memory != 48*1024*1024*1024 {
		t.Fatalf("expected 48GiB, got %d", memory)
	}
}

func TestGKECapacitySource_ShouldRequireProjectAndCluster(t *testing.T) {
	source, err := NewGKECapacitySource(
		&gkeClusterMock{},
		&machineTypeMock{cores: 2, memory: 1},
		&managedGroupMock{counts: map[string]int64{}},
	)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = source.GetCapacity(context.Background(), domain.AnalysisContext{Provider: domain.CloudProviderGCP})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
