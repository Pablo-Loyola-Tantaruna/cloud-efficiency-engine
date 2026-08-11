package resolver

import (
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func TestResolver_Resolve(t *testing.T) {

	tests := []struct {
		name            string
		recommendations []domain.Recommendation
		expectedCount   int
		expectedRules   []string
	}{
		{
			name: "should return empty result when there are no recommendations",

			recommendations: nil,

			expectedCount: 0,

			expectedRules: []string{},
		},
		{
			name: "should keep one recommendation per workload and resource",

			recommendations: []domain.Recommendation{
				cpuRecommendation(
					"payments/payments-api",
					"CPU_RULE",
					domain.SeverityWarning,
					domain.ConfidenceMedium,
					1000,
					500,
				),
				cpuRecommendation(
					"payments/payments-api",
					"CPU_HISTORICAL_OPTIMIZATION",
					domain.SeverityWarning,
					domain.ConfidenceHigh,
					1000,
					600,
				),
			},

			expectedCount: 1,

			expectedRules: []string{
				"CPU_HISTORICAL_OPTIMIZATION",
			},
		},
		{
			name: "should prioritize critical severity",

			recommendations: []domain.Recommendation{
				cpuRecommendation(
					"payments/payments-api",
					"LOWER_REDUCTION",
					domain.SeverityWarning,
					domain.ConfidenceHigh,
					1000,
					700,
				),
				cpuRecommendation(
					"payments/payments-api",
					"HIGHER_SEVERITY",
					domain.SeverityCritical,
					domain.ConfidenceLow,
					1000,
					900,
				),
			},

			expectedCount: 1,

			expectedRules: []string{
				"HIGHER_SEVERITY",
			},
		},
		{
			name: "should prioritize higher confidence when severity is equal",

			recommendations: []domain.Recommendation{
				cpuRecommendation(
					"payments/payments-api",
					"LOW_CONFIDENCE",
					domain.SeverityWarning,
					domain.ConfidenceLow,
					1000,
					400,
				),
				cpuRecommendation(
					"payments/payments-api",
					"HIGH_CONFIDENCE",
					domain.SeverityWarning,
					domain.ConfidenceHigh,
					1000,
					700,
				),
			},

			expectedCount: 1,

			expectedRules: []string{
				"HIGH_CONFIDENCE",
			},
		},
		{
			name: "should prioritize higher reduction when severity and confidence are equal",

			recommendations: []domain.Recommendation{
				cpuRecommendation(
					"payments/payments-api",
					"LOWER_REDUCTION",
					domain.SeverityWarning,
					domain.ConfidenceHigh,
					1000,
					700,
				),
				cpuRecommendation(
					"payments/payments-api",
					"HIGHER_REDUCTION",
					domain.SeverityWarning,
					domain.ConfidenceHigh,
					1000,
					500,
				),
			},

			expectedCount: 1,

			expectedRules: []string{
				"HIGHER_REDUCTION",
			},
		},
		{
			name: "should keep cpu and memory recommendations independently",

			recommendations: []domain.Recommendation{
				cpuRecommendation(
					"payments/payments-api",
					"CPU_RULE",
					domain.SeverityWarning,
					domain.ConfidenceHigh,
					1000,
					500,
				),
				memoryRecommendation(
					"payments/payments-api",
					"MEMORY_RULE",
					domain.SeverityWarning,
					domain.ConfidenceHigh,
					1024*1024*1024,
					512*1024*1024,
				),
			},

			expectedCount: 2,

			expectedRules: []string{
				"CPU_RULE",
				"MEMORY_RULE",
			},
		},
		{
			name: "should keep recommendations from different workloads independently",

			recommendations: []domain.Recommendation{
				cpuRecommendation(
					"payments/payments-api",
					"PAYMENTS_CPU",
					domain.SeverityWarning,
					domain.ConfidenceHigh,
					1000,
					500,
				),
				cpuRecommendation(
					"orders/orders-api",
					"ORDERS_CPU",
					domain.SeverityWarning,
					domain.ConfidenceHigh,
					1000,
					500,
				),
			},

			expectedCount: 2,

			expectedRules: []string{
				"PAYMENTS_CPU",
				"ORDERS_CPU",
			},
		},
		{
			name: "should use rule name as deterministic tie breaker",

			recommendations: []domain.Recommendation{
				cpuRecommendation(
					"payments/payments-api",
					"Z_RULE",
					domain.SeverityWarning,
					domain.ConfidenceHigh,
					1000,
					500,
				),
				cpuRecommendation(
					"payments/payments-api",
					"A_RULE",
					domain.SeverityWarning,
					domain.ConfidenceHigh,
					1000,
					500,
				),
			},

			expectedCount: 1,

			expectedRules: []string{
				"A_RULE",
			},
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			// Arrange
			resolver :=
				NewResolver()

			// Act
			result :=
				resolver.Resolve(
					test.recommendations,
				)

			// Assert
			if len(result) !=
				test.expectedCount {

				t.Fatalf(
					"expected %d recommendations, got %d",
					test.expectedCount,
					len(result),
				)
			}

			for index, expectedRule := range test.expectedRules {

				if result[index].Rule !=
					expectedRule {

					t.Fatalf(
						"expected rule %q at index %d, got %q",
						expectedRule,
						index,
						result[index].Rule,
					)
				}
			}
		})
	}
}

func TestResolver_Resolve_ShouldNotMutateInput(
	t *testing.T,
) {

	// Arrange
	input :=
		[]domain.Recommendation{
			cpuRecommendation(
				"payments/payments-api",
				"CPU_RULE",
				domain.SeverityWarning,
				domain.ConfidenceMedium,
				1000,
				500,
			),
			cpuRecommendation(
				"payments/payments-api",
				"CPU_HISTORICAL_OPTIMIZATION",
				domain.SeverityCritical,
				domain.ConfidenceHigh,
				1000,
				400,
			),
		}

	originalFirstRule :=
		input[0].Rule

	resolver :=
		NewResolver()

	// Act
	_ = resolver.Resolve(input)

	// Assert
	if input[0].Rule !=
		originalFirstRule {

		t.Fatalf(
			"resolver mutated the input recommendations",
		)
	}
}

func TestReductionPercentage(t *testing.T) {

	tests := []struct {
		name      string
		current   float64
		suggested float64
		expected  float64
	}{
		{
			name:      "should calculate reduction percentage",
			current:   1000,
			suggested: 500,
			expected:  50,
		},
		{
			name:      "should return zero when suggested is equal",
			current:   1000,
			suggested: 1000,
			expected:  0,
		},
		{
			name:      "should return zero when suggested is greater",
			current:   1000,
			suggested: 1200,
			expected:  0,
		},
		{
			name:      "should return zero when current is zero",
			current:   0,
			suggested: 500,
			expected:  0,
		},
		{
			name:      "should return zero when suggested is zero",
			current:   1000,
			suggested: 0,
			expected:  0,
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			// Arrange
			current :=
				test.current

			suggested :=
				test.suggested

			// Act
			result :=
				calculateReduction(
					current,
					suggested,
				)

			// Assert
			if result !=
				test.expected {

				t.Fatalf(
					"expected %.2f, got %.2f",
					test.expected,
					result,
				)
			}
		})
	}
}

func cpuRecommendation(
	workload string,
	rule string,
	severity domain.Severity,
	confidence domain.Confidence,
	current int64,
	suggested int64,
) domain.Recommendation {

	return domain.Recommendation{
		Rule:        rule,
		Workload:    workload,
		Description: "CPU optimization recommendation",
		Severity:    severity,
		Confidence:  confidence,

		CurrentCPURequestMillicores: current,

		SuggestedCPURequestMillicores: suggested,
	}
}

func memoryRecommendation(
	workload string,
	rule string,
	severity domain.Severity,
	confidence domain.Confidence,
	current int64,
	suggested int64,
) domain.Recommendation {

	return domain.Recommendation{
		Rule:        rule,
		Workload:    workload,
		Description: "Memory optimization recommendation",
		Severity:    severity,
		Confidence:  confidence,

		CurrentMemoryRequestBytes: current,

		SuggestedMemoryRequestBytes: suggested,
	}
}
