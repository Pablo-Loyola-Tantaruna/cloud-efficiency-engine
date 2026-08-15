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

type prometheusResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

type prometheusRangeResult struct {
	Metric map[string]string `json:"metric"`
	Values [][]interface{}   `json:"values"`
}

type prometheusRangeResponse struct {
	Status string `json:"status"`

	Data struct {
		ResultType string                  `json:"resultType"`
		Result     []prometheusRangeResult `json:"result"`
	} `json:"data"`

	ErrorType string `json:"errorType,omitempty"`
	Error     string `json:"error,omitempty"`
}

type prometheusResponse struct {
	Status string `json:"status"`

	Data struct {
		ResultType string             `json:"resultType"`
		Result     []prometheusResult `json:"result"`
	} `json:"data"`

	ErrorType string `json:"errorType,omitempty"`
	Error     string `json:"error,omitempty"`
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

func workloadMetricQuery(
	metric string,
	namespace string,
) string {

	return metric +
		`{namespace="` +
		namespace +
		`"}`
}

func (p *PrometheusProvider) query(
	ctx context.Context,
	query string,
) ([]prometheusResult, error) {

	endpoint, err :=
		url.Parse(
			p.baseURL + "/api/v1/query",
		)

	if err != nil {
		return nil, err
	}

	params :=
		endpoint.Query()

	params.Set(
		"query",
		query,
	)

	endpoint.RawQuery =
		params.Encode()

	req, err :=
		http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			endpoint.String(),
			nil,
		)

	if err != nil {
		return nil, err
	}

	response, err :=
		p.httpClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {

		return nil, fmt.Errorf(
			"prometheus returned HTTP %d",
			response.StatusCode,
		)
	}

	var payload prometheusResponse

	if err :=
		json.NewDecoder(
			response.Body,
		).Decode(&payload); err != nil {

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

func (p *PrometheusProvider) queryRange(
	ctx context.Context,
	query string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]prometheusRangeResult, error) {

	if end.Before(start) ||
		end.Equal(start) {

		return nil, fmt.Errorf(
			"invalid time range",
		)
	}

	if step <= 0 {

		return nil, fmt.Errorf(
			"step must be greater than zero",
		)
	}

	endpoint, err :=
		url.Parse(
			p.baseURL + "/api/v1/query_range",
		)

	if err != nil {
		return nil, err
	}

	params :=
		endpoint.Query()

	params.Set(
		"query",
		query,
	)

	params.Set(
		"start",
		strconv.FormatInt(
			start.Unix(),
			10,
		),
	)

	params.Set(
		"end",
		strconv.FormatInt(
			end.Unix(),
			10,
		),
	)

	params.Set(
		"step",
		strconv.FormatInt(
			int64(step.Seconds()),
			10,
		),
	)

	endpoint.RawQuery =
		params.Encode()

	req, err :=
		http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			endpoint.String(),
			nil,
		)

	if err != nil {
		return nil, err
	}

	response, err :=
		p.httpClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {

		return nil, fmt.Errorf(
			"prometheus returned HTTP %d",
			response.StatusCode,
		)
	}

	var payload prometheusRangeResponse

	if err :=
		json.NewDecoder(
			response.Body,
		).Decode(&payload); err != nil {

		return nil, err
	}

	if payload.Status != "success" {

		return nil, fmt.Errorf(
			"prometheus range query failed: %s",
			payload.Error,
		)
	}

	return payload.Data.Result, nil
}

func parsePrometheusValue(
	value []interface{},
) (float64, error) {

	if len(value) < 2 {

		return 0, fmt.Errorf(
			"invalid prometheus value",
		)
	}

	rawValue, ok :=
		value[1].(string)

	if !ok {

		return 0, fmt.Errorf(
			"invalid prometheus numeric value",
		)
	}

	return strconv.ParseFloat(
		rawValue,
		64,
	)
}

func parsePrometheusRangeValues(
	values [][]interface{},
) ([]domain.MetricSample, error) {

	result :=
		make(
			[]domain.MetricSample,
			0,
			len(values),
		)

	for _, value := range values {

		if len(value) < 2 {

			return nil, fmt.Errorf(
				"invalid prometheus range value",
			)
		}

		rawTimestamp, ok :=
			value[0].(float64)

		if !ok {

			return nil, fmt.Errorf(
				"invalid prometheus timestamp",
			)
		}

		rawValue, ok :=
			value[1].(string)

		if !ok {

			return nil, fmt.Errorf(
				"invalid prometheus range numeric value",
			)
		}

		parsedValue, err :=
			strconv.ParseFloat(
				rawValue,
				64,
			)

		if err != nil {

			return nil, fmt.Errorf(
				"parse prometheus range value: %w",
				err,
			)
		}

		result =
			append(
				result,
				domain.MetricSample{
					Timestamp: time.Unix(
						int64(rawTimestamp),
						0,
					),

					Value: parsedValue,
				},
			)
	}

	return result, nil
}

func (p *PrometheusProvider) GetWorkloads(
	ctx context.Context,
	namespace string,
) ([]domain.WorkloadMetrics, error) {

	cpuRequests, err :=
		p.query(
			ctx,
			workloadMetricQuery(
				"cee_workload_cpu_request_millicores",
				namespace,
			),
		)

	if err != nil {
		return nil, err
	}

	cpuUsage, err :=
		p.query(
			ctx,
			workloadMetricQuery(
				"cee_workload_cpu_usage_millicores",
				namespace,
			),
		)

	if err != nil {
		return nil, err
	}

	memoryRequests, err :=
		p.query(
			ctx,
			workloadMetricQuery(
				"cee_workload_memory_request_bytes",
				namespace,
			),
		)

	if err != nil {
		return nil, err
	}

	memoryUsage, err :=
		p.query(
			ctx,
			workloadMetricQuery(
				"cee_workload_memory_usage_bytes",
				namespace,
			),
		)

	if err != nil {
		return nil, err
	}

	replicas, err :=
		p.query(
			ctx,
			workloadMetricQuery(
				"cee_workload_replicas",
				namespace,
			),
		)

	if err != nil {
		return nil, err
	}

	return mergeMetrics(
		cpuRequests,
		cpuUsage,
		memoryRequests,
		memoryUsage,
		replicas,
	)
}

func (p *PrometheusProvider) GetWorkloadsWithContext(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
) ([]domain.WorkloadMetrics, error) {

	return p.GetWorkloads(
		ctx,
		namespace,
	)
}

func (p *PrometheusProvider) GetWorkloadHistory(
	ctx context.Context,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {

	cpuResults, err :=
		p.queryRange(
			ctx,
			workloadMetricQuery(
				"cee_workload_cpu_usage_millicores",
				namespace,
			),
			start,
			end,
			step,
		)

	if err != nil {
		return nil, err
	}

	memoryResults, err :=
		p.queryRange(
			ctx,
			workloadMetricQuery(
				"cee_workload_memory_usage_bytes",
				namespace,
			),
			start,
			end,
			step,
		)

	if err != nil {
		return nil, err
	}

	return mergeHistoricalMetrics(
		cpuResults,
		memoryResults,
	)
}

func (p *PrometheusProvider) GetWorkloadHistoryWithContext(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {

	return p.GetWorkloadHistory(
		ctx,
		namespace,
		start,
		end,
		step,
	)
}

func (p *PrometheusProvider) QueryInstant(
	ctx context.Context,
	query string,
) ([]map[string]string, []float64, error) {

	results, err :=
		p.query(
			ctx,
			query,
		)

	if err != nil {
		return nil, nil, err
	}

	labels :=
		make(
			[]map[string]string,
			0,
			len(results),
		)

	values :=
		make(
			[]float64,
			0,
			len(results),
		)

	for _, result := range results {

		value, err :=
			parsePrometheusValue(
				result.Value,
			)

		if err != nil {
			return nil, nil, err
		}

		labels =
			append(
				labels,
				result.Metric,
			)

		values =
			append(
				values,
				value,
			)
	}

	return labels, values, nil
}
