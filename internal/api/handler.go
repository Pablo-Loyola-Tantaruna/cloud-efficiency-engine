package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"cloud-efficiency-engine/internal/analysis"
)

type Handler struct {
	engine *analysis.Engine
}

func NewHandler(
	engine *analysis.Engine,
) *Handler {

	return &Handler{
		engine: engine,
	}
}

func (h *Handler) Health(
	w http.ResponseWriter,
	r *http.Request,
) {

	response :=
		map[string]string{
			"status": "UP",
		}

	writeJSON(
		w,
		http.StatusOK,
		response,
	)
}

func (h *Handler) Analyze(
	w http.ResponseWriter,
	r *http.Request,
) {

	now :=
		time.Now().UTC()

	lookbackHours :=
		getLookbackHours()

	options :=
		analysis.AnalysisOptions{
			Start: now.Add(
				-time.Duration(
					lookbackHours,
				) * time.Hour,
			),

			End: now,

			Step: 5 * time.Minute,
		}

	result, err :=
		h.engine.Analyze(
			r.Context(),
			options,
		)

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

	writeJSON(
		w,
		http.StatusOK,
		result,
	)
}

func getLookbackHours() int {

	raw :=
		os.Getenv(
			"ANALYSIS_LOOKBACK_HOURS",
		)

	if raw == "" {
		return 168
	}

	value, err :=
		strconv.Atoi(raw)

	if err != nil ||
		value <= 0 {

		return 168
	}

	return value
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	data interface{},
) {

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	_ = json.NewEncoder(
		w,
	).Encode(data)
}
