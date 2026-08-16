package azure

import "fmt"

func NewAKSCapacityProviderFromClients(
	clients *Clients,
) (*CapacityProvider, error) {
	if clients == nil {
		return nil, fmt.Errorf(
			"Azure clients must not be nil",
		)
	}

	source, err := NewAKSCapacitySource(
		NewARMManagedClusterReader(clients.ManagedClusters),
		NewARMVMSizeReader(clients.VirtualMachineSizes),
	)
	if err != nil {
		return nil, err
	}

	return NewCapacityProvider(source), nil
}
