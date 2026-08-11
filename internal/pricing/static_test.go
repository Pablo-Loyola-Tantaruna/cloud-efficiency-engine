package pricing

import (
	"context"
	"testing"
)

func TestStaticProvider_GetPricing(t *testing.T) {

	// Arrange
	expected :=
		ResourcePricing{
			CPUPerCoreHour:  0.04,
			MemoryPerGBHour: 0.005,
		}

	provider :=
		NewStaticProvider(
			expected,
		)

	// Act
	result, err :=
		provider.GetPricing(
			context.Background(),
		)

	// Assert
	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if result.CPUPerCoreHour !=
		expected.CPUPerCoreHour {

		t.Fatalf(
			"expected CPU price %.4f, got %.4f",
			expected.CPUPerCoreHour,
			result.CPUPerCoreHour,
		)
	}

	if result.MemoryPerGBHour !=
		expected.MemoryPerGBHour {

		t.Fatalf(
			"expected memory price %.4f, got %.4f",
			expected.MemoryPerGBHour,
			result.MemoryPerGBHour,
		)
	}
}

func TestStaticProvider_GetPricing_ShouldReturnConfiguredPricing(
	t *testing.T,
) {

	// Arrange
	provider :=
		NewStaticProvider(
			ResourcePricing{
				CPUPerCoreHour:  0.10,
				MemoryPerGBHour: 0.02,
			},
		)

	// Act
	result, err :=
		provider.GetPricing(
			context.Background(),
		)

	// Assert
	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	expectedCPU :=
		0.10

	expectedMemory :=
		0.02

	if result.CPUPerCoreHour !=
		expectedCPU {

		t.Fatalf(
			"expected CPU price %.4f, got %.4f",
			expectedCPU,
			result.CPUPerCoreHour,
		)
	}

	if result.MemoryPerGBHour !=
		expectedMemory {

		t.Fatalf(
			"expected memory price %.4f, got %.4f",
			expectedMemory,
			result.MemoryPerGBHour,
		)
	}
}
