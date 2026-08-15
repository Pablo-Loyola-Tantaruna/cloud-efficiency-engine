package analysis

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/domain"
)

type analysisBillingProviderMock struct {
	report billing.CostReport
}

func (
	p *analysisBillingProviderMock,
) GetCost(
	ctx context.Context,
	query billing.CostQuery,
) (
	billing.CostReport,
	error,
) {

	return p.report, nil
}

func TestEngineCalculateBillingSummary(
	t *testing.T,
) {

	engine :=
		&Engine{}

	report :=
		&AnalysisReport{
			Summary: AnalysisSummary{
				CurrentMonthlyCostUSD: 1000,
			},

			Billing: &billing.CostReport{
				Provider: domain.CloudProviderAWS,

				Start: time.Date(
					2026,
					8,
					1,
					0,
					0,
					0,
					0,
					time.UTC,
				),

				End: time.Date(
					2026,
					8,
					31,
					0,
					0,
					0,
					0,
					time.UTC,
				),

				Currency: "USD",

				TotalUSD: 1250,
			},
		}

	engine.calculateBillingSummary(
		report,
	)

	if report.Summary.ActualCloudCostUSD !=
		1250 {

		t.Fatalf(
			"expected actual cost 1250, got %f",
			report.Summary.ActualCloudCostUSD,
		)
	}

	if report.Summary.CostVarianceUSD !=
		250 {

		t.Fatalf(
			"expected variance 250, got %f",
			report.Summary.CostVarianceUSD,
		)
	}
}
