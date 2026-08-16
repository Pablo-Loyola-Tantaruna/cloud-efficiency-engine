package domain

import "testing"

func TestSafetyScoreForConfidence(t *testing.T) {
	cases := []struct {
		name       string
		confidence Confidence
		expected   int
	}{
		{name: "high", confidence: ConfidenceHigh, expected: 90},
		{name: "medium", confidence: ConfidenceMedium, expected: 70},
		{name: "low", confidence: ConfidenceLow, expected: 40},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SafetyScoreForConfidence(tc.confidence); got != tc.expected {
				t.Fatalf("expected %d, got %d", tc.expected, got)
			}
		})
	}
}
