package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnalysisMetrics_RecordAnalysis_ShouldExposeMetrics(
	t *testing.T,
) {
	// Arrange

	metrics :=
		NewAnalysisMetrics()

	metrics.RecordAnalysis(
		5,
		3,
		500,
		350,
		150,
	)

	metrics.RecordAnalysis(
		2,
		1,
		200,
		150,
		50,
	)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/metrics",
			nil,
		)

	responseRecorder :=
		httptest.NewRecorder()

	// Act

	metrics.Handler(
		responseRecorder,
		request,
	)

	// Assert

	if responseRecorder.Code !=
		http.StatusOK {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			responseRecorder.Code,
		)
	}

	body :=
		responseRecorder.Body.String()

	expectedMetrics :=
		[]string{
			`cee_analysis_total 2`,
			`cee_analysis_errors_total 0`,
			`cee_workloads_analyzed_total 7`,
			`cee_optimizable_workloads_total 4`,
			`cee_current_monthly_cost_usd 200.00`,
			`cee_optimized_monthly_cost_usd 150.00`,
			`cee_potential_savings_usd 50.00`,
			`cee_savings_percentage 25.00`,
		}

	for _, expected := range expectedMetrics {

		if !strings.Contains(
			body,
			expected,
		) {

			t.Fatalf(
				"expected metric %q in output:\n%s",
				expected,
				body,
			)
		}
	}
}

func TestAnalysisMetrics_RecordAnalysisError_ShouldIncrementCounter(
	t *testing.T,
) {
	// Arrange

	metrics :=
		NewAnalysisMetrics()

	// Act

	metrics.RecordAnalysisError()
	metrics.RecordAnalysisError()

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/metrics",
			nil,
		)

	responseRecorder :=
		httptest.NewRecorder()

	metrics.Handler(
		responseRecorder,
		request,
	)

	// Assert

	body :=
		responseRecorder.Body.String()

	expected :=
		`cee_analysis_errors_total 2`

	if !strings.Contains(
		body,
		expected,
	) {

		t.Fatalf(
			"expected metric %q in output:\n%s",
			expected,
			body,
		)
	}
}

func TestAnalysisMetrics_RecordAnalysis_ShouldReplaceCurrentCost(
	t *testing.T,
) {
	// Arrange

	metrics :=
		NewAnalysisMetrics()

	metrics.RecordAnalysis(
		10,
		5,
		1000,
		700,
		300,
	)

	// Act

	metrics.RecordAnalysis(
		10,
		2,
		500,
		450,
		50,
	)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/metrics",
			nil,
		)

	responseRecorder :=
		httptest.NewRecorder()

	metrics.Handler(
		responseRecorder,
		request,
	)

	// Assert

	body :=
		responseRecorder.Body.String()

	expectedMetrics :=
		[]string{
			`cee_current_monthly_cost_usd 500.00`,
			`cee_optimized_monthly_cost_usd 450.00`,
			`cee_potential_savings_usd 50.00`,
			`cee_savings_percentage 10.00`,
		}

	for _, expected := range expectedMetrics {

		if !strings.Contains(
			body,
			expected,
		) {

			t.Fatalf(
				"expected metric %q in output:\n%s",
				expected,
				body,
			)
		}
	}
}
