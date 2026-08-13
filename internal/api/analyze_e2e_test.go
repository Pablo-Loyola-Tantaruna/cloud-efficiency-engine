package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/analysis"
	"cloud-efficiency-engine/internal/analysis/optimizer"
	"cloud-efficiency-engine/internal/analysis/resolver"
	"cloud-efficiency-engine/internal/analysis/rules"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics/providers"
	"cloud-efficiency-engine/internal/pricing"
)

func TestAPI_Analyze_EndToEnd_ShouldReturnAnalysisReport(
	t *testing.T,
) {

	// Arrange

	namespace :=
		"cloud-efficiency-engine"

	workload :=
		domain.WorkloadMetrics{
			Namespace: namespace,
			Name:      "payments-api",

			Type: domain.WorkloadDeployment,

			Replicas: 3,

			CPURequestMillicores: 1000,
			CPUUsageMillicores:   200,

			MemoryRequestBytes: 2 *
				1024 *
				1024 *
				1024,

			MemoryUsageBytes: 700 *
				1024 *
				1024,
		}

	metricsProvider :=
		providers.NewMockProvider(
			[]domain.WorkloadMetrics{
				workload,
			},
		)

	now :=
		time.Now().UTC()

	historicalProvider :=
		providers.NewMockHistoricalProvider(
			[]domain.WorkloadHistory{
				{
					Namespace: namespace,
					Name:      "payments-api",

					CPUUsageMillicores: buildAPIHistory(
						now,
						200,
						false,
					),

					MemoryUsageBytes: buildAPIHistory(
						now,
						700*
							1024*
							1024,
						true,
					),
				},
			},
		)

	pricingProvider :=
		pricing.NewStaticProvider(
			pricing.ResourcePricing{
				CPUPerCoreHour:  0.04,
				MemoryPerGBHour: 0.005,
			},
		)

	calculator :=
		cost.NewCalculator(
			730,
		)

	engine :=
		analysis.NewEngine(
			metricsProvider,
			historicalProvider,
			pricingProvider,

			[]rules.Rule{
				rules.NewCPUOverprovisioningRule(),
				rules.NewMemoryOverprovisioningRule(),
			},

			optimizer.DefaultOptimizationPolicy(),

			resolver.NewResolver(),

			calculator,
		)

	analysisMetrics :=
		NewAnalysisMetrics()

	handler :=
		NewHandler(
			engine,
			analysisMetrics,
		)

	router :=
		NewRouter(
			handler,
			NewLogger(),
			NewHTTPMetrics(),
			NewAnalysisMetrics(),
		)

	server :=
		httptest.NewServer(router)

	defer server.Close()

	// Act

	body :=
		`{
			"namespace": "cloud-efficiency-engine",
			"lookbackHours": 24,
			"stepSeconds": 300
		}`

	request, err :=
		http.NewRequest(
			http.MethodPost,
			server.URL+"/api/v1/analyze",
			strings.NewReader(body),
		)

	if err != nil {
		t.Fatalf(
			"failed to create request: %v",
			err,
		)
	}

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	request.Header.Set(
		"X-Request-ID",
		"e2e-test-123",
	)

	response, err :=
		server.Client().Do(request)

	if err != nil {
		t.Fatalf(
			"request failed: %v",
			err,
		)
	}

	defer response.Body.Close()

	// Assert

	if response.StatusCode !=
		http.StatusOK {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			response.StatusCode,
		)
	}

	if response.Header.Get(
		"X-Request-ID",
	) != "e2e-test-123" {

		t.Fatalf(
			"expected request ID e2e-test-123, got %s",
			response.Header.Get(
				"X-Request-ID",
			),
		)
	}

	if !strings.Contains(
		response.Header.Get("Content-Type"),
		"application/json",
	) {

		t.Fatalf(
			"expected JSON response",
		)
	}

	var report analysis.AnalysisReport

	if err :=
		json.NewDecoder(
			response.Body,
		).Decode(&report); err != nil {

		t.Fatalf(
			"failed to decode analysis report: %v",
			err,
		)
	}

	if report.GeneratedAt.IsZero() {

		t.Fatal(
			"expected generatedAt",
		)
	}

	if report.Summary.TotalWorkloads != 1 {

		t.Fatalf(
			"expected 1 workload, got %d",
			report.Summary.TotalWorkloads,
		)
	}

	if len(report.Workloads) != 1 {

		t.Fatalf(
			"expected 1 workload analysis, got %d",
			len(report.Workloads),
		)
	}

	workloadAnalysis :=
		report.Workloads[0]

	if workloadAnalysis.Workload.Namespace !=
		namespace {

		t.Fatalf(
			"expected namespace %s, got %s",
			namespace,
			workloadAnalysis.Workload.Namespace,
		)
	}

	if workloadAnalysis.Workload.Name !=
		"payments-api" {

		t.Fatalf(
			"expected payments-api, got %s",
			workloadAnalysis.Workload.Name,
		)
	}

	if len(
		workloadAnalysis.Recommendations,
	) == 0 {

		t.Fatal(
			"expected at least one recommendation",
		)
	}

	if workloadAnalysis.Cost == nil {

		t.Fatal(
			"expected cost estimate",
		)
	}

	if workloadAnalysis.Cost.
		CurrentMonthlyCostUSD <= 0 {

		t.Fatal(
			"expected current monthly cost",
		)
	}

	if workloadAnalysis.Cost.
		PotentialSavingsUSD <= 0 {

		t.Fatal(
			"expected potential savings",
		)
	}

	if report.Summary.
		PotentialSavingsUSD <= 0 {

		t.Fatal(
			"expected report potential savings",
		)
	}
}

func TestAPI_Analyze_ShouldRejectInvalidRequest(
	t *testing.T,
) {

	// Arrange

	handler :=
		NewHandler(
			nil,
			NewAnalysisMetrics(),
		)

	router :=
		NewRouter(
			handler,
			NewLogger(),
			NewHTTPMetrics(),
			NewAnalysisMetrics(),
		)

	server :=
		httptest.NewServer(router)

	defer server.Close()

	request, err :=
		http.NewRequest(
			http.MethodPost,
			server.URL+"/api/v1/analyze",
			strings.NewReader(
				`{
					"namespace": "",
					"lookbackHours": 168,
					"stepSeconds": 300
				}`,
			),
		)

	if err != nil {
		t.Fatalf(
			"failed to create request: %v",
			err,
		)
	}

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	// Act

	response, err :=
		server.Client().Do(request)

	if err != nil {
		t.Fatalf(
			"request failed: %v",
			err,
		)
	}

	defer response.Body.Close()

	// Assert

	if response.StatusCode !=
		http.StatusBadRequest {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.StatusCode,
		)
	}
}

func TestAPI_Analyze_ShouldRejectUnsupportedMethod(
	t *testing.T,
) {

	// Arrange

	handler :=
		NewHandler(
			nil,
			NewAnalysisMetrics(),
		)

	router :=
		NewRouter(
			handler,
			NewLogger(),
			NewHTTPMetrics(),
			NewAnalysisMetrics(),
		)

	server :=
		httptest.NewServer(router)

	defer server.Close()

	request, err :=
		http.NewRequest(
			http.MethodGet,
			server.URL+"/api/v1/analyze",
			nil,
		)

	if err != nil {
		t.Fatalf(
			"failed to create request: %v",
			err,
		)
	}

	// Act

	response, err :=
		server.Client().Do(request)

	if err != nil {
		t.Fatalf(
			"request failed: %v",
			err,
		)
	}

	defer response.Body.Close()

	// Assert

	if response.StatusCode !=
		http.StatusMethodNotAllowed {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			response.StatusCode,
		)
	}
}

func TestAPI_Health_ShouldReturnUP(
	t *testing.T,
) {

	// Arrange

	handler :=
		NewHandler(
			nil,
			NewAnalysisMetrics(),
		)

	router :=
		NewRouter(
			handler,
			NewLogger(),
			NewHTTPMetrics(),
			NewAnalysisMetrics(),
		)

	server :=
		httptest.NewServer(router)

	defer server.Close()

	// Act

	response, err :=
		server.Client().Get(
			server.URL + "/health",
		)

	if err != nil {
		t.Fatalf(
			"request failed: %v",
			err,
		)
	}

	defer response.Body.Close()

	// Assert

	if response.StatusCode !=
		http.StatusOK {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			response.StatusCode,
		)
	}
}

func TestAPI_Ready_ShouldReturnUP(
	t *testing.T,
) {

	// Arrange

	handler :=
		NewHandler(
			nil,
			NewAnalysisMetrics(),
		)

	router :=
		NewRouter(
			handler,
			NewLogger(),
			NewHTTPMetrics(),
			NewAnalysisMetrics(),
		)

	server :=
		httptest.NewServer(router)

	defer server.Close()

	// Act

	response, err :=
		server.Client().Get(
			server.URL + "/ready",
		)

	if err != nil {
		t.Fatalf(
			"request failed: %v",
			err,
		)
	}

	defer response.Body.Close()

	// Assert

	if response.StatusCode !=
		http.StatusOK {

		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			response.StatusCode,
		)
	}
}

func buildAPIHistory(
	now time.Time,
	base float64,
	memory bool,
) []domain.MetricSample {

	result :=
		make(
			[]domain.MetricSample,
			0,
			288,
		)

	values :=
		[]float64{
			0.80,
			0.85,
			0.90,
			0.95,
			1.00,
		}

	for i := 0; i < 288; i++ {

		timestamp :=
			now.Add(
				-time.Duration(
					288-i,
				) * 5 * time.Minute,
			)

		value :=
			base *
				values[i%len(values)]

		result =
			append(
				result,
				domain.MetricSample{
					Timestamp: timestamp,
					Value:     value,
				},
			)
	}

	return result
}
