package azure

import (
	"context"
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

type aksClusterReaderMock struct {
	cluster AKSCluster
	err     error
}

func (m *aksClusterReaderMock) FindCluster(
	ctx context.Context,
	clusterName string,
) (AKSCluster, error) {
	return m.cluster, m.err
}

type azureVMSizeReaderMock struct {
	sizes map[string]struct {
		cores  int64
		memory int64
	}
	err error
}

func (m *azureVMSizeReaderMock) GetSize(
	ctx context.Context,
	location string,
	vmSize string,
) (int64, int64, error) {
	if m.err != nil {
		return 0, 0, m.err
	}
	value, ok := m.sizes[vmSize]
	if !ok {
		return 0, 0, nil
	}
	return value.cores, value.memory, nil
}

func TestAKSCapacitySource_ShouldAggregateNodePools(t *testing.T) {
	memoryPerNode := int64(16 * 1024 * 1024 * 1024)

	source, err := NewAKSCapacitySource(
		&aksClusterReaderMock{
			cluster: AKSCluster{
				Name:     "prod",
				Location: "eastus",
				NodePools: []AKSNodePool{
					{
						Name:      "system",
						VMSize:    "Standard_D4s_v5",
						NodeCount: 2,
					},
				},
			},
		},
		&azureVMSizeReaderMock{
			sizes: map[string]struct {
				cores  int64
				memory int64
			}{
				"Standard_D4s_v5": {
					cores:  4,
					memory: memoryPerNode,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cpu, memory, err := source.GetCapacity(
		context.Background(),
		domain.AnalysisContext{
			Provider:    domain.CloudProviderAzure,
			ClusterName: "prod",
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cpu != 8000 {
		t.Fatalf("expected 8000 millicores, got %d", cpu)
	}
	if memory != 2*memoryPerNode {
		t.Fatalf("expected %d bytes, got %d", 2*memoryPerNode, memory)
	}
}

func TestAKSCapacitySource_ShouldRequireClusterName(t *testing.T) {
	source, err := NewAKSCapacitySource(
		&aksClusterReaderMock{},
		&azureVMSizeReaderMock{},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, _, err = source.GetCapacity(
		context.Background(),
		domain.AnalysisContext{
			Provider: domain.CloudProviderAzure,
		},
	)
	if err == nil {
		t.Fatal("expected cluster name validation error")
	}
}

func TestAKSCapacitySource_ShouldFailForMissingVMSize(t *testing.T) {
	source, err := NewAKSCapacitySource(
		&aksClusterReaderMock{
			cluster: AKSCluster{
				Name:     "prod",
				Location: "eastus",
				NodePools: []AKSNodePool{
					{
						Name:      "system",
						VMSize:    "",
						NodeCount: 2,
					},
				},
			},
		},
		&azureVMSizeReaderMock{},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, _, err = source.GetCapacity(
		context.Background(),
		domain.AnalysisContext{
			Provider:    domain.CloudProviderAzure,
			ClusterName: "prod",
		},
	)
	if err == nil {
		t.Fatal("expected VM size validation error")
	}
}
