package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloud-efficiency-engine/internal/analysis"
)

type mockAnalysisService struct {
	report *analysis.AnalysisReport
	err    error
}

func (m *mockAnalysisService) Analyze(
	ctx context.Context,
	options analysis.AnalysisOptions,
) (*analysis.AnalysisReport, error) {

	return m.report, m.err
}

func TestHandler_Health_ShouldReturn200(
	t *testing.T,
) {

	handler :=
		NewHandler(
			nil,
			NewAnalysisMetrics(),
		)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/health",
			nil,
		)

	recorder :=
		httptest.NewRecorder()

	handler.Health(
		recorder,
		request,
	)

	if recorder.Code != http.StatusOK {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	if !strings.Contains(
		recorder.Body.String(),
		`"status":"UP"`,
	) {

		t.Fatalf(
			"expected UP status, got %s",
			recorder.Body.String(),
		)
	}
}

func TestHandler_Ready_ShouldReturn200(
	t *testing.T,
) {

	handler :=
		NewHandler(
			nil,
			NewAnalysisMetrics(),
		)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/ready",
			nil,
		)

	recorder :=
		httptest.NewRecorder()

	handler.Ready(
		recorder,
		request,
	)

	if recorder.Code != http.StatusOK {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}
}

func TestHandler_Analyze_ShouldRejectInvalidJSON(
	t *testing.T,
) {

	handler :=
		NewHandler(
			nil,
			NewAnalysisMetrics(),
		)

	request :=
		httptest.NewRequest(
			http.MethodPost,
			"/api/v1/analyze",
			strings.NewReader(
				`{"namespace":`,
			),
		)

	recorder :=
		httptest.NewRecorder()

	handler.Analyze(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}

	if !strings.Contains(
		recorder.Body.String(),
		ErrCodeInvalidRequest,
	) {

		t.Fatalf(
			"expected error code %s, got %s",
			ErrCodeInvalidRequest,
			recorder.Body.String(),
		)
	}
}

func TestHandler_Analyze_ShouldRejectEmptyNamespace(
	t *testing.T,
) {

	handler :=
		NewHandler(
			nil,
			NewAnalysisMetrics(),
		)

	request :=
		httptest.NewRequest(
			http.MethodPost,
			"/api/v1/analyze",
			strings.NewReader(
				`{
					"namespace": "",
					"lookbackHours": 168,
					"stepSeconds": 300
				}`,
			),
		)

	recorder :=
		httptest.NewRecorder()

	handler.Analyze(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}

	if !strings.Contains(
		recorder.Body.String(),
		"namespace must not be empty",
	) {

		t.Fatalf(
			"expected namespace validation error, got %s",
			recorder.Body.String(),
		)
	}
}

func TestHandler_Analyze_ShouldRejectWhitespaceNamespace(
	t *testing.T,
) {

	handler :=
		NewHandler(
			nil,
			NewAnalysisMetrics(),
		)

	request :=
		httptest.NewRequest(
			http.MethodPost,
			"/api/v1/analyze",
			strings.NewReader(
				`{
					"namespace": "   ",
					"lookbackHours": 168,
					"stepSeconds": 300
				}`,
			),
		)

	recorder :=
		httptest.NewRecorder()

	handler.Analyze(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestHandler_Analyze_ShouldRejectLookbackBelowMinimum(
	t *testing.T,
) {

	handler :=
		NewHandler(
			nil,
			NewAnalysisMetrics(),
		)

	request :=
		httptest.NewRequest(
			http.MethodPost,
			"/api/v1/analyze",
			strings.NewReader(
				`{
					"namespace": "cloud-efficiency-engine",
					"lookbackHours": -1,
					"stepSeconds": 300
				}`,
			),
		)

	recorder :=
		httptest.NewRecorder()

	handler.Analyze(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestHandler_Analyze_ShouldRejectLookbackAboveMaximum(
	t *testing.T,
) {

	handler :=
		NewHandler(
			nil,
			NewAnalysisMetrics(),
		)

	request :=
		httptest.NewRequest(
			http.MethodPost,
			"/api/v1/analyze",
			strings.NewReader(
				`{
					"namespace": "cloud-efficiency-engine",
					"lookbackHours": 721,
					"stepSeconds": 300
				}`,
			),
		)

	recorder :=
		httptest.NewRecorder()

	handler.Analyze(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestHandler_Analyze_ShouldRejectStepBelowMinimum(
	t *testing.T,
) {

	handler :=
		NewHandler(
			nil,
			NewAnalysisMetrics(),
		)

	request :=
		httptest.NewRequest(
			http.MethodPost,
			"/api/v1/analyze",
			strings.NewReader(
				`{
					"namespace": "cloud-efficiency-engine",
					"lookbackHours": 168,
					"stepSeconds": 14
				}`,
			),
		)

	recorder :=
		httptest.NewRecorder()

	handler.Analyze(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestHandler_Analyze_ShouldRejectStepAboveMaximum(
	t *testing.T,
) {

	handler :=
		NewHandler(
			nil,
			NewAnalysisMetrics(),
		)

	request :=
		httptest.NewRequest(
			http.MethodPost,
			"/api/v1/analyze",
			strings.NewReader(
				`{
					"namespace": "cloud-efficiency-engine",
					"lookbackHours": 168,
					"stepSeconds": 3601
				}`,
			),
		)

	recorder :=
		httptest.NewRecorder()

	handler.Analyze(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestHandler_Analyze_ShouldRejectUnsupportedMethod(
	t *testing.T,
) {

	handler :=
		NewHandler(
			nil,
			NewAnalysisMetrics(),
		)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/api/v1/analyze",
			nil,
		)

	recorder :=
		httptest.NewRecorder()

	handler.Analyze(
		recorder,
		request,
	)

	if recorder.Code !=
		http.StatusMethodNotAllowed {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			recorder.Code,
		)
	}
}

func TestHandler_Analyze_ShouldReturn500WhenEngineFails(
	t *testing.T,
) {

	service :=
		&mockAnalysisService{
			err: context.Canceled,
		}

	analysisMetrics :=
		NewAnalysisMetrics()

	handler :=
		NewHandler(
			service,
			analysisMetrics,
		)

	request :=
		httptest.NewRequest(
			http.MethodPost,
			"/api/v1/analyze",
			strings.NewReader(
				`{
					"namespace": "cloud-efficiency-engine",
					"lookbackHours": 168,
					"stepSeconds": 300
				}`,
			),
		)

	recorder :=
		httptest.NewRecorder()

	handler.Analyze(
		recorder,
		request,
	)

	if recorder.Code !=
		http.StatusInternalServerError {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}

	if !strings.Contains(
		recorder.Body.String(),
		ErrCodeAnalysisFailed,
	) {

		t.Fatalf(
			"expected error code %s, got %s",
			ErrCodeAnalysisFailed,
			recorder.Body.String(),
		)
	}
}

func TestHandler_Analyze_ShouldReturn200(
	t *testing.T,
) {

	service :=
		&mockAnalysisService{
			report: &analysis.AnalysisReport{},
		}

	analysisMetrics :=
		NewAnalysisMetrics()

	handler :=
		NewHandler(
			service,
			analysisMetrics,
		)

	request :=
		httptest.NewRequest(
			http.MethodPost,
			"/api/v1/analyze",
			strings.NewReader(
				`{
					"namespace": "cloud-efficiency-engine",
					"lookbackHours": 168,
					"stepSeconds": 300
				}`,
			),
		)

	recorder :=
		httptest.NewRecorder()

	handler.Analyze(
		recorder,
		request,
	)

	if recorder.Code != http.StatusOK {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	if !strings.Contains(
		recorder.Header().Get("Content-Type"),
		"application/json",
	) {

		t.Fatal(
			"expected application/json content type",
		)
	}
}

func TestRequestIDMiddleware_ShouldGenerateRequestID(
	t *testing.T,
) {

	next :=
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {

				requestID :=
					requestIDFromContext(
						r.Context(),
					)

				if requestID == "" {

					t.Fatal(
						"expected request ID in context",
					)
				}

				w.WriteHeader(
					http.StatusOK,
				)
			},
		)

	handler :=
		requestIDMiddleware(next)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/health",
			nil,
		)

	recorder :=
		httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusOK {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	requestID :=
		recorder.Header().Get(
			"X-Request-ID",
		)

	if requestID == "" {

		t.Fatal(
			"expected X-Request-ID response header",
		)
	}
}

func TestRequestIDMiddleware_ShouldPreserveExistingRequestID(
	t *testing.T,
) {

	expectedRequestID :=
		"test-request-123"

	next :=
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {

				actualRequestID :=
					requestIDFromContext(
						r.Context(),
					)

				if actualRequestID !=
					expectedRequestID {

					t.Fatalf(
						"expected request ID %s, got %s",
						expectedRequestID,
						actualRequestID,
					)
				}

				w.WriteHeader(
					http.StatusOK,
				)
			},
		)

	handler :=
		requestIDMiddleware(next)

	request :=
		httptest.NewRequest(
			http.MethodGet,
			"/health",
			nil,
		)

	request.Header.Set(
		"X-Request-ID",
		expectedRequestID,
	)

	recorder :=
		httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Header().Get(
		"X-Request-ID",
	) != expectedRequestID {

		t.Fatalf(
			"expected response request ID %s, got %s",
			expectedRequestID,
			recorder.Header().Get(
				"X-Request-ID",
			),
		)
	}
}
