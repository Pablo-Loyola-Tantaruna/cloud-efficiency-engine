package api

import (
	"net/http"

	"cloud-efficiency-engine/internal/observability"
)

func observabilityMetricsHandler(
	metrics *observability.Metrics,
) http.Handler {

	if metrics == nil {

		return http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {

				writeError(
					w,
					http.StatusInternalServerError,
					ErrCodeAnalysisFailed,
					"observability metrics are not configured",
					requestIDFromContext(
						r.Context(),
					),
				)
			},
		)
	}

	return metrics.Handler()
}
