package azure

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics/providers"
	providerregistry "cloud-efficiency-engine/internal/providers"
)

// KubernetesMetricsClient adapts the shared Prometheus workload metrics
// implementation to the Azure AKS runtime contract. Workloads remain
// Kubernetes objects; Azure only contributes cloud-specific capacity, billing
// and pricing.
type KubernetesMetricsClient struct {
	provider *providers.PrometheusProvider
}

func NewKubernetesMetricsClient(
	prometheusURL string,
	httpClient *http.Client,
) *KubernetesMetricsClient {
	return &KubernetesMetricsClient{
		provider: providers.NewPrometheusProvider(
			prometheusURL,
			httpClient,
		),
	}
}

func (c *KubernetesMetricsClient) GetWorkloads(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
) ([]domain.WorkloadMetrics, error) {
	if c == nil || c.provider == nil {
		return nil, fmt.Errorf("Azure Kubernetes metrics client is not configured")
	}

	return c.provider.GetWorkloadsWithContext(
		ctx,
		analysisContext,
		namespace,
	)
}

func (c *KubernetesMetricsClient) GetWorkloadHistory(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {
	if c == nil || c.provider == nil {
		return nil, fmt.Errorf("Azure Kubernetes metrics client is not configured")
	}

	return c.provider.GetWorkloadHistoryWithContext(
		ctx,
		analysisContext,
		namespace,
		start,
		end,
		step,
	)
}

type billingContextAdapter struct {
	source *BillingSource
}

func (a *billingContextAdapter) GetCost(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	query billing.CostQuery,
) (billing.CostReport, error) {
	if a == nil || a.source == nil {
		return billing.CostReport{}, fmt.Errorf("Azure billing context adapter is not configured")
	}
	return a.source.GetCostWithContext(ctx, analysisContext, query)
}

// RegisterRuntime composes the real Azure cloud runtime without putting
// provider-specific logic into the analysis engine.
func RegisterRuntime(
	registry *providerregistry.Registry,
	clients *Clients,
	prometheusURL string,
	httpClient *http.Client,
) error {
	if registry == nil {
		return fmt.Errorf("provider registry must not be nil")
	}
	if clients == nil {
		return fmt.Errorf("Azure clients must not be nil")
	}
	if clients.ManagedClusters == nil {
		return fmt.Errorf("Azure managed clusters client must not be nil")
	}
	if clients.VirtualMachineSizes == nil {
		return fmt.Errorf("Azure VM sizes client must not be nil")
	}
	if clients.Query == nil {
		return fmt.Errorf("Azure Cost Management query client must not be nil")
	}
	if prometheusURL == "" {
		return fmt.Errorf("Prometheus URL must not be empty")
	}

	metricsClient := NewKubernetesMetricsClient(
		prometheusURL,
		httpClient,
	)

	clusterReader := NewARMManagedClusterReader(
		clients.ManagedClusters,
	)

	sizeReader := NewARMVMSizeReader(
		clients.VirtualMachineSizes,
	)

	pricingHTTPClient := httpClient
	if pricingHTTPClient == nil {
		pricingHTTPClient = http.DefaultClient
	}

	retailPricesClient := NewAzureRetailPricesClient(
		pricingHTTPClient,
	)

	pricingSource, err := NewAzurePricingSource(
		clusterReader,
		retailPricesClient,
		sizeReader,
	)
	if err != nil {
		return fmt.Errorf("create Azure pricing source: %w", err)
	}

	billingSource := NewBillingSource(
		clients.Query,
	)
	billingClient := &billingContextAdapter{source: billingSource}

	capacitySource, err := NewAKSCapacitySource(
		clusterReader,
		sizeReader,
	)
	if err != nil {
		return fmt.Errorf("create Azure capacity source: %w", err)
	}

	if err := Register(
		registry,
		metricsClient,
		pricingSource,
		billingClient,
		capacitySource,
	); err != nil {
		return fmt.Errorf("register Azure runtime: %w", err)
	}

	return nil
}

var _ MetricsClient = (*KubernetesMetricsClient)(nil)
var _ BillingClient = (*billingContextAdapter)(nil)
