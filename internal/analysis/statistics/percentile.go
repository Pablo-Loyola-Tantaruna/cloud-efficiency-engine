package statistics

import (
	"fmt"
	"sort"
)

func Percentile(
	values []float64,
	percentile float64,
) (float64, error) {

	if len(values) == 0 {
		return 0, fmt.Errorf(
			"cannot calculate percentile from empty values",
		)
	}

	if percentile < 0 || percentile > 100 {
		return 0, fmt.Errorf(
			"percentile must be between 0 and 100",
		)
	}

	sorted := append(
		[]float64(nil),
		values...,
	)

	sort.Float64s(sorted)

	if len(sorted) == 1 {
		return sorted[0], nil
	}

	position :=
		(percentile / 100) *
			float64(len(sorted)-1)

	lower := int(position)

	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[lower], nil
	}

	weight := position -
		float64(lower)

	return sorted[lower] +
		(sorted[upper]-sorted[lower])*weight, nil
}
