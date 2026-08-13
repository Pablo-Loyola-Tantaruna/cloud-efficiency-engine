package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"cloud-efficiency-engine/internal/analysis"
)

const (
	defaultLookbackHours = 168
	defaultStepSeconds   = 300

	minLookbackHours = 1
	maxLookbackHours = 720

	minStepSeconds = 15
	maxStepSeconds = 3600
)

type Handler struct {
	engine AnalysisService
}

func NewHandler(
	engine AnalysisService,
) *Handler {

	return &Handler{
		engine: engine,
	}
}

func (h *Handler) Analyze(
	w http.ResponseWriter,
	r *http.Request,
) {

	requestID :=
		requestIDFromContext(
			r.Context(),
		)

	if r.Method != http.MethodPost {

		writeError(
			w,
			http.StatusMethodNotAllowed,
			ErrCodeInvalidRequest,
			"method not allowed",
			requestID,
		)

		return
	}

	var request AnalyzeRequest

	if err :=
		json.NewDecoder(
			r.Body,
		).Decode(&request); err != nil {

		writeError(
			w,
			http.StatusBadRequest,
			ErrCodeInvalidRequest,
			"invalid JSON request body",
			requestID,
		)

		return
	}

	if strings.TrimSpace(
		request.Namespace,
	) == "" {

		writeError(
			w,
			http.StatusBadRequest,
			ErrCodeInvalidRequest,
			"namespace must not be empty",
			requestID,
		)

		return
	}

	request.Namespace =
		strings.TrimSpace(
			request.Namespace,
		)

	if request.LookbackHours == 0 {
		request.LookbackHours =
			defaultLookbackHours
	}

	if request.StepSeconds == 0 {
		request.StepSeconds =
			defaultStepSeconds
	}

	if request.LookbackHours <
		minLookbackHours ||
		request.LookbackHours >
			maxLookbackHours {

		writeError(
			w,
			http.StatusBadRequest,
			ErrCodeInvalidRequest,
			"lookbackHours must be between 1 and 720",
			requestID,
		)

		return
	}

	if request.StepSeconds <
		minStepSeconds ||
		request.StepSeconds >
			maxStepSeconds {

		writeError(
			w,
			http.StatusBadRequest,
			ErrCodeInvalidRequest,
			"stepSeconds must be between 15 and 3600",
			requestID,
		)

		return
	}

	end :=
		time.Now().UTC()

	start :=
		end.Add(
			-time.Duration(
				request.LookbackHours,
			) * time.Hour,
		)

	report, err :=
		h.engine.Analyze(
			r.Context(),
			analysis.AnalysisOptions{
				Namespace: request.Namespace,
				Start:     start,
				End:       end,
				Step: time.Duration(
					request.StepSeconds,
				) * time.Second,
			},
		)

	if err != nil {

		writeError(
			w,
			http.StatusInternalServerError,
			ErrCodeAnalysisFailed,
			"analysis failed",
			requestID,
		)

		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(
		http.StatusOK,
	)

	_ = json.NewEncoder(
		w,
	).Encode(report)
}

func (h *Handler) Health(
	w http.ResponseWriter,
	r *http.Request,
) {

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(
		http.StatusOK,
	)

	_ = json.NewEncoder(
		w,
	).Encode(
		HealthResponse{
			Status: "UP",
		},
	)
}

func (h *Handler) Ready(
	w http.ResponseWriter,
	r *http.Request,
) {

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(
		http.StatusOK,
	)

	_ = json.NewEncoder(
		w,
	).Encode(
		ReadinessResponse{
			Status: "UP",
		},
	)
}
