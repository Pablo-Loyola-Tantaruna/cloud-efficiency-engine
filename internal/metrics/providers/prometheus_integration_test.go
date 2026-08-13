package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPrometheusProvider_GetWorkloads_Integration(
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
						r.URL.Query().Get("query")

					var response interface{}

					switch query {

					case `cee_workload_cpu_request_millicores{namespace="cloud-efficiency"}`:

						response =
							prometheusVectorResponse(
								"cloud-efficiency",
								"payments-api",
								"1000",
							)

					case `cee_workload_cpu_usage_millicores{namespace="cloud-efficiency"}`:

						response =
							prometheusVectorResponse(
								"cloud-efficiency",
								"payments-api",
								"350",
							)

					case `cee_workload_memory_request_bytes{namespace="cloud-efficiency"}`:

						response =
							prometheusVectorResponse(
								"cloud-efficiency",
								"payments-api",
								"2147483648",
							)

					case `cee_workload_memory_usage_bytes{namespace="cloud-efficiency"}`:

						response =
							prometheusVectorResponse(
								"cloud-efficiency",
								"payments-api",
								"734003200",
							)

					case `cee_workload_replicas{namespace="cloud-efficiency"}`:

						response =
							prometheusVectorResponse(
								"cloud-efficiency",
								"payments-api",
								"3",
							)

					default:

						t.Fatalf(
							"unexpected Prometheus query: %s",
							query,
						)
					}

					w.Header().Set(
						"Content-Type",
						"application/json",
					)

					w.WriteHeader(
						http.StatusOK,
					)

					if err :=
						json.NewEncoder(w).
							Encode(response); err != nil {

						t.Fatalf(
							"failed to encode response: %v",
							err,
						)
					}
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

	workloads, err :=
		provider.GetWorkloads(
			context.Background(),
			"cloud-efficiency",
		)

	// Assert

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if len(workloads) != 1 {

		t.Fatalf(
			"expected 1 workload, got %d",
			len(workloads),
		)
	}

	workload :=
		workloads[0]

	if workload.Namespace !=
		"cloud-efficiency" {

		t.Fatalf(
			"expected namespace cloud-efficiency, got %s",
			workload.Namespace,
		)
	}

	if workload.Name !=
		"payments-api" {

		t.Fatalf(
			"expected workload payments-api, got %s",
			workload.Name,
		)
	}

	if workload.CPURequestMillicores !=
		1000 {

		t.Fatalf(
			"expected CPU request 1000m, got %d",
			workload.CPURequestMillicores,
		)
	}

	if workload.CPUUsageMillicores !=
		350 {

		t.Fatalf(
			"expected CPU usage 350m, got %d",
			workload.CPUUsageMillicores,
		)
	}

	if workload.MemoryRequestBytes !=
		2147483648 {

		t.Fatalf(
			"expected memory request 2147483648, got %d",
			workload.MemoryRequestBytes,
		)
	}

	if workload.MemoryUsageBytes !=
		734003200 {

		t.Fatalf(
			"expected memory usage 734003200, got %d",
			workload.MemoryUsageBytes,
		)
	}

	if workload.Replicas != 3 {

		t.Fatalf(
			"expected 3 replicas, got %d",
			workload.Replicas,
		)
	}
}

func prometheusVectorResponse(
	namespace string,
	workload string,
	value string,
) map[string]interface{} {

	return map[string]interface{}{
		"status": "success",

		"data": map[string]interface{}{
			"resultType": "vector",

			"result": []map[string]interface{}{
				{
					"metric": map[string]string{
						"namespace": namespace,
						"workload":  workload,
					},

					"value": []interface{}{
						float64(1720000000),
						value,
					},
				},
			},
		},
	}
}
