package aws

import (
	"fmt"

	"cloud-efficiency-engine/internal/analysis/capacity"
	"cloud-efficiency-engine/internal/domain"
	providerregistry "cloud-efficiency-engine/internal/providers"
)

func RegisterCapacityProvider(
	registry *providerregistry.Registry,
	ec2Client EC2API,
) error {

	if registry == nil {
		return fmt.Errorf(
			"provider registry must not be nil",
		)
	}

	if ec2Client == nil {
		return fmt.Errorf(
			"AWS EC2 client must not be nil",
		)
	}

	registry.RegisterCapacityProvider(
		domain.CloudProviderAWS,
		func(
			analysisContext domain.AnalysisContext,
		) (
			capacity.Provider,
			error,
		) {

			return NewCapacityProvider(
				ec2Client,
			), nil
		},
	)

	return nil
}
