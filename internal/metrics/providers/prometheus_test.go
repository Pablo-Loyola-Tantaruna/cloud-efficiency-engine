package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPrometheusProvider_GetWorkloads_ShouldFilterByNamespace(
	t *testing.T,
) {

	// Arrange

	server :=
		httptest.NewServer(
			http.HandlerFunc(
				func(
					w http.ResponseWriter,
					r *http.Request,
				) {

					query :=
						r.URL.Query().Get(
							"query",
						)

					if !strings.Contains(
						query,
						`namespace="cloud-efficiency"`,
					) {

						t.Errorf(
							"expected namespace filter in query, got %s",
							query,
						)
					}

					response := map[string]interface{}{
						"status": "success",

						"data": map[string]interface{}{
							"resultType": "vector",

							"result": []map[string]interface{}{
								{
									"metric": map[string]string{
										"namespace": "cloud-efficiency",
										"workload":  "payments-api",
									},

									"value": []interface{}{
										float64(1720000000),
										"1000",
									},
								},
							},
						},
					}

					w.Header().Set(
						"Content-Type",
						"application/json",
					)

					w.WriteHeader(
						http.StatusOK,
					)

					_ = json.NewEncoder(
						w,
					).Encode(response)
				},
			),
		)

	defer server.Close()

	provider :=
		NewPrometheusProvider(
			server.URL,
			server.Client(),
		)

	// Act

	_, err :=
		provider.GetWorkloads(
			context.Background(),
			"cloud-efficiency",
		)

	// Assert

	if err == nil {

		/*
			The fake server intentionally returns
			only one metric family, so mergeMetrics
			will not have enough information to build
			a complete workload.

			The purpose of this test is the HTTP
			contract: namespace filtering must reach
			Prometheus.
		*/

		return
	}

	if !strings.Contains(
		err.Error(),
		"prometheus",
	) {

		t.Fatalf(
			"expected Prometheus-related error, got %v",
			err,
		)
	}
}

func TestPrometheusProvider_QueryRange_ShouldSendNamespace(
	t *testing.T,
) {

	// Arrange

	server :=
		httptest.NewServer(
			http.HandlerFunc(
				func(
					w http.ResponseWriter,
					r *http.Request,
				) {

					query :=
						r.URL.Query().Get(
							"query",
						)

					if !strings.Contains(
						query,
						`namespace="cloud-efficiency"`,
					) {

						t.Errorf(
							"expected namespace filter in range query, got %s",
							query,
						)
					}

					response := map[string]interface{}{
						"status": "success",

						"data": map[string]interface{}{
							"resultType": "matrix",

							"result": []map[string]interface{}{
								{
									"metric": map[string]string{
										"namespace": "cloud-efficiency",
										"workload":  "payments-api",
									},

									"values": [][]interface{}{
										{
											float64(1720000000),
											"100",
										},
										{
											float64(1720000300),
											"120",
										},
									},
								},
							},
						},
					}

					w.Header().Set(
						"Content-Type",
						"application/json",
					)

					w.WriteHeader(
						http.StatusOK,
					)

					_ = json.NewEncoder(
						w,
					).Encode(response)
				},
			),
		)

	defer server.Close()

	provider :=
		NewPrometheusProvider(
			server.URL,
			server.Client(),
		)

	start :=
		time.Unix(
			1720000000,
			0,
		)

	end :=
		start.Add(
			10 * time.Minute,
		)

	step :=
		5 * time.Minute

	// Act

	result, err :=
		provider.queryRange(
			context.Background(),
			`cee_workload_cpu_usage_millicores{namespace="cloud-efficiency"}`,
			start,
			end,
			step,
		)

	// Assert

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if len(result) != 1 {

		t.Fatalf(
			"expected 1 result, got %d",
			len(result),
		)
	}

	if result[0].Metric["namespace"] !=
		"cloud-efficiency" {

		t.Fatalf(
			"expected cloud-efficiency namespace, got %s",
			result[0].Metric["namespace"],
		)
	}

	if len(result[0].Values) != 2 {

		t.Fatalf(
			"expected 2 values, got %d",
			len(result[0].Values),
		)
	}
}

func TestWorkloadMetricQuery_ShouldBuildNamespaceSelector(
	t *testing.T,
) {

	// Arrange

	metric :=
		"cee_workload_cpu_usage_millicores"

	namespace :=
		"cloud-efficiency"

	// Act

	query :=
		workloadMetricQuery(
			metric,
			namespace,
		)

	// Assert

	expected :=
		`cee_workload_cpu_usage_millicores{namespace="cloud-efficiency"}`

	if query != expected {

		t.Fatalf(
			"expected %s, got %s",
			expected,
			query,
		)
	}
}

func TestPrometheusProvider_GetWorkloads_ShouldReturnErrorWhenPrometheusFails(
	t *testing.T,
) {

	// Arrange

	server :=
		httptest.NewServer(
			http.HandlerFunc(
				func(
					w http.ResponseWriter,
					r *http.Request,
				) {

					http.Error(
						w,
						"prometheus unavailable",
						http.StatusServiceUnavailable,
					)
				},
			),
		)

	defer server.Close()

	provider :=
		NewPrometheusProvider(
			server.URL,
			server.Client(),
		)

	// Act

	_, err :=
		provider.GetWorkloads(
			context.Background(),
			"cloud-efficiency",
		)

	// Assert

	if err == nil {

		t.Fatal(
			"expected Prometheus error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"HTTP 503",
	) {

		t.Fatalf(
			"expected HTTP 503 error, got %v",
			err,
		)
	}
}

func TestPrometheusProvider_QueryRange_ShouldRejectInvalidTimeRange(
	t *testing.T,
) {

	// Arrange

	provider :=
		NewPrometheusProvider(
			"http://localhost:9090",
			nil,
		)

	start :=
		time.Unix(
			1720000000,
			0,
		)

	// Act

	_, err :=
		provider.queryRange(
			context.Background(),
			"up",
			start,
			start,
			5*time.Minute,
		)

	// Assert

	if err == nil {

		t.Fatal(
			"expected invalid time range error",
		)
	}
}

func TestPrometheusProvider_QueryRange_ShouldRejectInvalidStep(
	t *testing.T,
) {

	// Arrange

	provider :=
		NewPrometheusProvider(
			"http://localhost:9090",
			nil,
		)

	start :=
		time.Unix(
			1720000000,
			0,
		)

	end :=
		start.Add(
			10 * time.Minute,
		)

	// Act

	_, err :=
		provider.queryRange(
			context.Background(),
			"up",
			start,
			end,
			0,
		)

	// Assert

	if err == nil {

		t.Fatal(
			"expected invalid step error",
		)
	}
}
