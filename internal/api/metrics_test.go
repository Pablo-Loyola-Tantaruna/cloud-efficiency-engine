package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPMetrics_RecordRequest_ShouldExposePrometheusMetrics(
	t *testing.T,
) {

	// Arrange

	metrics :=
		NewHTTPMetrics()

	metrics.RecordRequest(
		http.MethodGet,
		"/health",
		http.StatusOK,
		100*time.Millisecond,
	)

	metrics.RecordRequest(
		http.MethodGet,
		"/health",
		http.StatusOK,
		200*time.Millisecond,
	)

	metrics.RecordRequest(
		http.MethodPost,
		"/api/v1/analyze",
		http.StatusInternalServerError,
		500*time.Millisecond,
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
			"# HELP cee_http_requests_total",
			"# TYPE cee_http_requests_total counter",
			`cee_http_requests_total{method="GET",path="/health",status="200"} 2`,
			`cee_http_requests_total{method="POST",path="/api/v1/analyze",status="500"} 1`,
			"cee_http_request_duration_seconds_sum",
			"cee_http_request_duration_seconds_count 3",
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

func TestHTTPMetrics_ShouldRecordMiddlewareRequest(
	t *testing.T,
) {

	// Arrange

	metrics :=
		NewHTTPMetrics()

	handler :=
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {

				w.WriteHeader(
					http.StatusCreated,
				)
			},
		)

	middleware :=
		metricsMiddleware(
			metrics,
			handler,
		)

	request :=
		httptest.NewRequest(
			http.MethodPost,
			"/api/v1/analyze",
			nil,
		)

	responseRecorder :=
		httptest.NewRecorder()

	// Act

	middleware.ServeHTTP(
		responseRecorder,
		request,
	)

	// Assert

	if responseRecorder.Code !=
		http.StatusCreated {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			responseRecorder.Code,
		)
	}

	metricsRequest :=
		httptest.NewRequest(
			http.MethodGet,
			"/metrics",
			nil,
		)

	metricsResponse :=
		httptest.NewRecorder()

	metrics.Handler(
		metricsResponse,
		metricsRequest,
	)

	body :=
		metricsResponse.Body.String()

	expected :=
		`cee_http_requests_total{method="POST",path="/api/v1/analyze",status="201"} 1`

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

func TestHTTPMetrics_ShouldRecordDefaultStatusAsOK(
	t *testing.T,
) {

	// Arrange

	metrics :=
		NewHTTPMetrics()

	handler :=
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {

				_, _ =
					w.Write(
						[]byte("ok"),
					)
			},
		)

	middleware :=
		metricsMiddleware(
			metrics,
			handler,
		)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/health",
			nil,
		)

	responseRecorder :=
		httptest.NewRecorder()

	// Act

	middleware.ServeHTTP(
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
}
