package statistics

import (
	"fmt"
)

type ResourceStatistics struct {
	P50 float64
	P90 float64
	P95 float64
	P99 float64
	Max float64

	Samples int
}

func CalculateResourceStatistics(
	values []float64,
) (ResourceStatistics, error) {

	if len(values) == 0 {
		return ResourceStatistics{}, fmt.Errorf(
			"cannot calculate resource statistics from empty values",
		)
	}

	p50, err := Percentile(values, 50)
	if err != nil {
		return ResourceStatistics{}, err
	}

	p90, err := Percentile(values, 90)
	if err != nil {
		return ResourceStatistics{}, err
	}

	p95, err := Percentile(values, 95)
	if err != nil {
		return ResourceStatistics{}, err
	}

	p99, err := Percentile(values, 99)
	if err != nil {
		return ResourceStatistics{}, err
	}

	max := values[0]

	for _, value := range values {
		if value > max {
			max = value
		}
	}

	return ResourceStatistics{
		P50:     p50,
		P90:     p90,
		P95:     p95,
		P99:     p99,
		Max:     max,
		Samples: len(values),
	}, nil
}
