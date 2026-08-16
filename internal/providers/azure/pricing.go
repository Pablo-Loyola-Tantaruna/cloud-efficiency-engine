package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"

	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/pricing"
)

const azureRetailPricesEndpoint = "https://prices.azure.com/api/retail/prices"

type RetailPriceItem struct {
	CurrencyCode         string  `json:"currencyCode"`
	RetailPrice          float64 `json:"retailPrice"`
	UnitPrice            float64 `json:"unitPrice"`
	ArmRegionName        string  `json:"armRegionName"`
	MeterName            string  `json:"meterName"`
	ProductName          string  `json:"productName"`
	SKUName              string  `json:"skuName"`
	ServiceName          string  `json:"serviceName"`
	UnitOfMeasure        string  `json:"unitOfMeasure"`
	Type                 string  `json:"type"`
	IsPrimaryMeterRegion bool    `json:"isPrimaryMeterRegion"`
	ArmSKUName           string  `json:"armSkuName"`
}

type retailPriceResponse struct {
	Items        []RetailPriceItem `json:"Items"`
	NextPageLink string            `json:"NextPageLink"`
}

type RetailPriceHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type AzureRetailPricesClient struct {
	httpClient RetailPriceHTTPClient
	endpoint   string
}

func NewAzureRetailPricesClient(
	httpClient RetailPriceHTTPClient,
) *AzureRetailPricesClient {
	return &AzureRetailPricesClient{
		httpClient: httpClient,
		endpoint:   azureRetailPricesEndpoint,
	}
}

func (c *AzureRetailPricesClient) GetHourlyPrice(
	ctx context.Context,
	region string,
	armSKUName string,
) (float64, error) {
	if c == nil || c.httpClient == nil {
		return 0, fmt.Errorf("Azure Retail Prices client is not configured")
	}
	region = strings.TrimSpace(region)
	armSKUName = strings.TrimSpace(armSKUName)
	if region == "" {
		return 0, fmt.Errorf("Azure pricing region must not be empty")
	}
	if armSKUName == "" {
		return 0, fmt.Errorf("Azure pricing ARM SKU name must not be empty")
	}

	filter := fmt.Sprintf(
		"serviceName eq 'Virtual Machines' and armRegionName eq '%s' and armSkuName eq '%s' and priceType eq 'Consumption'",
		escapeOData(region),
		escapeOData(armSKUName),
	)

	requestURL, err := url.Parse(c.endpoint)
	if err != nil {
		return 0, fmt.Errorf("parse Azure Retail Prices endpoint: %w", err)
	}
	query := requestURL.Query()
	query.Set("$filter", filter)
	requestURL.RawQuery = query.Encode()

	for requestURL != nil {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
		if err != nil {
			return 0, fmt.Errorf("create Azure Retail Prices request: %w", err)
		}
		request.Header.Set("Accept", "application/json")

		response, err := c.httpClient.Do(request)
		if err != nil {
			return 0, fmt.Errorf("call Azure Retail Prices API: %w", err)
		}

		body, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if readErr != nil {
			return 0, fmt.Errorf("read Azure Retail Prices response: %w", readErr)
		}
		if closeErr != nil {
			return 0, fmt.Errorf("close Azure Retail Prices response: %w", closeErr)
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			return 0, fmt.Errorf(
				"Azure Retail Prices API returned status %d: %s",
				response.StatusCode,
				strings.TrimSpace(string(body)),
			)
		}

		var payload retailPriceResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			return 0, fmt.Errorf("decode Azure Retail Prices response: %w", err)
		}

		if price, ok := chooseAzureHourlyPrice(payload.Items, region, armSKUName); ok {
			return price, nil
		}

		if payload.NextPageLink == "" {
			break
		}
		nextURL, err := url.Parse(payload.NextPageLink)
		if err != nil {
			return 0, fmt.Errorf("parse Azure Retail Prices next page: %w", err)
		}
		requestURL = nextURL
	}

	return 0, fmt.Errorf(
		"Azure hourly retail price not found for SKU %q in region %q",
		armSKUName,
		region,
	)
}

func chooseAzureHourlyPrice(
	items []RetailPriceItem,
	region string,
	armSKUName string,
) (float64, bool) {
	for _, item := range items {
		if !strings.EqualFold(item.CurrencyCode, "USD") {
			continue
		}
		if !strings.EqualFold(item.ArmRegionName, region) {
			continue
		}
		if !strings.EqualFold(item.ArmSKUName, armSKUName) {
			continue
		}
		if !strings.EqualFold(item.ServiceName, "Virtual Machines") {
			continue
		}
		if !strings.EqualFold(item.Type, "Consumption") {
			continue
		}
		if item.UnitOfMeasure != "1 Hour" {
			continue
		}
		if !item.IsPrimaryMeterRegion {
			continue
		}
		if strings.Contains(strings.ToLower(item.ProductName), "windows") {
			continue
		}
		if strings.Contains(strings.ToLower(item.MeterName), "spot") ||
			strings.Contains(strings.ToLower(item.ProductName), "spot") {
			continue
		}
		price := item.RetailPrice
		if price <= 0 {
			price = item.UnitPrice
		}
		if price > 0 {
			return price, true
		}
	}
	return 0, false
}

func escapeOData(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

type AzurePricingSource struct {
	clusters AKSClusterReader
	prices   *AzureRetailPricesClient
	sizes    AzureVMSizeReader
}

func NewAzurePricingSource(
	clusters AKSClusterReader,
	prices *AzureRetailPricesClient,
	sizes AzureVMSizeReader,
) (*AzurePricingSource, error) {
	if clusters == nil {
		return nil, fmt.Errorf("Azure AKS cluster reader must not be nil")
	}
	if prices == nil {
		return nil, fmt.Errorf("Azure Retail Prices client must not be nil")
	}
	if sizes == nil {
		return nil, fmt.Errorf("Azure VM size reader must not be nil")
	}
	return &AzurePricingSource{
		clusters: clusters,
		prices:   prices,
		sizes:    sizes,
	}, nil
}

func (s *AzurePricingSource) GetPricing(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (pricing.ResourcePricing, error) {
	if s == nil || s.clusters == nil || s.prices == nil || s.sizes == nil {
		return pricing.ResourcePricing{}, fmt.Errorf("Azure pricing source is not configured")
	}
	clusterName := strings.TrimSpace(analysisContext.ClusterName)
	if clusterName == "" {
		return pricing.ResourcePricing{}, fmt.Errorf("Azure AKS cluster name must not be empty")
	}
	cluster, err := s.clusters.FindCluster(ctx, clusterName)
	if err != nil {
		return pricing.ResourcePricing{}, err
	}
	if strings.TrimSpace(cluster.Location) == "" {
		return pricing.ResourcePricing{}, fmt.Errorf("Azure AKS cluster %q has no location", clusterName)
	}
	if len(cluster.NodePools) == 0 {
		return pricing.ResourcePricing{}, fmt.Errorf("Azure AKS cluster %q has no node pools", clusterName)
	}

	var totalNodeCost float64
	var totalCores float64
	var totalMemoryGB float64

	for _, pool := range cluster.NodePools {
		if pool.NodeCount <= 0 {
			continue
		}
		cores, memoryBytes, err := s.sizes.GetSize(ctx, cluster.Location, pool.VMSize)
		if err != nil {
			return pricing.ResourcePricing{}, err
		}
		hourlyPrice, err := s.prices.GetHourlyPrice(ctx, cluster.Location, pool.VMSize)
		if err != nil {
			return pricing.ResourcePricing{}, err
		}

		nodes := float64(pool.NodeCount)
		totalNodeCost += hourlyPrice * nodes
		totalCores += float64(cores) * nodes
		totalMemoryGB += (float64(memoryBytes) / (1024 * 1024 * 1024)) * nodes
	}

	if totalCores <= 0 || totalMemoryGB <= 0 {
		return pricing.ResourcePricing{}, fmt.Errorf("Azure AKS cluster %q has no active capacity for pricing", clusterName)
	}
	cpuPerCoreHour := totalNodeCost / totalCores
	memoryPerGBHour := totalNodeCost / totalMemoryGB
	if cpuPerCoreHour <= 0 || memoryPerGBHour <= 0 {
		return pricing.ResourcePricing{}, fmt.Errorf("Azure AKS cluster %q has invalid normalized pricing", clusterName)
	}

	return pricing.ResourcePricing{
		CPUPerCoreHour:  roundAzurePrice(cpuPerCoreHour),
		MemoryPerGBHour: roundAzurePrice(memoryPerGBHour),
	}, nil
}

func roundAzurePrice(value float64) float64 {
	return math.Round(value*1e8) / 1e8
}

// AzurePricingSource intentionally implements the context-aware pricing
// client contract. The concrete pricing.Provider adapter is Azure Provider.
