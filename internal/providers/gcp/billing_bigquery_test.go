package gcp

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/domain"

	"cloud.google.com/go/bigquery"
)

type billingRowsMock struct {
	rows   []GCPBillingRow
	sql    string
	params []bigquery.QueryParameter
}

func (m *billingRowsMock) Query(_ context.Context, sql string, params []bigquery.QueryParameter) ([]GCPBillingRow, error) {
	m.sql = sql
	m.params = params
	return m.rows, nil
}

func TestBigQueryBillingClient_ShouldMapUSDRows(t *testing.T) {
	mock := &billingRowsMock{rows: []GCPBillingRow{
		{Amount: 100.25, Currency: "USD", Service: "Google Kubernetes Engine"},
		{Amount: 25.75, Currency: "USD", Service: "Google Kubernetes Engine"},
	}}

	client, err := NewBigQueryBillingClient("billing-project", "finops", "gcp_billing_export_resource_v1", mock)
	if err != nil {
		t.Fatal(err)
	}

	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	result, err := client.GetCostWithContext(context.Background(), domain.AnalysisContext{
		Provider:  domain.CloudProviderGCP,
		AccountID: "workload-project",
	}, billing.CostQuery{Start: start, End: end})
	if err != nil {
		t.Fatal(err)
	}

	if result.TotalUSD != 126 {
		t.Fatalf("expected 126 USD, got %f", result.TotalUSD)
	}
	if result.Provider != domain.CloudProviderGCP {
		t.Fatalf("expected GCP provider, got %q", result.Provider)
	}
	if len(result.Periods) != 2 {
		t.Fatalf("expected 2 periods, got %d", len(result.Periods))
	}
	if mock.sql == "" || len(mock.params) != 4 {
		t.Fatal("expected parameterized billing query")
	}
}

func TestBigQueryBillingClient_ShouldRejectMissingProject(t *testing.T) {
	client, err := NewBigQueryBillingClient("billing-project", "finops", "billing", &billingRowsMock{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.GetCostWithContext(context.Background(), domain.AnalysisContext{Provider: domain.CloudProviderGCP}, billing.CostQuery{
		Start: time.Now().Add(-time.Hour),
		End:   time.Now(),
	})
	if err == nil {
		t.Fatal("expected project validation error")
	}
}
