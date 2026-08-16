package azure

import (
	"context"
	"fmt"
	"strings"

	"cloud-efficiency-engine/internal/domain"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v8"
)

type AzureVMInventory struct {
	virtualMachines     *armcompute.VirtualMachinesClient
	virtualMachineSizes *armcompute.VirtualMachineSizesClient
}

func NewAzureVMInventory(
	virtualMachines *armcompute.VirtualMachinesClient,
	virtualMachineSizes *armcompute.VirtualMachineSizesClient,
) *AzureVMInventory {

	return &AzureVMInventory{
		virtualMachines:     virtualMachines,
		virtualMachineSizes: virtualMachineSizes,
	}
}

func (i *AzureVMInventory) ListResources(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) ([]AzureResource, error) {

	if i == nil || i.virtualMachines == nil {
		return nil, fmt.Errorf("Azure virtual machine client is not configured")
	}

	if i.virtualMachineSizes == nil {
		return nil, fmt.Errorf("Azure virtual machine sizes client is not configured")
	}

	pager := i.virtualMachines.NewListAllPager(nil)

	resources := make([]AzureResource, 0)
	sizeCache := make(map[string]map[string]vmCapacity)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list Azure virtual machines: %w", err)
		}

		for _, vm := range page.Value {
			if vm == nil || vm.Name == nil || vm.ID == nil {
				continue
			}

			if !isAzureVMInContext(*vm.ID, analysisContext) {
				continue
			}

			if vm.Location == nil || *vm.Location == "" {
				return nil, fmt.Errorf(
					"Azure virtual machine %q has no location",
					*vm.Name,
				)
			}

			if vm.Properties == nil ||
				vm.Properties.HardwareProfile == nil ||
				vm.Properties.HardwareProfile.VMSize == nil ||
				*vm.Properties.HardwareProfile.VMSize == "" {
				return nil, fmt.Errorf(
					"Azure virtual machine %q has no VM size",
					*vm.Name,
				)
			}

			vmSizeName := string(*vm.Properties.HardwareProfile.VMSize)
			location := *vm.Location

			if _, ok := sizeCache[location]; !ok {
				sizes, err := i.listSizes(ctx, location)
				if err != nil {
					return nil, err
				}
				sizeCache[location] = sizes
			}

			capacity, ok := sizeCache[location][vmSizeName]
			if !ok {
				return nil, fmt.Errorf(
					"Azure VM size %q was not found in location %q",
					vmSizeName,
					location,
				)
			}

			resources = append(
				resources,
				AzureResource{
					ID:                   *vm.ID,
					Namespace:            resourceGroupFromAzureID(*vm.ID),
					Name:                 *vm.Name,
					Type:                 domain.WorkloadUnknown,
					CPUCores:             capacity.CPUCores,
					CPURequestMillicores: capacity.CPUCores * 1000,
					MemoryRequestBytes:   capacity.MemoryBytes,
				},
			)
		}
	}

	return resources, nil
}

type vmCapacity struct {
	CPUCores    int64
	MemoryBytes int64
}

func (i *AzureVMInventory) listSizes(
	ctx context.Context,
	location string,
) (map[string]vmCapacity, error) {

	pager := i.virtualMachineSizes.NewListPager(
		location,
		nil,
	)

	result := make(map[string]vmCapacity)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf(
				"list Azure VM sizes for location %q: %w",
				location,
				err,
			)
		}

		for _, size := range page.Value {
			if size == nil ||
				size.Name == nil ||
				size.NumberOfCores == nil ||
				size.MemoryInMB == nil {
				continue
			}

			if *size.NumberOfCores <= 0 || *size.MemoryInMB <= 0 {
				continue
			}

			result[*size.Name] = vmCapacity{
				CPUCores:    int64(*size.NumberOfCores),
				MemoryBytes: int64(*size.MemoryInMB) * 1024 * 1024,
			}
		}
	}

	return result, nil
}

func isAzureVMInContext(
	resourceID string,
	analysisContext domain.AnalysisContext,
) bool {

	if analysisContext.AccountID != "" &&
		!strings.Contains(resourceID, "/subscriptions/"+analysisContext.AccountID+"/") {
		return false
	}

	return true
}

func resourceGroupFromAzureID(resourceID string) string {
	parts := strings.Split(resourceID, "/")

	for index := 0; index+1 < len(parts); index++ {
		if strings.EqualFold(parts[index], "resourceGroups") {
			return parts[index+1]
		}
	}

	return ""
}

var _ ResourceInventory = (*AzureVMInventory)(nil)
