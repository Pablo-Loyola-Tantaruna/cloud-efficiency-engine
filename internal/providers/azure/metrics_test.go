package azure

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
)

type metricsInventoryMock struct{}

func (m *metricsInventoryMock) ListResources(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) ([]AzureResource, error) {

	return []AzureResource{
		{
			ID:                   "/subscriptions/test/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/api",
			Namespace:            "production",
			Name:                 "api",
			Type:                 domain.WorkloadUnknown,
			CPURequestMillicores: 4000,
			MemoryRequestBytes:   16 * 1024 * 1024 * 1024,
		},
	}, nil
}

func TestFormatAzureInterval(t *testing.T) {

	tests := []struct {
		name string
		step time.Duration
		want string
	}{
		{
			name: "seconds",
			step: 30 * time.Second,
			want: "PT30S",
		},
		{
			name: "minutes",
			step: 5 * time.Minute,
			want: "PT5M",
		},
		{
			name: "hours",
			step: time.Hour,
			want: "PT1H",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatAzureInterval(tt.step); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestAzureMonitorMetricsClient_ShouldRejectMissingMonitor(t *testing.T) {

	client := NewAzureMonitorMetricsClient(
		nil,
		&metricsInventoryMock{},
	)

	_, err := client.GetWorkloads(
		context.Background(),
		domain.AnalysisContext{
			Provider: domain.CloudProviderAzure,
		},
		"production",
	)

	if err == nil {
		t.Fatal("expected configuration error")
	}
}

func TestAzureMonitorMetricsClient_ShouldRejectMissingInventory(t *testing.T) {

	client := NewAzureMonitorMetricsClient(
		&monitorMetricsMock{},
		nil,
	)

	_, err := client.GetWorkloads(
		context.Background(),
		domain.AnalysisContext{
			Provider: domain.CloudProviderAzure,
		},
		"production",
	)

	if err == nil {
		t.Fatal("expected configuration error")
	}
}

type monitorMetricsMock struct{}

func (m *monitorMetricsMock) QueryResource(
	ctx context.Context,
	resourceURI string,
	options *azquery.MetricsClientQueryResourceOptions,
) (azquery.MetricsClientQueryResourceResponse, error) {

	return azquery.MetricsClientQueryResourceResponse{}, nil
}
