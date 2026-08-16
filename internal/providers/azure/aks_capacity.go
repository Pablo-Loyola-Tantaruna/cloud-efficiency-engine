package azure

import (
	"context"
	"fmt"
	"strings"

	"cloud-efficiency-engine/internal/domain"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v8"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v9"
)

type AKSNodePool struct {
	Name      string
	VMSize    string
	NodeCount int64
}

type AKSCluster struct {
	Name      string
	Location  string
	NodePools []AKSNodePool
}

type AKSClusterReader interface {
	FindCluster(
		ctx context.Context,
		clusterName string,
	) (AKSCluster, error)
}

type AzureVMSizeReader interface {
	GetSize(
		ctx context.Context,
		location string,
		vmSize string,
	) (int64, int64, error)
}

type AKSCapacitySource struct {
	clusters AKSClusterReader
	sizes    AzureVMSizeReader
}

func NewAKSCapacitySource(
	clusters AKSClusterReader,
	sizes AzureVMSizeReader,
) (*AKSCapacitySource, error) {
	if clusters == nil {
		return nil, fmt.Errorf(
			"Azure AKS cluster reader must not be nil",
		)
	}
	if sizes == nil {
		return nil, fmt.Errorf(
			"Azure VM size reader must not be nil",
		)
	}

	return &AKSCapacitySource{
		clusters: clusters,
		sizes:    sizes,
	}, nil
}

func (s *AKSCapacitySource) GetCapacity(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (int64, int64, error) {
	if s == nil || s.clusters == nil || s.sizes == nil {
		return 0, 0, fmt.Errorf(
			"Azure AKS capacity source is not configured",
		)
	}

	clusterName := strings.TrimSpace(
		analysisContext.ClusterName,
	)
	if clusterName == "" {
		return 0, 0, fmt.Errorf(
			"Azure AKS cluster name must not be empty",
		)
	}

	cluster, err := s.clusters.FindCluster(
		ctx,
		clusterName,
	)
	if err != nil {
		return 0, 0, err
	}

	if len(cluster.NodePools) == 0 {
		return 0, 0, fmt.Errorf(
			"Azure AKS cluster %q has no node pools",
			clusterName,
		)
	}

	var cpuMillicores int64
	var memoryBytes int64

	for _, pool := range cluster.NodePools {
		if pool.NodeCount <= 0 {
			continue
		}
		if strings.TrimSpace(pool.VMSize) == "" {
			return 0, 0, fmt.Errorf(
				"Azure AKS node pool %q has no VM size",
				pool.Name,
			)
		}

		cores, memoryBytesPerNode, err := s.sizes.GetSize(
			ctx,
			cluster.Location,
			pool.VMSize,
		)
		if err != nil {
			return 0, 0, err
		}

		cpuMillicores += pool.NodeCount * cores * 1000
		memoryBytes += pool.NodeCount * memoryBytesPerNode
	}

	if cpuMillicores == 0 && memoryBytes == 0 {
		return 0, 0, fmt.Errorf(
			"Azure AKS cluster %q has no active node capacity",
			clusterName,
		)
	}

	return cpuMillicores, memoryBytes, nil
}

type ARMManagedClusterReader struct {
	client *armcontainerservice.ManagedClustersClient
}

func NewARMManagedClusterReader(
	client *armcontainerservice.ManagedClustersClient,
) *ARMManagedClusterReader {
	return &ARMManagedClusterReader{
		client: client,
	}
}

func (r *ARMManagedClusterReader) FindCluster(
	ctx context.Context,
	clusterName string,
) (AKSCluster, error) {
	if r == nil || r.client == nil {
		return AKSCluster{}, fmt.Errorf(
			"Azure managed clusters client is not configured",
		)
	}

	clusterName = strings.TrimSpace(clusterName)
	pager := r.client.NewListPager(nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return AKSCluster{}, fmt.Errorf(
				"list Azure AKS clusters: %w",
				err,
			)
		}

		for _, cluster := range page.Value {
			if cluster == nil ||
				cluster.Name == nil ||
				!strings.EqualFold(*cluster.Name, clusterName) {
				continue
			}

			location := ""
			if cluster.Location != nil {
				location = *cluster.Location
			}

			result := AKSCluster{
				Name:     *cluster.Name,
				Location: location,
			}

			if cluster.Properties == nil {
				return AKSCluster{}, fmt.Errorf(
					"Azure AKS cluster %q has no properties",
					clusterName,
				)
			}

			for _, pool := range cluster.Properties.AgentPoolProfiles {
				if pool == nil {
					continue
				}

				poolName := ""
				if pool.Name != nil {
					poolName = *pool.Name
				}

				vmSize := ""
				if pool.VMSize != nil {
					vmSize = *pool.VMSize
				}

				count := int64(0)
				if pool.Count != nil {
					count = int64(*pool.Count)
				}

				if vmSize != "" && count > 0 {
					result.NodePools = append(
						result.NodePools,
						AKSNodePool{
							Name:      poolName,
							VMSize:    vmSize,
							NodeCount: count,
						},
					)
				}
			}

			return result, nil
		}
	}

	return AKSCluster{}, fmt.Errorf(
		"Azure AKS cluster %q was not found",
		clusterName,
	)
}

type ARMVMSizeReader struct {
	client *armcompute.VirtualMachineSizesClient
}

func NewARMVMSizeReader(
	client *armcompute.VirtualMachineSizesClient,
) *ARMVMSizeReader {
	return &ARMVMSizeReader{
		client: client,
	}
}

func (r *ARMVMSizeReader) GetSize(
	ctx context.Context,
	location string,
	vmSize string,
) (int64, int64, error) {
	if r == nil || r.client == nil {
		return 0, 0, fmt.Errorf(
			"Azure VM sizes client is not configured",
		)
	}

	location = strings.TrimSpace(location)
	vmSize = strings.TrimSpace(vmSize)
	if location == "" || vmSize == "" {
		return 0, 0, fmt.Errorf(
			"Azure VM size location and name must not be empty",
		)
	}

	pager := r.client.NewListPager(location, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return 0, 0, fmt.Errorf(
				"list Azure VM sizes: %w",
				err,
			)
		}

		for _, size := range page.Value {
			if size == nil ||
				size.Name == nil ||
				!strings.EqualFold(*size.Name, vmSize) {
				continue
			}

			if size.NumberOfCores == nil || size.MemoryInMB == nil {
				return 0, 0, fmt.Errorf(
					"Azure VM size %q has incomplete capacity metadata",
					vmSize,
				)
			}

			return int64(*size.NumberOfCores),
				int64(*size.MemoryInMB) * 1024 * 1024,
				nil
		}
	}

	return 0, 0, fmt.Errorf(
		"Azure VM size %q was not found in location %q",
		vmSize,
		location,
	)
}

var _ CapacitySource = (*AKSCapacitySource)(nil)
