package api

import (
	"log/slog"
	"net/http"
)

// NewRouter preserves the original router constructor for existing callers.
func NewRouter(
	handler *Handler,
	logger *slog.Logger,
	httpMetrics *HTTPMetrics,
	analysisMetrics *AnalysisMetrics,
	finopsMetrics ...http.Handler,
) http.Handler {
	return buildRouter(handler, logger, httpMetrics, analysisMetrics, nil, finopsMetrics...)
}

// NewFinOpsRouter registers the FinOps control plane in addition to the analysis API.
func NewFinOpsRouter(
	handler *Handler,
	logger *slog.Logger,
	httpMetrics *HTTPMetrics,
	analysisMetrics *AnalysisMetrics,
	finOpsHandler *FinOpsHandler,
	finopsMetrics ...http.Handler,
) http.Handler {
	return buildRouter(handler, logger, httpMetrics, analysisMetrics, finOpsHandler, finopsMetrics...)
}

func buildRouter(
	handler *Handler,
	logger *slog.Logger,
	httpMetrics *HTTPMetrics,
	analysisMetrics *AnalysisMetrics,
	finOpsHandler *FinOpsHandler,
	finopsMetrics ...http.Handler,
) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", handler.Health)
	mux.HandleFunc("/ready", handler.Ready)
	mux.Handle("/metrics", metricsHandler(httpMetrics, analysisMetrics))

	if len(finopsMetrics) > 0 && finopsMetrics[0] != nil {
		mux.Handle("/metrics/finops", finopsMetrics[0])
	}

	mux.HandleFunc("/api/v1/analyze", handler.Analyze)
	if finOpsHandler != nil {
		finOpsHandler.Register(mux)
	}

	return requestIDMiddleware(
		metricsMiddleware(
			httpMetrics,
			loggingMiddleware(logger, mux),
		),
	)
}
