package api

import "net/http"

func NewRouter(
	handler *Handler,
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

	mux.HandleFunc(
		"/api/v1/analyze",
		handler.Analyze,
	)

	return requestIDMiddleware(
		mux,
	)
}
