package aws

import (
	"context"
	"fmt"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type MetricsSource struct {
	client MetricsClient
}

func NewMetricsSource(
	client MetricsClient,
) *MetricsSource {

	return &MetricsSource{
		client: client,
	}
}

func (s *MetricsSource) GetWorkloads(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
) ([]domain.WorkloadMetrics, error) {

	if s.client == nil {
		return nil,
			fmt.Errorf(
				"AWS metrics client is not configured",
			)
	}

	resources, err :=
		s.client.GetWorkloads(
			ctx,
			analysisContext,
		)

	if err != nil {
		return nil, err
	}

	result :=
		make(
			[]domain.WorkloadMetrics,
			0,
			len(resources),
		)

	for _, resource := range resources {

		if namespace != "" &&
			resource.Namespace != namespace {
			continue
		}

		result =
			append(
				result,
				domain.WorkloadMetrics{
					Namespace: resource.Namespace,

					Name: resource.Name,

					Type: domain.WorkloadType(
						resource.Type,
					),

					Replicas: resource.Replicas,

					CPURequestMillicores: resource.CPURequestMillicores,

					CPUUsageMillicores: resource.CPUUsageMillicores,

					MemoryRequestBytes: resource.MemoryRequestBytes,

					MemoryUsageBytes: resource.MemoryUsageBytes,
				},
			)
	}

	return result, nil
}

func (s *MetricsSource) GetWorkloadHistory(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {

	if s.client == nil {
		return nil,
			fmt.Errorf(
				"AWS metrics client is not configured",
			)
	}

	history, err :=
		s.client.GetWorkloadHistory(
			ctx,
			analysisContext,
		)

	if err != nil {
		return nil, err
	}

	result :=
		make(
			[]domain.WorkloadHistory,
			0,
			len(history),
		)

	for workloadName, samples := range history {

		cpuSamples :=
			make(
				[]domain.MetricSample,
				0,
				len(samples),
			)

		memorySamples :=
			make(
				[]domain.MetricSample,
				0,
				len(samples),
			)

		for _, sample := range samples {

			if sample.Timestamp.Before(start) ||
				sample.Timestamp.After(end) {
				continue
			}

			cpuSamples =
				append(
					cpuSamples,
					domain.MetricSample{
						Timestamp: sample.Timestamp,

						Value: sample.CPUUsageMillicores,
					},
				)

			memorySamples =
				append(
					memorySamples,
					domain.MetricSample{
						Timestamp: sample.Timestamp,

						Value: sample.MemoryUsageBytes,
					},
				)
		}

		result =
			append(
				result,
				domain.WorkloadHistory{
					Namespace: namespace,

					Name: workloadName,

					CPUUsageMillicores: cpuSamples,

					MemoryUsageBytes: memorySamples,
				},
			)
	}

	return result, nil
}
