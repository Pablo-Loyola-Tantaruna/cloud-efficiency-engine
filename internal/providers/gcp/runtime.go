package gcp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/domain"
	metricproviders "cloud-efficiency-engine/internal/metrics/providers"
	providerregistry "cloud-efficiency-engine/internal/providers"
)

func RegisterRuntime(
	ctx context.Context,
	registry *providerregistry.Registry,
	analysisContext ProjectContext,
	prometheusURL string,
	httpClient *http.Client,
) (*Clients, error) {
	if registry == nil {
		return nil, fmt.Errorf("GCP provider registry must not be nil")
	}
	if strings.TrimSpace(analysisContext.ProjectID) == "" {
		return nil, fmt.Errorf("GCP project ID must not be empty")
	}
	if strings.TrimSpace(analysisContext.Region) == "" {
		return nil, fmt.Errorf("GCP GKE location must not be empty")
	}
	if strings.TrimSpace(analysisContext.Cluster) == "" {
		return nil, fmt.Errorf("GCP GKE cluster name must not be empty")
	}
	if prometheusURL == "" {
		return nil, fmt.Errorf("Prometheus URL must not be empty")
	}

	clients, err := NewClients(ctx, analysisContext.ProjectID)
	if err != nil {
		return nil, err
	}

	prometheus := metricproviders.NewPrometheusProvider(prometheusURL, httpClient)
	metricsSource := providerregistry.NewMetricsSourceAdapter(prometheus, prometheus)

	clusterReader := NewGKEClusterManagerClientReader(clients.ClusterManager)
	machineReader := NewGCPMachineTypeReader(clients.MachineTypes)
	managedGroupReader := NewGCPManagedInstanceGroupReader(clients.InstanceGroupManagers)

	capacitySource, err := NewGKECapacitySource(
		clusterReader,
		machineReader,
		managedGroupReader,
	)
	if err != nil {
		_ = clients.Close()
		return nil, err
	}

	pricingAPIKey := strings.TrimSpace(os.Getenv("GCP_BILLING_API_KEY"))
	pricingClient, err := NewGCPCatalogPricingClient(httpClientOrDefault(httpClient), pricingAPIKey)
	if err != nil {
		_ = clients.Close()
		return nil, err
	}

	billingProject := strings.TrimSpace(os.Getenv("GCP_BILLING_QUERY_PROJECT"))
	billingDataset := strings.TrimSpace(os.Getenv("GCP_BILLING_DATASET"))
	billingTable := strings.TrimSpace(os.Getenv("GCP_BILLING_TABLE"))
	billingQueryer := NewBigQueryBillingQueryer(clients.BigQuery)
	billingSource, err := NewBigQueryBillingClient(
		billingProject,
		billingDataset,
		billingTable,
		billingQueryer,
	)
	if err != nil {
		_ = clients.Close()
		return nil, err
	}
	billingClient := &billingContextAdapter{source: billingSource}

	if err := Register(
		registry,
		metricsSource,
		pricingClient,
		billingClient,
		capacitySource,
	); err != nil {
		_ = clients.Close()
		return nil, err
	}

	return clients, nil
}

type ProjectContext struct {
	ProjectID string
	Region    string
	Cluster   string
}

type billingContextAdapter struct {
	source *BigQueryBillingClient
}

func (a *billingContextAdapter) GetCost(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	query billing.CostQuery,
) (billing.CostReport, error) {
	if a == nil || a.source == nil {
		return billing.CostReport{}, fmt.Errorf("GCP billing context adapter is not configured")
	}
	return a.source.GetCostWithContext(ctx, analysisContext, query)
}

func httpClientOrDefault(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{}
}

var _ BillingClient = (*billingContextAdapter)(nil)
