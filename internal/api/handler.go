package api

import (
	"encoding/json"
	"net/http"

	"cloud-efficiency-engine/internal/analysis"
)

type Handler struct {
	analyzer *analysis.Analyzer
}

func NewHandler(analyzer *analysis.Analyzer) *Handler {
	return &Handler{
		analyzer: analyzer,
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status": "UP",
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) Analyze(w http.ResponseWriter, r *http.Request) {
	result, err := h.analyzer.Analyze(r.Context())

	if err != nil {
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": err.Error(),
			},
		)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	data interface{},
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
