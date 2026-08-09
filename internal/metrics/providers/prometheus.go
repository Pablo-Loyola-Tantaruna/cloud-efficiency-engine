package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type PrometheusProvider struct {
	baseURL    string
	httpClient *http.Client
}

func NewPrometheusProvider(
	baseURL string,
	httpClient *http.Client,
) *PrometheusProvider {

	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	return &PrometheusProvider{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

type prometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
	ErrorType string `json:"errorType,omitempty"`
	Error     string `json:"error,omitempty"`
}

func (p *PrometheusProvider) query(
	ctx context.Context,
	query string,
) ([]struct {
	Metric map[string]string
	Value  []interface{}
}, error) {

	endpoint, err := url.Parse(
		p.baseURL + "/api/v1/query",
	)

	if err != nil {
		return nil, err
	}

	params := endpoint.Query()
	params.Set("query", query)
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint.String(),
		nil,
	)

	if err != nil {
		return nil, err
	}

	response, err := p.httpClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"prometheus returned HTTP %d",
			response.StatusCode,
		)
	}

	var payload prometheusResponse

	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}

	if payload.Status != "success" {
		return nil, fmt.Errorf(
			"prometheus query failed: %s",
			payload.Error,
		)
	}

	return payload.Data.Result, nil
}

func parsePrometheusValue(
	value []interface{},
) (float64, error) {

	if len(value) < 2 {
		return 0, fmt.Errorf("invalid prometheus value")
	}

	rawValue, ok := value[1].(string)

	if !ok {
		return 0, fmt.Errorf("invalid prometheus numeric value")
	}

	return strconv.ParseFloat(rawValue, 64)
}

func (p *PrometheusProvider) GetWorkloads(
	ctx context.Context,
) ([]domain.WorkloadMetrics, error) {

	cpuRequests, err := p.query(
		ctx,
		`cee_workload_cpu_request_millicores`,
	)

	if err != nil {
		return nil, err
	}

	cpuUsage, err := p.query(
		ctx,
		`cee_workload_cpu_usage_millicores`,
	)

	if err != nil {
		return nil, err
	}

	memoryRequests, err := p.query(
		ctx,
		`cee_workload_memory_request_bytes`,
	)

	if err != nil {
		return nil, err
	}

	memoryUsage, err := p.query(
		ctx,
		`cee_workload_memory_usage_bytes`,
	)

	if err != nil {
		return nil, err
	}

	return mergeMetrics(
		cpuRequests,
		cpuUsage,
		memoryRequests,
		memoryUsage,
	)
}
