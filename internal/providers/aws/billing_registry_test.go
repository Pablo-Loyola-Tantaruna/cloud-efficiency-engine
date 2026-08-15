package aws

import (
	"context"
	"testing"

	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/domain"
	providerregistry "cloud-efficiency-engine/internal/providers"
)

type billingRegistryMock struct {
}

func (m *billingRegistryMock) GetCost(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	query billing.CostQuery,
) (billing.CostReport, error) {

	return billing.CostReport{
		Provider: domain.CloudProviderAWS,
		Currency: "USD",
		TotalUSD: 100,
	}, nil
}

func TestRegisterBillingProvider_ShouldRegisterAWS(
	t *testing.T,
) {

	registry :=
		providerregistry.NewRegistry()

	client :=
		&billingRegistryMock{}

	err :=
		RegisterBillingProvider(
			registry,
			client,
		)

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	bundle, err :=
		registry.Resolve(
			context.Background(),
			domain.AnalysisContext{
				Provider: domain.CloudProviderAWS,
			},
		)

	if err == nil {

		t.Fatal(
			"expected resolve to fail because metrics and pricing are not registered",
		)
	}

	if bundle != nil {

		t.Fatal(
			"expected nil bundle",
		)
	}
}

func TestRegisterBillingProvider_ShouldRegisterAWSBillingFactory(
	t *testing.T,
) {

	registry :=
		providerregistry.NewRegistry()

	client :=
		&billingRegistryMock{}

	if err :=
		RegisterBillingProvider(
			registry,
			client,
		); err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}
}
