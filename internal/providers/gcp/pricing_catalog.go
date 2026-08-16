package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/pricing"
)

const gcpCloudBillingCatalogEndpoint = "https://cloudbilling.googleapis.com/v1"

var gcpPricingCPUDescription = regexp.MustCompile(`(?i)(cpu|core).*?(running|instance|compute)`)
var gcpPricingMemoryDescription = regexp.MustCompile(`(?i)(ram|memory).*?(running|instance|compute)`)

type GCPCatalogHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type GCPCatalogPricingClient struct {
	httpClient GCPCatalogHTTPClient
	apiKey     string
	endpoint   string
}

func NewGCPCatalogPricingClient(httpClient GCPCatalogHTTPClient, apiKey string) (*GCPCatalogPricingClient, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("GCP pricing HTTP client must not be nil")
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("GCP Cloud Billing API key must not be empty")
	}
	return &GCPCatalogPricingClient{httpClient: httpClient, apiKey: apiKey, endpoint: gcpCloudBillingCatalogEndpoint}, nil
}

func (c *GCPCatalogPricingClient) GetPricing(ctx context.Context, analysisContext domain.AnalysisContext) (pricing.ResourcePricing, error) {
	if c == nil || c.httpClient == nil {
		return pricing.ResourcePricing{}, fmt.Errorf("GCP catalog pricing client is not configured")
	}
	region := strings.TrimSpace(analysisContext.Region)
	if region == "" {
		return pricing.ResourcePricing{}, fmt.Errorf("GCP pricing region must not be empty")
	}

	serviceID, err := c.findComputeService(ctx)
	if err != nil {
		return pricing.ResourcePricing{}, err
	}
	cpu, err := c.findSKUPrice(ctx, serviceID, region, gcpPricingCPUDescription)
	if err != nil {
		return pricing.ResourcePricing{}, err
	}
	memory, err := c.findSKUPrice(ctx, serviceID, region, gcpPricingMemoryDescription)
	if err != nil {
		return pricing.ResourcePricing{}, err
	}
	return pricing.ResourcePricing{CPUPerCoreHour: cpu, MemoryPerGBHour: memory}, nil
}

type gcpServiceListResponse struct {
	Services []struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		ServiceID   string `json:"serviceId"`
	} `json:"services"`
	NextPageToken string `json:"nextPageToken"`
}

func (c *GCPCatalogPricingClient) findComputeService(ctx context.Context) (string, error) {
	pageToken := ""
	for {
		queryURL, err := url.Parse(c.endpoint + "/services")
		if err != nil {
			return "", err
		}
		params := queryURL.Query()
		params.Set("key", c.apiKey)
		if pageToken != "" {
			params.Set("pageToken", pageToken)
		}
		queryURL.RawQuery = params.Encode()

		var payload gcpServiceListResponse
		if err := c.doJSON(ctx, queryURL.String(), &payload); err != nil {
			return "", err
		}
		for _, service := range payload.Services {
			if strings.EqualFold(service.DisplayName, "Compute Engine") || strings.Contains(strings.ToLower(service.DisplayName), "compute engine") {
				return service.Name, nil
			}
		}
		if payload.NextPageToken == "" {
			break
		}
		pageToken = payload.NextPageToken
	}
	return "", fmt.Errorf("GCP Compute Engine billing service not found")
}

type gcpSKUListResponse struct {
	Skus          []gcpSKU `json:"skus"`
	NextPageToken string   `json:"nextPageToken"`
}

type gcpSKU struct {
	Name           string           `json:"name"`
	Description    string           `json:"description"`
	ServiceRegions []string         `json:"serviceRegions"`
	PricingInfo    []gcpPricingInfo `json:"pricingInfo"`
}

type gcpPricingInfo struct {
	PricingExpression struct {
		UsageUnit   string `json:"usageUnit"`
		TieredRates []struct {
			UnitPrice struct {
				CurrencyCode string `json:"currencyCode"`
				Units        int64  `json:"units,string"`
				Nanos        int64  `json:"nanos"`
			} `json:"unitPrice"`
		} `json:"tieredRates"`
	} `json:"pricingExpression"`
}

func (c *GCPCatalogPricingClient) findSKUPrice(ctx context.Context, serviceName, region string, description *regexp.Regexp) (float64, error) {
	serviceName = strings.TrimPrefix(serviceName, "services/")
	pageToken := ""
	for {
		queryURL, err := url.Parse(c.endpoint + "/services/" + url.PathEscape(serviceName) + "/skus")
		if err != nil {
			return 0, err
		}
		params := queryURL.Query()
		params.Set("key", c.apiKey)
		params.Set("pageSize", "5000")
		if pageToken != "" {
			params.Set("pageToken", pageToken)
		}
		queryURL.RawQuery = params.Encode()

		var payload gcpSKUListResponse
		if err := c.doJSON(ctx, queryURL.String(), &payload); err != nil {
			return 0, err
		}
		for _, sku := range payload.Skus {
			if !description.MatchString(sku.Description) || !gcpRegionMatches(sku.ServiceRegions, region) {
				continue
			}
			if len(sku.PricingInfo) == 0 || len(sku.PricingInfo[0].PricingExpression.TieredRates) == 0 {
				continue
			}
			rate := sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice
			if !strings.EqualFold(rate.CurrencyCode, "USD") {
				continue
			}
			amount := float64(rate.Units) + float64(rate.Nanos)/1e9
			if amount <= 0 {
				continue
			}
			unit := strings.ToLower(sku.PricingInfo[0].PricingExpression.UsageUnit)
			if strings.Contains(unit, "second") {
				amount *= 3600
			}
			if strings.Contains(unit, "minute") {
				amount *= 60
			}
			if strings.Contains(unit, "hour") || strings.Contains(unit, "second") || strings.Contains(unit, "minute") {
				if gcpPricingMemoryDescription.MatchString(sku.Description) {
					if !strings.Contains(unit, "gib") && !strings.Contains(unit, "gbyte") && !strings.Contains(unit, "byte") {
						continue
					}
					return amount, nil
				}
				if gcpPricingCPUDescription.MatchString(sku.Description) {
					return amount, nil
				}
			}
		}
		if payload.NextPageToken == "" {
			break
		}
		pageToken = payload.NextPageToken
	}
	return 0, fmt.Errorf("GCP pricing SKU not found for region %q", region)
}

func gcpRegionMatches(regions []string, region string) bool {
	if len(regions) == 0 {
		return true
	}
	for _, value := range regions {
		if value == "global" || strings.EqualFold(value, region) || strings.HasPrefix(region, value+"-") {
			return true
		}
	}
	return false
}

func (c *GCPCatalogPricingClient) doJSON(ctx context.Context, rawURL string, target interface{}) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("GCP Cloud Billing API returned HTTP %d", response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

var _ PricingClient = (*GCPCatalogPricingClient)(nil)
