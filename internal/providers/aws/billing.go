package aws

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

const (
	awsBillingCurrency = "USD"

	awsCostMetric = "AmortizedCost"

	awsServiceDimension = types.DimensionService

	awsServiceGroup = types.GroupDefinitionTypeDimension

	awsCostGranularity = types.GranularityDaily
)

type BillingSource struct {
	client BillingClient
}

func NewBillingSource(
	client BillingClient,
) *BillingSource {

	return &BillingSource{
		client: client,
	}
}

func (s *BillingSource) GetCost(
	ctx context.Context,
	query billing.CostQuery,
) (billing.CostReport, error) {

	return s.GetCostWithContext(
		ctx,
		domain.AnalysisContext{},
		query,
	)
}

func (s *BillingSource) GetCostWithContext(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	query billing.CostQuery,
) (billing.CostReport, error) {

	if s.client == nil {

		return billing.CostReport{},
			fmt.Errorf(
				"AWS billing client is not configured",
			)
	}

	if query.Start.IsZero() {

		return billing.CostReport{},
			fmt.Errorf(
				"billing start time must not be zero",
			)
	}

	if query.End.IsZero() {

		return billing.CostReport{},
			fmt.Errorf(
				"billing end time must not be zero",
			)
	}

	if !query.End.After(
		query.Start,
	) {

		return billing.CostReport{},
			fmt.Errorf(
				"billing end time must be after start time",
			)
	}

	return s.client.GetCost(
		ctx,
		analysisContext,
		query,
	)
}

type CostExplorerBillingClient struct {
	client CostExplorerAPI
}

func NewCostExplorerBillingClient(
	client CostExplorerAPI,
) *CostExplorerBillingClient {

	return &CostExplorerBillingClient{
		client: client,
	}
}

func (c *CostExplorerBillingClient) GetCost(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	query billing.CostQuery,
) (billing.CostReport, error) {

	if c.client == nil {

		return billing.CostReport{},
			fmt.Errorf(
				"AWS Cost Explorer client is not configured",
			)
	}

	if query.Start.IsZero() {

		return billing.CostReport{},
			fmt.Errorf(
				"billing start time must not be zero",
			)
	}

	if query.End.IsZero() {

		return billing.CostReport{},
			fmt.Errorf(
				"billing end time must not be zero",
			)
	}

	if !query.End.After(
		query.Start,
	) {

		return billing.CostReport{},
			fmt.Errorf(
				"billing end time must be after start time",
			)
	}

	start :=
		query.Start.UTC()

	end :=
		query.End.UTC()

	startDate :=
		start.Format(
			"2006-01-02",
		)

	endDate :=
		end.Format(
			"2006-01-02",
		)

	if startDate == endDate {

		endDate =
			end.Add(
				24 * time.Hour,
			).Format(
				"2006-01-02",
			)
	}

	input :=
		&costexplorer.GetCostAndUsageInput{
			TimePeriod: &types.DateInterval{
				Start: aws.String(
					startDate,
				),

				End: aws.String(
					endDate,
				),
			},

			Granularity: awsCostGranularity,

			Metrics: []string{
				awsCostMetric,
			},

			GroupBy: []types.GroupDefinition{
				{
					Type: awsServiceGroup,

					Key: aws.String(
						"SERVICE",
					),
				},
			},
		}

	if query.Service != "" {

		input.Filter =
			&types.Expression{
				Dimensions: &types.DimensionValues{
					Key: awsServiceDimension,

					Values: []string{
						query.Service,
					},
				},
			}
	}

	output, err :=
		c.client.GetCostAndUsage(
			ctx,
			input,
		)

	if err != nil {

		return billing.CostReport{},
			fmt.Errorf(
				"AWS Cost Explorer GetCostAndUsage: %w",
				err,
			)
	}

	return mapCostExplorerResponse(
		output,
		start,
		end,
	), nil
}

func mapCostExplorerResponse(
	output *costexplorer.GetCostAndUsageOutput,
	start time.Time,
	end time.Time,
) billing.CostReport {

	report :=
		billing.CostReport{
			Provider: domain.CloudProviderAWS,

			Start: start,

			End: end,

			Currency: awsBillingCurrency,

			Periods: make(
				[]billing.CostPeriod,
				0,
			),
		}

	if output == nil {

		return report
	}

	for _, result := range output.ResultsByTime {

		estimated := result.Estimated

		if estimated {

			report.Estimated =
				true
		}

		for _, group := range result.Groups {

			service :=
				""

			if len(group.Keys) > 0 {

				service =
					group.Keys[0]
			}

			metric,
				ok :=
				group.Metrics[awsCostMetric]

			if !ok {

				continue
			}

			amount := 0.0

			if metric.Amount != nil {

				parsed,
					err :=
					strconv.ParseFloat(
						*metric.Amount,
						64,
					)

				if err == nil {

					amount =
						parsed
				}
			}

			unit :=
				awsBillingCurrency

			if metric.Unit != nil {

				unit =
					*metric.Unit
			}

			periodStart :=
				start

			periodEnd :=
				end

			if result.TimePeriod != nil {

				if result.TimePeriod.Start != nil {

					if parsed,
						err :=
						time.Parse(
							"2006-01-02",
							*result.TimePeriod.Start,
						); err == nil {

						periodStart =
							parsed.UTC()
					}
				}

				if result.TimePeriod.End != nil {

					if parsed,
						err :=
						time.Parse(
							"2006-01-02",
							*result.TimePeriod.End,
						); err == nil {

						periodEnd =
							parsed.UTC()
					}
				}
			}

			report.Periods =
				append(
					report.Periods,
					billing.CostPeriod{
						Start: periodStart,

						End: periodEnd,

						Service: service,

						AmountUSD: amount,

						Unit: unit,

						Estimated: estimated,
					},
				)

			report.TotalUSD +=
				amount
		}
	}

	return report
}
