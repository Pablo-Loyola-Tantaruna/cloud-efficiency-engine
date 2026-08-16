package gcp

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/domain"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

var bigQueryIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9_\-]+$`)

type GCPBillingQueryClient interface {
	Query(ctx context.Context, sql string, parameters []bigquery.QueryParameter) ([]GCPBillingRow, error)
}

type GCPBillingRow struct {
	Amount   float64
	Currency string
	Service  string
}

type BigQueryBillingClient struct {
	queryProject string
	dataset      string
	table        string
	queryer      GCPBillingQueryClient
}

func NewBigQueryBillingClient(queryProject, dataset, table string, queryer GCPBillingQueryClient) (*BigQueryBillingClient, error) {
	for name, value := range map[string]string{
		"GCP billing query project": queryProject,
		"GCP billing dataset":       dataset,
		"GCP billing table":         table,
	} {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("%s must not be empty", name)
		}
		if !bigQueryIdentifierPattern.MatchString(value) {
			return nil, fmt.Errorf("%s contains invalid characters", name)
		}
	}
	if queryer == nil {
		return nil, fmt.Errorf("GCP billing query client must not be nil")
	}
	return &BigQueryBillingClient{
		queryProject: queryProject,
		dataset:      dataset,
		table:        table,
		queryer:      queryer,
	}, nil
}

// GetCost implements the billing.Provider contract using the billing query
// project as the default GCP account/project context.
func (c *BigQueryBillingClient) GetCost(
	ctx context.Context,
	query billing.CostQuery,
) (billing.CostReport, error) {
	if c == nil {
		return billing.CostReport{}, fmt.Errorf("GCP BigQuery billing client is not configured")
	}
	return c.GetCostWithContext(
		ctx,
		domain.AnalysisContext{
			Provider:  domain.CloudProviderGCP,
			AccountID: c.queryProject,
		},
		query,
	)
}

// GetCostWithContext keeps the cloud runtime account/project explicit while
// reusing the same billing query implementation.
func (c *BigQueryBillingClient) GetCostWithContext(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	query billing.CostQuery,
) (billing.CostReport, error) {
	if c == nil || c.queryer == nil {
		return billing.CostReport{}, fmt.Errorf("GCP BigQuery billing client is not configured")
	}
	if query.Start.IsZero() || query.End.IsZero() || !query.End.After(query.Start) {
		return billing.CostReport{}, fmt.Errorf("GCP billing query requires a valid time range")
	}

	projectID := strings.TrimSpace(analysisContext.AccountID)
	if projectID == "" {
		return billing.CostReport{}, fmt.Errorf("GCP project ID must not be empty in analysis context accountId")
	}

	tableName := fmt.Sprintf("%s.%s.%s", c.queryProject, c.dataset, c.table)
	sql := "SELECT\n" +
		"  SUM(CAST(cost AS FLOAT64)) AS amount,\n" +
		"  ANY_VALUE(currency) AS currency,\n" +
		"  ANY_VALUE(service.description) AS service\n" +
		"FROM `" + tableName + "`\n" +
		"WHERE usage_start_time >= @start_time\n" +
		"  AND usage_start_time < @end_time\n" +
		"  AND project.id = @project_id\n" +
		"  AND (@service = '' OR service.description = @service)"

	rows, err := c.queryer.Query(ctx, sql, []bigquery.QueryParameter{
		{Name: "start_time", Value: query.Start.UTC()},
		{Name: "end_time", Value: query.End.UTC()},
		{Name: "project_id", Value: projectID},
		{Name: "service", Value: query.Service},
	})
	if err != nil {
		return billing.CostReport{}, fmt.Errorf("query GCP billing export: %w", err)
	}

	report := billing.CostReport{
		Provider: domain.CloudProviderGCP,
		Start:    query.Start.UTC(),
		End:      query.End.UTC(),
		Currency: "USD",
		Periods:  []billing.CostPeriod{},
	}

	for _, row := range rows {
		if row.Currency != "" {
			report.Currency = row.Currency
		}
		service := row.Service
		if service == "" {
			service = query.Service
		}
		amountUSD := row.Amount
		if report.Currency == "USD" {
			report.TotalUSD += amountUSD
		} else {
			amountUSD = 0
		}
		report.Periods = append(report.Periods, billing.CostPeriod{
			Start:     query.Start.UTC(),
			End:       query.End.UTC(),
			Service:   service,
			AmountUSD: amountUSD,
			Unit:      report.Currency,
			Estimated: false,
		})
	}

	if report.Currency != "USD" {
		report.TotalUSD = 0
	}
	return report, nil
}

type BigQueryBillingQueryer struct {
	client *bigquery.Client
}

func NewBigQueryBillingQueryer(client *bigquery.Client) *BigQueryBillingQueryer {
	return &BigQueryBillingQueryer{client: client}
}

func (q *BigQueryBillingQueryer) Query(
	ctx context.Context,
	sql string,
	parameters []bigquery.QueryParameter,
) ([]GCPBillingRow, error) {
	if q == nil || q.client == nil {
		return nil, fmt.Errorf("GCP BigQuery client is not configured")
	}

	query := q.client.Query(sql)
	query.Parameters = parameters
	iteratorRows, err := query.Read(ctx)
	if err != nil {
		return nil, err
	}

	var rows []GCPBillingRow
	for {
		var row struct {
			Amount   float64 `bigquery:"amount"`
			Currency string  `bigquery:"currency"`
			Service  string  `bigquery:"service"`
		}
		if err := iteratorRows.Next(&row); err == iterator.Done {
			break
		} else if err != nil {
			return nil, err
		}
		rows = append(rows, GCPBillingRow{
			Amount:   row.Amount,
			Currency: row.Currency,
			Service:  row.Service,
		})
	}
	return rows, nil
}

var _ billing.Provider = (*BigQueryBillingClient)(nil)
var _ billing.ContextAwareProvider = (*BigQueryBillingClient)(nil)
