package aws

import (
	"cloud-efficiency-engine/internal/domain"
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/billing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

type costExplorerMock struct {
	called bool

	input *costexplorer.GetCostAndUsageInput

	output *costexplorer.GetCostAndUsageOutput

	err error
}

func (m *costExplorerMock) GetCostAndUsage(
	ctx context.Context,
	input *costexplorer.GetCostAndUsageInput,
	optFns ...func(*costexplorer.Options),
) (*costexplorer.GetCostAndUsageOutput, error) {

	m.called = true

	m.input = input

	return m.output, m.err
}

func TestCostExplorerBillingClient_ShouldReturnCost(
	t *testing.T,
) {

	mock :=
		&costExplorerMock{
			output: &costexplorer.GetCostAndUsageOutput{
				ResultsByTime: []types.ResultByTime{
					{
						Estimated: false,

						TimePeriod: &types.DateInterval{
							Start: aws.String(
								"2026-08-01",
							),

							End: aws.String(
								"2026-08-02",
							),
						},

						Groups: []types.Group{
							{
								Keys: []string{
									"Amazon Elastic Compute Cloud - Compute",
								},

								Metrics: map[string]types.MetricValue{
									"AmortizedCost": {
										Amount: aws.String(
											"12.50",
										),

										Unit: aws.String(
											"USD",
										),
									},
								},
							},
						},
					},
				},
			},
		}

	client :=
		NewCostExplorerBillingClient(
			mock,
		)

	report, err :=
		client.GetCost(
			context.Background(),
			domain.AnalysisContext{},
			billing.CostQuery{
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
					2,
					0,
					0,
					0,
					0,
					time.UTC,
				),
			},
		)

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if !mock.called {

		t.Fatal(
			"expected Cost Explorer to be called",
		)
	}

	if report.TotalUSD != 12.50 {

		t.Fatalf(
			"expected total 12.50, got %f",
			report.TotalUSD,
		)
	}

	if len(report.Periods) != 1 {

		t.Fatalf(
			"expected 1 period, got %d",
			len(report.Periods),
		)
	}

	if report.Periods[0].Service !=
		"Amazon Elastic Compute Cloud - Compute" {

		t.Fatalf(
			"unexpected service: %s",
			report.Periods[0].Service,
		)
	}
}

func TestCostExplorerBillingClient_ShouldValidateQuery(
	t *testing.T,
) {

	client :=
		NewCostExplorerBillingClient(
			&costExplorerMock{},
		)

	_, err :=
		client.GetCost(
			context.Background(),
			domain.AnalysisContext{},
			billing.CostQuery{},
		)

	if err == nil {

		t.Fatal(
			"expected validation error",
		)
	}
}
