package api

import (
	"log/slog"
	"net/http"

	"cloud-efficiency-engine/internal/security"
)

// NewRouter preserves the original router constructor for existing callers.
func NewRouter(
	handler *Handler,
	logger *slog.Logger,
	httpMetrics *HTTPMetrics,
	analysisMetrics *AnalysisMetrics,
	finopsMetrics ...http.Handler,
) http.Handler {
	return buildRouter(handler, logger, httpMetrics, analysisMetrics, nil, nil, finopsMetrics...)
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
	return buildRouter(handler, logger, httpMetrics, analysisMetrics, finOpsHandler, nil, finopsMetrics...)
}

// NewSecureFinOpsRouter exposes the control plane only through authenticated requests.
func NewSecureFinOpsRouter(
	handler *Handler,
	logger *slog.Logger,
	httpMetrics *HTTPMetrics,
	analysisMetrics *AnalysisMetrics,
	finOpsHandler *FinOpsHandler,
	auth *security.Middleware,
	finopsMetrics ...http.Handler,
) http.Handler {
	return buildRouter(handler, logger, httpMetrics, analysisMetrics, finOpsHandler, auth, finopsMetrics...)
}

func buildRouter(
	handler *Handler,
	logger *slog.Logger,
	httpMetrics *HTTPMetrics,
	analysisMetrics *AnalysisMetrics,
	finOpsHandler *FinOpsHandler,
	auth *security.Middleware,
	finopsMetrics ...http.Handler,
) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", handler.Health)
	mux.HandleFunc("/ready", handler.Ready)
	mux.Handle("/metrics", metricsHandler(httpMetrics, analysisMetrics))

	if len(finopsMetrics) > 0 && finopsMetrics[0] != nil {
		mux.Handle("/metrics/finops", finopsMetrics[0])
	}

	apiHandler := http.Handler(http.HandlerFunc(handler.Analyze))
	if auth != nil {
		apiHandler = auth.Protect(security.AuthorizeFinOps(apiHandler))
	}
	mux.Handle("/api/v1/analyze", apiHandler)

	if finOpsHandler != nil {
		if auth != nil {
			mux.Handle("/api/v1/action-plans", auth.Protect(security.AuthorizeFinOps(http.HandlerFunc(finOpsHandler.actionPlans))))
			mux.Handle("/api/v1/action-plans/", auth.Protect(security.AuthorizeFinOps(http.HandlerFunc(finOpsHandler.actionPlan))))
			mux.Handle("/api/v1/executions/", auth.Protect(security.AuthorizeFinOps(http.HandlerFunc(finOpsHandler.execution))))
			mux.Handle("/api/v1/recovery/", auth.Protect(security.AuthorizeFinOps(http.HandlerFunc(finOpsHandler.recovery))))
		} else {
			finOpsHandler.Register(mux)
		}
	}

	return requestIDMiddleware(
		metricsMiddleware(
			httpMetrics,
			loggingMiddleware(logger, mux),
		),
	)
}
