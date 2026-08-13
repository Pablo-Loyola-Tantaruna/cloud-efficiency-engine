package api

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HTTPMetrics struct {
	mu sync.RWMutex

	requests map[requestMetricKey]*requestMetric

	requestDurationSeconds float64
	requestDurationCount   uint64
}

type requestMetricKey struct {
	method string
	path   string
	status int
}

type requestMetric struct {
	count uint64
}

func NewHTTPMetrics() *HTTPMetrics {
	return &HTTPMetrics{
		requests: make(
			map[requestMetricKey]*requestMetric,
		),
	}
}

func (m *HTTPMetrics) RecordRequest(
	method string,
	path string,
	status int,
	duration time.Duration,
) {

	m.mu.Lock()
	defer m.mu.Unlock()

	key :=
		requestMetricKey{
			method: method,
			path:   path,
			status: status,
		}

	metric :=
		m.requests[key]

	if metric == nil {

		metric =
			&requestMetric{}

		m.requests[key] =
			metric
	}

	metric.count++

	m.requestDurationSeconds +=
		duration.Seconds()

	m.requestDurationCount++
}

func (m *HTTPMetrics) WriteMetrics(
	w http.ResponseWriter,
) {

	m.mu.RLock()

	requests :=
		make(
			map[requestMetricKey]uint64,
			len(m.requests),
		)

	for key, metric := range m.requests {

		requests[key] =
			metric.count
	}

	durationSum :=
		m.requestDurationSeconds

	durationCount :=
		m.requestDurationCount

	m.mu.RUnlock()

	_, _ =
		w.Write(
			[]byte(
				"# HELP cee_http_requests_total Total number of HTTP requests handled by the Cloud Efficiency Engine.\n" +
					"# TYPE cee_http_requests_total counter\n",
			),
		)

	keys :=
		make(
			[]requestMetricKey,
			0,
			len(requests),
		)

	for key := range requests {

		keys =
			append(
				keys,
				key,
			)
	}

	sort.Slice(
		keys,
		func(i, j int) bool {

			if keys[i].method !=
				keys[j].method {

				return keys[i].method <
					keys[j].method
			}

			if keys[i].path !=
				keys[j].path {

				return keys[i].path <
					keys[j].path
			}

			return keys[i].status <
				keys[j].status
		},
	)

	for _, key := range keys {

		_, _ =
			fmt.Fprintf(
				w,
				"cee_http_requests_total{method=%q,path=%q,status=%q} %d\n",
				key.method,
				key.path,
				strconv.Itoa(
					key.status,
				),
				requests[key],
			)
	}

	_, _ =
		w.Write(
			[]byte(
				"# HELP cee_http_request_duration_seconds_sum Total HTTP request processing time in seconds.\n" +
					"# TYPE cee_http_request_duration_seconds_sum counter\n",
			),
		)

	_, _ =
		fmt.Fprintf(
			w,
			"cee_http_request_duration_seconds_sum %f\n",
			durationSum,
		)

	_, _ =
		w.Write(
			[]byte(
				"# HELP cee_http_request_duration_seconds_count Total number of HTTP request duration observations.\n" +
					"# TYPE cee_http_request_duration_seconds_count counter\n",
			),
		)

	_, _ =
		fmt.Fprintf(
			w,
			"cee_http_request_duration_seconds_count %d\n",
			durationCount,
		)
}

func (m *HTTPMetrics) Handler(
	w http.ResponseWriter,
	r *http.Request,
) {

	w.Header().Set(
		"Content-Type",
		"text/plain; version=0.0.4; charset=utf-8",
	)

	w.WriteHeader(
		http.StatusOK,
	)

	m.WriteMetrics(
		w,
	)
}

func metricsMiddleware(
	metrics *HTTPMetrics,
	next http.Handler,
) http.Handler {

	return http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {

			start :=
				time.Now()

			responseWriter :=
				newStatusResponseWriter(
					w,
				)

			next.ServeHTTP(
				responseWriter,
				r,
			)

			duration :=
				time.Since(start)

			path :=
				r.URL.Path

			if path == "" {
				path = "/"
			}

			metrics.RecordRequest(
				r.Method,
				path,
				responseWriter.statusCode,
				duration,
			)
		},
	)
}

func MetricsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {

	if r.Method != http.MethodGet {

		writeError(
			w,
			http.StatusMethodNotAllowed,
			ErrCodeInvalidRequest,
			"method not allowed",
			requestIDFromContext(
				r.Context(),
			),
		)

		return
	}

	promhttp.Handler().ServeHTTP(
		w,
		r,
	)
}
