package statistics

import (
	"testing"
)

func TestCalculateResourceStatistics_ShouldCalculateExpectedStatistics(
	t *testing.T,
) {

	// Arrange

	values := []float64{
		100,
		200,
		300,
		400,
		500,
	}

	// Act

	result, err := CalculateResourceStatistics(values)

	// Assert

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if result.P50 != 300 {
		t.Errorf(
			"expected P50 300, got %.2f",
			result.P50,
		)
	}

	if result.P90 != 460 {
		t.Errorf(
			"expected P90 460, got %.2f",
			result.P90,
		)
	}

	if result.P95 != 480 {
		t.Errorf(
			"expected P95 480, got %.2f",
			result.P95,
		)
	}

	if result.P99 != 496 {
		t.Errorf(
			"expected P99 496, got %.2f",
			result.P99,
		)
	}

	if result.Max != 500 {
		t.Errorf(
			"expected max 500, got %.2f",
			result.Max,
		)
	}

	if result.Samples != 5 {
		t.Errorf(
			"expected 5 samples, got %d",
			result.Samples,
		)
	}
}

func TestCalculateResourceStatistics_ShouldReturnErrorForEmptyValues(
	t *testing.T,
) {

	// Arrange

	values := []float64{}

	// Act

	_, err := CalculateResourceStatistics(values)

	// Assert

	if err == nil {
		t.Fatal(
			"expected error for empty values",
		)
	}
}
