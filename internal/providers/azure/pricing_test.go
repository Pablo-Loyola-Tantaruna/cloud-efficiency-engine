package azure

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

type retailHTTPMock struct {
	statusCode int
	body       string
	requestURL string
}

func (m *retailHTTPMock) Do(request *http.Request) (*http.Response, error) {
	m.requestURL = request.URL.String()
	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(strings.NewReader(m.body)),
		Header:     make(http.Header),
	}, nil
}

type pricingClusterMock struct {
	cluster AKSCluster
}

func (m *pricingClusterMock) FindCluster(
	ctx context.Context,
	clusterName string,
) (AKSCluster, error) {
	return m.cluster, nil
}

type pricingSizeMock struct {
	cores  int64
	memory int64
}

func (m *pricingSizeMock) GetSize(
	ctx context.Context,
	location string,
	vmSize string,
) (int64, int64, error) {
	return m.cores, m.memory, nil
}

func TestAzureRetailPricesClient_ShouldSelectLinuxConsumptionHourlyPrice(t *testing.T) {
	mock := &retailHTTPMock{
		statusCode: http.StatusOK,
		body: `{
			"Items": [
				{"currencyCode":"USD","retailPrice":0.2,"armRegionName":"eastus","armSkuName":"Standard_D4s_v5","serviceName":"Virtual Machines","type":"Consumption","unitOfMeasure":"1 Hour","isPrimaryMeterRegion":true,"productName":"Virtual Machines Dsv5 Series Windows","meterName":"D4s v5"},
				{"currencyCode":"USD","retailPrice":0.1,"armRegionName":"eastus","armSkuName":"Standard_D4s_v5","serviceName":"Virtual Machines","type":"Consumption","unitOfMeasure":"1 Hour","isPrimaryMeterRegion":true,"productName":"Virtual Machines Dsv5 Series","meterName":"D4s v5"}
			]
		}`,
	}

	client := NewAzureRetailPricesClient(mock)
	price, err := client.GetHourlyPrice(
		context.Background(),
		"eastus",
		"Standard_D4s_v5",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if price != 0.1 {
		t.Fatalf("expected Linux hourly price 0.1, got %f", price)
	}
	if !strings.Contains(mock.requestURL, "serviceName+eq+%27Virtual+Machines%27") {
		t.Fatalf("expected filtered Retail Prices request, got %s", mock.requestURL)
	}
}

func TestAzureRetailPricesClient_ShouldRejectMissingPrice(t *testing.T) {
	mock := &retailHTTPMock{
		statusCode: http.StatusOK,
		body:       `{"Items":[]}`,
	}

	client := NewAzureRetailPricesClient(mock)
	_, err := client.GetHourlyPrice(
		context.Background(),
		"eastus",
		"Standard_D4s_v5",
	)

	if err == nil {
		t.Fatal("expected missing price error")
	}
}

func TestAzurePricingSource_ShouldNormalizeClusterPricing(t *testing.T) {
	httpMock := &retailHTTPMock{
		statusCode: http.StatusOK,
		body:       `{"Items":[{"currencyCode":"USD","retailPrice":0.40,"armRegionName":"eastus","armSkuName":"Standard_D4s_v5","serviceName":"Virtual Machines","type":"Consumption","unitOfMeasure":"1 Hour","isPrimaryMeterRegion":true,"productName":"Virtual Machines Dsv5 Series","meterName":"D4s v5"}]}`,
	}

	source, err := NewAzurePricingSource(
		&pricingClusterMock{
			cluster: AKSCluster{
				Name:     "aks-prod",
				Location: "eastus",
				NodePools: []AKSNodePool{
					{Name: "system", VMSize: "Standard_D4s_v5", NodeCount: 2},
				},
			},
		},
		NewAzureRetailPricesClient(httpMock),
		&pricingSizeMock{
			cores:  4,
			memory: 16 * 1024 * 1024 * 1024,
		},
	)
	if err != nil {
		t.Fatalf("expected no constructor error, got %v", err)
	}

	result, err := source.GetPricing(
		context.Background(),
		domain.AnalysisContext{
			Provider:    domain.CloudProviderAzure,
			ClusterName: "aks-prod",
		},
	)
	if err != nil {
		t.Fatalf("expected no pricing error, got %v", err)
	}

	if result.CPUPerCoreHour != 0.1 {
		t.Fatalf("expected CPU price 0.1, got %f", result.CPUPerCoreHour)
	}
	if result.MemoryPerGBHour != 0.025 {
		t.Fatalf("expected memory price 0.025, got %f", result.MemoryPerGBHour)
	}
}

func TestAzurePricingSource_ShouldValidateClusterContext(t *testing.T) {
	source, err := NewAzurePricingSource(
		&pricingClusterMock{},
		NewAzureRetailPricesClient(&retailHTTPMock{}),
		&pricingSizeMock{cores: 4, memory: 16 * 1024 * 1024 * 1024},
	)
	if err != nil {
		t.Fatalf("expected no constructor error, got %v", err)
	}

	_, err = source.GetPricing(
		context.Background(),
		domain.AnalysisContext{Provider: domain.CloudProviderAzure},
	)
	if err == nil {
		t.Fatal("expected missing cluster validation error")
	}
}
