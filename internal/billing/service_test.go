package billing

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type serviceProviderMock struct {
	report CostReport
}

func (m *serviceProviderMock) GetCost(
	ctx context.Context,
	query CostQuery,
) (CostReport, error) {

	return m.report, nil
}

func TestServiceGetCost_ShouldDelegate(
	t *testing.T,
) {

	expected :=
		CostReport{
			Provider: domain.CloudProviderAWS,
			Currency: "USD",
			TotalUSD: 123.45,
		}

	provider :=
		&serviceProviderMock{
			report: expected,
		}

	service :=
		NewService(
			provider,
		)

	report, err :=
		service.GetCost(
			context.Background(),
			CostQuery{
				Start: time.Now().Add(
					-24 * time.Hour,
				),
				End: time.Now(),
			},
		)

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if report.TotalUSD != expected.TotalUSD {

		t.Fatalf(
			"expected total %.2f, got %.2f",
			expected.TotalUSD,
			report.TotalUSD,
		)
	}
}
