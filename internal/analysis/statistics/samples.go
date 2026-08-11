package statistics

import "cloud-efficiency-engine/internal/domain"

func SampleValues(
	samples []domain.MetricSample,
) []float64 {

	values := make(
		[]float64,
		0,
		len(samples),
	)

	for _, sample := range samples {
		values = append(
			values,
			sample.Value,
		)
	}

	return values
}
