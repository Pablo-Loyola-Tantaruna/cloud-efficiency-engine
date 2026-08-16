package azure

import (
	"context"
	"fmt"
	"time"

	"cloud-efficiency-engine/internal/domain"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
)

type AzureResource struct {
	ID string

	Namespace string

	Name string

	Type domain.WorkloadType

	CPUCores int64

	CPURequestMillicores int64

	MemoryRequestBytes int64
}

type ResourceInventory interface {
	ListResources(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
	) ([]AzureResource, error)
}

type MonitorMetricsClient interface {
	QueryResource(
		ctx context.Context,
		resourceURI string,
		options *azquery.MetricsClientQueryResourceOptions,
	) (azquery.MetricsClientQueryResourceResponse, error)
}

type AzureMonitorMetricsClient struct {
	monitor   MonitorMetricsClient
	inventory ResourceInventory
}

func NewAzureMonitorMetricsClient(
	monitor MonitorMetricsClient,
	inventory ResourceInventory,
) *AzureMonitorMetricsClient {

	return &AzureMonitorMetricsClient{
		monitor:   monitor,
		inventory: inventory,
	}
}

func (c *AzureMonitorMetricsClient) GetWorkloads(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
) ([]domain.WorkloadMetrics, error) {

	if c == nil || c.monitor == nil {
		return nil, fmt.Errorf("Azure Monitor metrics client is not configured")
	}

	if c.inventory == nil {
		return nil, fmt.Errorf("Azure resource inventory is not configured")
	}

	resources, err := c.inventory.ListResources(
		ctx,
		analysisContext,
	)

	if err != nil {
		return nil, err
	}

	end := time.Now().UTC()
	start := end.Add(-5 * time.Minute)

	result := make(
		[]domain.WorkloadMetrics,
		0,
		len(resources),
	)

	for _, resource := range resources {

		if namespace != "" &&
			resource.Namespace != namespace {
			continue
		}

		cpuUsage, err := c.queryCPU(
			ctx,
			resource.ID,
			resource.CPUCores,
			start,
			end,
			5*time.Minute,
		)

		if err != nil {
			return nil, err
		}

		result = append(
			result,
			domain.WorkloadMetrics{
				Namespace: resource.Namespace,
				Name:      resource.Name,
				Type:      resource.Type,
				Replicas:  1,

				CPURequestMillicores: resource.CPURequestMillicores,
				CPUUsageMillicores:   cpuUsage,
				MemoryRequestBytes:   resource.MemoryRequestBytes,
				MemoryUsageBytes:     0,
			},
		)
	}

	return result, nil
}

func (c *AzureMonitorMetricsClient) GetWorkloadHistory(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {

	if c == nil || c.monitor == nil {
		return nil, fmt.Errorf("Azure Monitor metrics client is not configured")
	}

	if c.inventory == nil {
		return nil, fmt.Errorf("Azure resource inventory is not configured")
	}

	resources, err := c.inventory.ListResources(
		ctx,
		analysisContext,
	)

	if err != nil {
		return nil, err
	}

	result := make(
		[]domain.WorkloadHistory,
		0,
		len(resources),
	)

	for _, resource := range resources {

		if namespace != "" &&
			resource.Namespace != namespace {
			continue
		}

		values, err := c.queryCPUHistory(
			ctx,
			resource.ID,
			resource.CPUCores,
			start,
			end,
			step,
		)

		if err != nil {
			return nil, err
		}

		cpuSamples := make(
			[]domain.MetricSample,
			0,
			len(values),
		)

		for _, value := range values {
			cpuSamples = append(
				cpuSamples,
				domain.MetricSample{
					Timestamp: value.Timestamp,
					Value:     float64(value.Value),
				},
			)
		}

		result = append(
			result,
			domain.WorkloadHistory{
				Namespace:          resource.Namespace,
				Name:               resource.Name,
				CPUUsageMillicores: cpuSamples,
				MemoryUsageBytes:   nil,
			},
		)
	}

	return result, nil
}

type azureMetricPoint struct {
	Timestamp time.Time
	Value     int64
}

func (c *AzureMonitorMetricsClient) queryCPU(
	ctx context.Context,
	resourceURI string,
	cpuCores int64,
	start time.Time,
	end time.Time,
	step time.Duration,
) (int64, error) {

	values, err := c.queryCPUHistory(
		ctx,
		resourceURI,
		cpuCores,
		start,
		end,
		step,
	)

	if err != nil {
		return 0, err
	}

	if len(values) == 0 {
		return 0, nil
	}

	return values[len(values)-1].Value, nil
}

func (c *AzureMonitorMetricsClient) queryCPUHistory(
	ctx context.Context,
	resourceURI string,
	cpuCores int64,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]azureMetricPoint, error) {

	if resourceURI == "" {
		return nil, fmt.Errorf("Azure resource ID must not be empty")
	}

	if cpuCores <= 0 {
		return nil, fmt.Errorf("Azure CPU cores must be greater than zero")
	}

	if !end.After(start) {
		return nil, fmt.Errorf("Azure metrics end time must be after start time")
	}

	if step <= 0 {
		return nil, fmt.Errorf("Azure metrics step must be greater than zero")
	}

	interval := formatAzureInterval(step)

	response, err := c.monitor.QueryResource(
		ctx,
		resourceURI,
		&azquery.MetricsClientQueryResourceOptions{
			Timespan: to.Ptr(
				azquery.NewTimeInterval(
					start.UTC(),
					end.UTC(),
				),
			),
			Interval:    to.Ptr(interval),
			MetricNames: to.Ptr("Percentage CPU"),
			Aggregation: to.SliceOfPtrs(
				azquery.AggregationTypeAverage,
			),
		},
	)

	if err != nil {
		return nil, fmt.Errorf(
			"query Azure Monitor CPU metrics: %w",
			err,
		)
	}

	result := make(
		[]azureMetricPoint,
		0,
	)

	for _, metric := range response.Value {

		if metric == nil || metric.TimeSeries == nil {
			continue
		}

		for _, series := range metric.TimeSeries {

			if series == nil {
				continue
			}

			for _, value := range series.Data {

				if value == nil ||
					value.TimeStamp == nil ||
					value.Average == nil {
					continue
				}

				percentage := *value.Average

				if percentage < 0 {
					percentage = 0
				}

				if percentage > 100 {
					percentage = 100
				}

				millicores := int64(
					percentage / 100 * float64(cpuCores) * 1000,
				)

				result = append(
					result,
					azureMetricPoint{
						Timestamp: value.TimeStamp.UTC(),
						Value:     millicores,
					},
				)
			}
		}
	}

	return result, nil
}

func formatAzureInterval(
	step time.Duration,
) string {

	seconds := int64(
		step / time.Second,
	)

	if seconds%3600 == 0 {
		return fmt.Sprintf(
			"PT%dH",
			seconds/3600,
		)
	}

	if seconds%60 == 0 {
		return fmt.Sprintf(
			"PT%dM",
			seconds/60,
		)
	}

	return fmt.Sprintf(
		"PT%dS",
		seconds,
	)
}
