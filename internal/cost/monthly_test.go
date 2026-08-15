package cost

import (
	"testing"
	"time"
)

func TestMonthlyizeCost_ShouldConvertWeeklyCost(
	t *testing.T,
) {

	start :=
		time.Date(
			2026,
			8,
			1,
			0,
			0,
			0,
			0,
			time.UTC,
		)

	end :=
		start.Add(
			7 * 24 * time.Hour,
		)

	result, err :=
		MonthlyizeCost(
			100,
			start,
			end,
		)

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	expected :=
		100.0 *
			730.0 /
			168.0

	if result != expected {

		t.Fatalf(
			"expected %f, got %f",
			expected,
			result,
		)
	}
}

func TestMonthlyizeCost_ShouldRejectInvalidPeriod(
	t *testing.T,
) {

	start :=
		time.Now().UTC()

	_, err :=
		MonthlyizeCost(
			100,
			start,
			start,
		)

	if err == nil {

		t.Fatal(
			"expected error",
		)
	}
}
