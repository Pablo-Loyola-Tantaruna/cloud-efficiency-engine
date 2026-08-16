package azure

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/domain"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/costmanagement/armcostmanagement/v3"
)

type billingQueryMock struct {
	calls int

	scope string

	parameters armcostmanagement.QueryDefinition

	response armcostmanagement.QueryClientUsageResponse
}

func (m *billingQueryMock) Usage(
	ctx context.Context,
	scope string,
	parameters armcostmanagement.QueryDefinition,
	options *armcostmanagement.QueryClientUsageOptions,
) (armcostmanagement.QueryClientUsageResponse, error) {

	m.calls++
	m.scope = scope
	m.parameters = parameters

	return m.response, nil
}

func TestBillingSource_ShouldMapCostManagementResponse(t *testing.T) {

	mock := &billingQueryMock{
		response: armcostmanagement.QueryClientUsageResponse{
			QueryResult: armcostmanagement.QueryResult{
				Properties: &armcostmanagement.QueryProperties{
					Columns: []*armcostmanagement.QueryColumn{
						{Name: to.Ptr("totalCost")},
						{Name: to.Ptr("Currency")},
					},
					Rows: [][]any{
						{100.25, "USD"},
						{25.75, "USD"},
					},
				},
			},
		},
	}

	source := NewBillingSource(mock)

	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	result, err := source.GetCostWithContext(
		context.Background(),
		domain.AnalysisContext{
			Provider:  domain.CloudProviderAzure,
			AccountID: "subscription-123",
		},
		billing.CostQuery{
			Start:   start,
			End:     end,
			Service: "Azure Kubernetes Service",
		},
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if mock.calls != 1 {
		t.Fatalf("expected one Usage call, got %d", mock.calls)
	}

	if mock.scope != "subscriptions/subscription-123" {
		t.Fatalf("unexpected scope: %s", mock.scope)
	}

	if result.Provider != domain.CloudProviderAzure {
		t.Fatalf("expected Azure provider, got %q", result.Provider)
	}

	if result.Currency != "USD" {
		t.Fatalf("expected USD currency, got %q", result.Currency)
	}

	if result.TotalUSD != 126.0 {
		t.Fatalf("expected total cost 126, got %f", result.TotalUSD)
	}

	if len(result.Periods) != 1 {
		t.Fatalf("expected one period, got %d", len(result.Periods))
	}

	if result.Periods[0].Service != "Azure Kubernetes Service" {
		t.Fatalf("unexpected service: %q", result.Periods[0].Service)
	}
}

func TestBillingSource_ShouldValidateSubscriptionContext(t *testing.T) {

	source := NewBillingSource(&billingQueryMock{})

	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	_, err := source.GetCostWithContext(
		context.Background(),
		domain.AnalysisContext{
			Provider: domain.CloudProviderAzure,
		},
		billing.CostQuery{
			Start: start,
			End:   end,
		},
	)

	if err == nil {
		t.Fatal("expected missing subscription validation error")
	}
}

func TestBillingSource_ShouldRejectNonUSD(t *testing.T) {

	mock := &billingQueryMock{
		response: armcostmanagement.QueryClientUsageResponse{
			QueryResult: armcostmanagement.QueryResult{
				Properties: &armcostmanagement.QueryProperties{
					Columns: []*armcostmanagement.QueryColumn{
						{Name: to.Ptr("totalCost")},
						{Name: to.Ptr("Currency")},
					},
					Rows: [][]any{
						{100.0, "EUR"},
					},
				},
			},
		},
	}

	source := NewBillingSource(mock)

	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	result, err := source.GetCostWithContext(
		context.Background(),
		domain.AnalysisContext{
			Provider:  domain.CloudProviderAzure,
			AccountID: "subscription-123",
		},
		billing.CostQuery{
			Start: start,
			End:   end,
		},
	)

	if err != nil {
		t.Fatalf("expected no transport error, got %v", err)
	}

	if result.Currency != "EUR" {
		t.Fatalf("expected EUR currency, got %q", result.Currency)
	}

	if result.TotalUSD != 0 {
		t.Fatalf("expected USD total to remain zero for non-USD billing, got %f", result.TotalUSD)
	}
}

func TestAzureFloat64_ShouldConvertNumbers(t *testing.T) {

	values := []any{
		float64(1.5),
		float32(2.5),
		int(3),
		int32(4),
		int64(5),
		uint(6),
		uint32(7),
		uint64(8),
	}

	for _, value := range values {
		if _, ok := azureFloat64(value); !ok {
			t.Fatalf("expected numeric value %T to convert", value)
		}
	}
}
