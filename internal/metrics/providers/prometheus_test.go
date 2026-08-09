package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPrometheusProvider_GetWorkloads_ShouldParseMetrics(t *testing.T) {
	// Arrange
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			query := r.URL.Query().Get("query")

			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			switch query {

			case "cee_workload_cpu_request_millicores":
				_, _ = w.Write([]byte(`
{
  "status": "success",
  "data": {
    "resultType": "vector",
    "result": [
      {
        "metric": {
          "namespace": "payments",
          "workload": "payments-api"
        },
        "value": [1700000000, "1000"]
      }
    ]
  }
}`))

			case "cee_workload_cpu_usage_millicores":
				_, _ = w.Write([]byte(`
{
  "status": "success",
  "data": {
    "resultType": "vector",
    "result": [
      {
        "metric": {
          "namespace": "payments",
          "workload": "payments-api"
        },
        "value": [1700000000, "180"]
      }
    ]
  }
}`))

			case "cee_workload_memory_request_bytes":
				_, _ = w.Write([]byte(`
{
  "status": "success",
  "data": {
    "resultType": "vector",
    "result": [
      {
        "metric": {
          "namespace": "payments",
          "workload": "payments-api"
        },
        "value": [1700000000, "2147483648"]
      }
    ]
  }
}`))

			case "cee_workload_memory_usage_bytes":
				_, _ = w.Write([]byte(`
{
  "status": "success",
  "data": {
    "resultType": "vector",
    "result": [
      {
        "metric": {
          "namespace": "payments",
          "workload": "payments-api"
        },
        "value": [1700000000, "671088640"]
      }
    ]
  }
}`))

			default:
				http.Error(w, "unknown query", http.StatusBadRequest)
			}
		}),
	)

	defer server.Close()

	provider := NewPrometheusProvider(
		server.URL,
		server.Client(),
	)

	// Act
	result, err := provider.GetWorkloads(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != 1 {
		t.Fatalf(
			"expected 1 workload, got %d",
			len(result),
		)
	}

	workload := result[0]

	if workload.Namespace != "payments" {
		t.Errorf(
			"expected namespace payments, got %s",
			workload.Namespace,
		)
	}

	if workload.Name != "payments-api" {
		t.Errorf(
			"expected workload payments-api, got %s",
			workload.Name,
		)
	}

	if workload.CPURequestMillicores != 1000 {
		t.Errorf(
			"expected CPU request 1000m, got %d",
			workload.CPURequestMillicores,
		)
	}

	if workload.CPUUsageMillicores != 180 {
		t.Errorf(
			"expected CPU usage 180m, got %d",
			workload.CPUUsageMillicores,
		)
	}

	if workload.MemoryRequestBytes != 2147483648 {
		t.Errorf(
			"expected memory request 2147483648, got %d",
			workload.MemoryRequestBytes,
		)
	}
}
