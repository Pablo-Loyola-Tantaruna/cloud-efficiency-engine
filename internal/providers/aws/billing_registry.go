package aws

import (
	"fmt"

	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/domain"
	providerregistry "cloud-efficiency-engine/internal/providers"
)

func RegisterBillingProvider(
	registry *providerregistry.Registry,
	billingClient BillingClient,
) error {

	if registry == nil {

		return fmt.Errorf(
			"provider registry must not be nil",
		)
	}

	if billingClient == nil {

		return fmt.Errorf(
			"AWS billing client must not be nil",
		)
	}

	registry.RegisterBillingProvider(
		domain.CloudProviderAWS,
		func(
			analysisContext domain.AnalysisContext,
		) (
			billing.Provider,
			error,
		) {

			return NewBillingSource(
				billingClient,
			), nil
		},
	)

	return nil
}
