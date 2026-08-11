package statistics

import (
	"testing"
)

func TestPercentile_ShouldCalculateExpectedValue(t *testing.T) {

	tests := []struct {
		name       string
		values     []float64
		percentile float64
		expected   float64
	}{
		{
			name:       "P50",
			values:     []float64{100, 200, 300, 400, 500},
			percentile: 50,
			expected:   300,
		},
		{
			name:       "P90",
			values:     []float64{100, 200, 300, 400, 500},
			percentile: 90,
			expected:   460,
		},
		{
			name:       "P95",
			values:     []float64{100, 200, 300, 400, 500},
			percentile: 95,
			expected:   480,
		},
		{
			name:       "P99",
			values:     []float64{100, 200, 300, 400, 500},
			percentile: 99,
			expected:   496,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			// Act

			result, err := Percentile(
				tt.values,
				tt.percentile,
			)

			// Assert

			if err != nil {
				t.Fatalf(
					"expected no error, got %v",
					err,
				)
			}

			if result != tt.expected {

				t.Errorf(
					"expected %.2f, got %.2f",
					tt.expected,
					result,
				)
			}
		})
	}
}

func TestPercentile_ShouldReturnErrorForEmptyValues(t *testing.T) {

	_, err := Percentile(
		[]float64{},
		95,
	)

	if err == nil {
		t.Fatal(
			"expected error for empty values",
		)
	}
}

func TestPercentile_ShouldReturnErrorForInvalidPercentile(t *testing.T) {

	_, err := Percentile(
		[]float64{100, 200, 300},
		101,
	)

	if err == nil {
		t.Fatal(
			"expected error for percentile greater than 100",
		)
	}
}
