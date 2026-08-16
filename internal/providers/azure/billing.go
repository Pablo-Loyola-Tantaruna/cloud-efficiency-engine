package azure

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/domain"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/costmanagement/armcostmanagement/v3"
)

const (
	azureBillingCurrency = "USD"
	azureCostMetric      = "PreTaxCost"
)

type CostManagementQueryClient interface {
	Usage(
		ctx context.Context,
		scope string,
		parameters armcostmanagement.QueryDefinition,
		options *armcostmanagement.QueryClientUsageOptions,
	) (armcostmanagement.QueryClientUsageResponse, error)
}

type BillingSource struct {
	client CostManagementQueryClient
}

func NewBillingSource(client CostManagementQueryClient) *BillingSource {
	return &BillingSource{client: client}
}

func (s *BillingSource) GetCost(ctx context.Context, query billing.CostQuery) (billing.CostReport, error) {
	return s.GetCostWithContext(ctx, domain.AnalysisContext{}, query)
}

func (s *BillingSource) GetCostWithContext(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	query billing.CostQuery,
) (billing.CostReport, error) {
	if s == nil || s.client == nil {
		return billing.CostReport{}, fmt.Errorf("Azure Cost Management client is not configured")
	}

	if query.Start.IsZero() {
		return billing.CostReport{}, fmt.Errorf("billing start time must not be zero")
	}

	if query.End.IsZero() {
		return billing.CostReport{}, fmt.Errorf("billing end time must not be zero")
	}

	if !query.End.After(query.Start) {
		return billing.CostReport{}, fmt.Errorf("billing end time must be after start time")
	}

	providerContext := domain.NormalizeAnalysisContext(analysisContext)
	if providerContext.AccountID == "" {
		return billing.CostReport{}, fmt.Errorf(
			"Azure subscription ID must not be empty in analysis context accountId",
		)
	}

	scope := "subscriptions/" + providerContext.AccountID
	parameters := armcostmanagement.QueryDefinition{
		Type: to.Ptr(armcostmanagement.ExportTypeUsage),
		Dataset: &armcostmanagement.QueryDataset{
			Aggregation: map[string]*armcostmanagement.QueryAggregation{
				"totalCost": {
					Name:     to.Ptr(azureCostMetric),
					Function: to.Ptr(armcostmanagement.FunctionTypeSum),
				},
			},
			Granularity: to.Ptr(armcostmanagement.GranularityType("None")),
		},
		Timeframe: to.Ptr(armcostmanagement.TimeframeTypeCustom),
		TimePeriod: &armcostmanagement.QueryTimePeriod{
			From: to.Ptr(query.Start.UTC()),
			To:   to.Ptr(query.End.UTC()),
		},
	}

	if query.Service != "" {
		parameters.Dataset.Filter = &armcostmanagement.QueryFilter{
			Dimensions: &armcostmanagement.QueryComparisonExpression{
				Name:     to.Ptr("ServiceName"),
				Operator: to.Ptr(armcostmanagement.QueryOperatorTypeIn),
				Values:   []*string{to.Ptr(query.Service)},
			},
		}
	}

	response, err := s.client.Usage(ctx, scope, parameters, nil)
	if err != nil {
		return billing.CostReport{}, fmt.Errorf("query Azure Cost Management usage: %w", err)
	}

	return mapAzureCostManagementResponse(response, query.Start.UTC(), query.End.UTC(), query.Service), nil
}

func mapAzureCostManagementResponse(
	response armcostmanagement.QueryClientUsageResponse,
	start time.Time,
	end time.Time,
	service string,
) billing.CostReport {
	report := billing.CostReport{
		Provider:  domain.CloudProviderAzure,
		Start:     start,
		End:       end,
		Currency:  azureBillingCurrency,
		Periods:   []billing.CostPeriod{},
		Estimated: false,
	}

	properties := response.QueryResult.Properties
	if properties == nil || properties.Columns == nil {
		return report
	}

	costIndex := -1
	currencyIndex := -1
	for index, column := range properties.Columns {
		if column == nil || column.Name == nil {
			continue
		}
		switch strings.ToLower(*column.Name) {
		case strings.ToLower(azureCostMetric), "totalcost":
			costIndex = index
		case "currency":
			currencyIndex = index
		}
	}

	if costIndex < 0 {
		return report
	}

	for _, row := range properties.Rows {
		if costIndex >= len(row) {
			continue
		}

		amount, ok := azureFloat64(row[costIndex])
		if !ok || math.IsNaN(amount) || math.IsInf(amount, 0) {
			continue
		}

		report.TotalUSD += amount

		currency := azureBillingCurrency
		if currencyIndex >= 0 && currencyIndex < len(row) {
			if value, ok := row[currencyIndex].(string); ok && value != "" {
				currency = value
			}
		}

		if report.Currency == azureBillingCurrency && currency != azureBillingCurrency {
			report.Currency = currency
		}
	}

	if report.Currency != azureBillingCurrency {
		report.TotalUSD = 0
		return report
	}

	report.Periods = append(report.Periods, billing.CostPeriod{
		Start:     start,
		End:       end,
		Service:   service,
		AmountUSD: report.TotalUSD,
		Unit:      azureBillingCurrency,
		Estimated: report.Estimated,
	})

	return report
}

func azureFloat64(value any) (float64, bool) {
	switch value := value.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case int32:
		return float64(value), true
	case uint:
		return float64(value), true
	case uint64:
		return float64(value), true
	case uint32:
		return float64(value), true
	default:
		return 0, false
	}
}

var _ billing.Provider = (*BillingSource)(nil)
var _ billing.ContextAwareProvider = (*BillingSource)(nil)
