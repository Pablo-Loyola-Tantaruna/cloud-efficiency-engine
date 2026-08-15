package cost

import (
	"fmt"
	"time"
)

func MonthlyizeCost(
	totalUSD float64,
	start time.Time,
	end time.Time,
) (float64, error) {

	if totalUSD < 0 {

		return 0,
			fmt.Errorf(
				"total cost must not be negative",
			)
	}

	if end.Before(start) {

		return 0,
			fmt.Errorf(
				"cost period end must not be before start",
			)
	}

	duration :=
		end.Sub(start)

	if duration <= 0 {

		return 0,
			fmt.Errorf(
				"cost period must be greater than zero",
			)
	}

	const hoursPerMonth = 730.0

	hours :=
		duration.Hours()

	return totalUSD *
		hoursPerMonth /
		hours, nil
}
