package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v8"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/costmanagement/armcostmanagement/v3"
)

type Clients struct {
	Compute *armcompute.ClientFactory

	CostManagement *armcostmanagement.ClientFactory
}

func NewClients(
	subscriptionID string,
) (*Clients, error) {

	if subscriptionID == "" {

		return nil, fmt.Errorf(
			"Azure subscription ID must not be empty",
		)
	}

	credential, err :=
		azidentity.NewDefaultAzureCredential(
			nil,
		)

	if err != nil {

		return nil, fmt.Errorf(
			"create Azure credential: %w",
			err,
		)
	}

	computeFactory, err :=
		armcompute.NewClientFactory(
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

	costManagementFactory, err :=
		armcostmanagement.NewClientFactory(
			credential,
			nil,
		)

	if err != nil {

		return nil, fmt.Errorf(
			"create Azure cost management client: %w",
			err,
		)
	}

	return &Clients{
		Compute: computeFactory,

		CostManagement: costManagementFactory,
	}, nil
}
