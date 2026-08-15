package gcp

import (
	"context"
	"fmt"
	"time"

	"cloud-efficiency-engine/internal/analysis/capacity"
	"cloud-efficiency-engine/internal/billing"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/pricing"
)

type MetricsClient interface {
	GetWorkloads(ctx context.Context, analysisContext domain.AnalysisContext, namespace string) ([]domain.WorkloadMetrics, error)
	GetWorkloadHistory(ctx context.Context, analysisContext domain.AnalysisContext, namespace string, start time.Time, end time.Time, step time.Duration) ([]domain.WorkloadHistory, error)
}

type PricingClient interface {
	GetPricing(ctx context.Context, analysisContext domain.AnalysisContext) (pricing.ResourcePricing, error)
}

type BillingClient interface {
	GetCost(ctx context.Context, analysisContext domain.AnalysisContext, query billing.CostQuery) (billing.CostReport, error)
}

type CapacityClient interface {
	GetCapacity(ctx context.Context, analysisContext domain.AnalysisContext) (int64, int64, error)
}

type Provider struct {
	metricsClient  MetricsClient
	pricingClient  PricingClient
	billingClient  BillingClient
	capacityClient CapacityClient
}

func NewProvider(metricsClient MetricsClient, pricingClient PricingClient, billingClient BillingClient, capacityClient CapacityClient) (*Provider, error) {
	if metricsClient == nil {
		return nil, fmt.Errorf("GCP metrics client must not be nil")
	}
	if pricingClient == nil {
		return nil, fmt.Errorf("GCP pricing client must not be nil")
	}
	if billingClient == nil {
		return nil, fmt.Errorf("GCP billing client must not be nil")
	}
	if capacityClient == nil {
		return nil, fmt.Errorf("GCP capacity client must not be nil")
	}
	return &Provider{metricsClient: metricsClient, pricingClient: pricingClient, billingClient: billingClient, capacityClient: capacityClient}, nil
}

func (p *Provider) GetWorkloads(ctx context.Context, namespace string) ([]domain.WorkloadMetrics, error) {
	return p.GetWorkloadsWithContext(ctx, domain.AnalysisContext{Provider: domain.CloudProviderGCP}, namespace)
}

func (p *Provider) GetWorkloadsWithContext(ctx context.Context, analysisContext domain.AnalysisContext, namespace string) ([]domain.WorkloadMetrics, error) {
	if p == nil || p.metricsClient == nil {
		return nil, fmt.Errorf("GCP metrics provider is not configured")
	}
	return p.metricsClient.GetWorkloads(ctx, analysisContext, namespace)
}

func (p *Provider) GetWorkloadHistory(ctx context.Context, namespace string, start time.Time, end time.Time, step time.Duration) ([]domain.WorkloadHistory, error) {
	return p.GetWorkloadHistoryWithContext(ctx, domain.AnalysisContext{Provider: domain.CloudProviderGCP}, namespace, start, end, step)
}

func (p *Provider) GetWorkloadHistoryWithContext(ctx context.Context, analysisContext domain.AnalysisContext, namespace string, start time.Time, end time.Time, step time.Duration) ([]domain.WorkloadHistory, error) {
	if p == nil || p.metricsClient == nil {
		return nil, fmt.Errorf("GCP metrics provider is not configured")
	}
	return p.metricsClient.GetWorkloadHistory(ctx, analysisContext, namespace, start, end, step)
}

func (p *Provider) GetPricing(ctx context.Context) (pricing.ResourcePricing, error) {
	return p.GetPricingWithContext(ctx, domain.AnalysisContext{Provider: domain.CloudProviderGCP})
}

func (p *Provider) GetPricingWithContext(ctx context.Context, analysisContext domain.AnalysisContext) (pricing.ResourcePricing, error) {
	if p == nil || p.pricingClient == nil {
		return pricing.ResourcePricing{}, fmt.Errorf("GCP pricing provider is not configured")
	}
	return p.pricingClient.GetPricing(ctx, analysisContext)
}

func (p *Provider) GetCost(ctx context.Context, query billing.CostQuery) (billing.CostReport, error) {
	return p.GetCostWithContext(ctx, domain.AnalysisContext{Provider: domain.CloudProviderGCP}, query)
}

func (p *Provider) GetCostWithContext(ctx context.Context, analysisContext domain.AnalysisContext, query billing.CostQuery) (billing.CostReport, error) {
	if p == nil || p.billingClient == nil {
		return billing.CostReport{}, fmt.Errorf("GCP billing provider is not configured")
	}
	return p.billingClient.GetCost(ctx, analysisContext, query)
}

func (p *Provider) GetCapacity(ctx context.Context, analysisContext domain.AnalysisContext) (cost.ClusterCapacity, error) {
	if p == nil || p.capacityClient == nil {
		return cost.ClusterCapacity{}, fmt.Errorf("GCP capacity provider is not configured")
	}
	cpu, memory, err := p.capacityClient.GetCapacity(ctx, analysisContext)
	if err != nil {
		return cost.ClusterCapacity{}, err
	}
	return cost.ClusterCapacity{CPUCapacityMillicores: cpu, MemoryCapacityBytes: memory}, nil
}

var _ metrics.Provider = (*Provider)(nil)
var _ metrics.ContextAwareProvider = (*Provider)(nil)
var _ metrics.HistoricalProvider = (*Provider)(nil)
var _ metrics.ContextAwareHistoricalProvider = (*Provider)(nil)
var _ pricing.Provider = (*Provider)(nil)
var _ pricing.ContextAwareProvider = (*Provider)(nil)
var _ billing.Provider = (*Provider)(nil)
var _ billing.ContextAwareProvider = (*Provider)(nil)
var _ capacity.Provider = (*Provider)(nil)
