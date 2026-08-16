package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v8"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v9"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/costmanagement/armcostmanagement/v3"
)

type Clients struct {
	Compute *armcompute.ClientFactory

	VirtualMachines *armcompute.VirtualMachinesClient

	VirtualMachineSizes *armcompute.VirtualMachineSizesClient

	ManagedClusters *armcontainerservice.ManagedClustersClient

	CostManagement *armcostmanagement.ClientFactory

	Query *armcostmanagement.QueryClient

	Monitor *azquery.MetricsClient
}

func NewClients(
	subscriptionID string,
) (*Clients, error) {
	if subscriptionID == "" {
		return nil, fmt.Errorf(
			"Azure subscription ID must not be empty",
		)
	}

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf(
			"create Azure credential: %w",
			err,
		)
	}

	computeFactory, err := armcompute.NewClientFactory(
		subscriptionID,
		credential,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create Azure compute client: %w",
			err,
		)
	}

	containerServiceFactory, err := armcontainerservice.NewClientFactory(
		subscriptionID,
		credential,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create Azure Container Service client: %w",
			err,
		)
	}

	costManagementFactory, err := armcostmanagement.NewClientFactory(
		credential,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create Azure cost management client: %w",
			err,
		)
	}

	monitorClient, err := azquery.NewMetricsClient(
		credential,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create Azure Monitor metrics client: %w",
			err,
		)
	}

	return &Clients{
		Compute:             computeFactory,
		VirtualMachines:     computeFactory.NewVirtualMachinesClient(),
		VirtualMachineSizes: computeFactory.NewVirtualMachineSizesClient(),
		ManagedClusters:     containerServiceFactory.NewManagedClustersClient(),
		CostManagement:      costManagementFactory,
		Query:               costManagementFactory.NewQueryClient(),
		Monitor:             monitorClient,
	}, nil
}
