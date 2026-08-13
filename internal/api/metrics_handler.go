package api

import (
	"net/http"
)

func metricsHandler(
	httpMetrics *HTTPMetrics,
	analysisMetrics *AnalysisMetrics,
) http.Handler {

	return http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {

			httpMetrics.WriteMetrics(
				w,
			)

			analysisMetrics.WriteMetrics(
				w,
			)
		},
	)
}
