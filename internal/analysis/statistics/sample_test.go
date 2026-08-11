package statistics

import (
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

func TestSampleValues_ShouldExtractValues(
	t *testing.T,
) {

	// Arrange

	samples := []domain.MetricSample{
		{
			Timestamp: time.Unix(1000, 0),
			Value:     100,
		},
		{
			Timestamp: time.Unix(1060, 0),
			Value:     200,
		},
		{
			Timestamp: time.Unix(1120, 0),
			Value:     300,
		},
	}

	// Act

	result := SampleValues(samples)

	// Assert

	if len(result) != 3 {
		t.Fatalf(
			"expected 3 values, got %d",
			len(result),
		)
	}

	expected := []float64{
		100,
		200,
		300,
	}

	for index, value := range expected {

		if result[index] != value {
			t.Errorf(
				"expected value %.2f at index %d, got %.2f",
				value,
				index,
				result[index],
			)
		}
	}
}
