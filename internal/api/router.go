package api

import (
	"log/slog"
	"net/http"
)

func NewRouter(
	handler *Handler,
	logger *slog.Logger,
	httpMetrics *HTTPMetrics,
	analysisMetrics *AnalysisMetrics,
	finopsMetrics ...http.Handler,
) http.Handler {

	mux :=
		http.NewServeMux()

	mux.HandleFunc(
		"/health",
		handler.Health,
	)

	mux.HandleFunc(
		"/ready",
		handler.Ready,
	)

	mux.Handle(
		"/metrics",
		metricsHandler(
			httpMetrics,
			analysisMetrics,
		),
	)

	if len(finopsMetrics) > 0 &&
		finopsMetrics[0] != nil {

		mux.Handle(
			"/metrics/finops",
			finopsMetrics[0],
		)
	}

	mux.HandleFunc(
		"/api/v1/analyze",
		handler.Analyze,
	)

	return requestIDMiddleware(
		metricsMiddleware(
			httpMetrics,
			loggingMiddleware(
				logger,
				mux,
			),
		),
	)
}
