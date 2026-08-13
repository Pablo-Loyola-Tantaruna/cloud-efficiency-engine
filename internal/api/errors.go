package api

import (
	"encoding/json"
	"net/http"
)

const (
	ErrCodeInvalidRequest        = "INVALID_REQUEST"
	ErrCodeAnalysisFailed        = "ANALYSIS_FAILED"
	ErrCodeDependencyUnavailable = "DEPENDENCY_UNAVAILABLE"
	ErrCodeInternal              = "INTERNAL_ERROR"
)

func writeError(
	w http.ResponseWriter,
	status int,
	code string,
	message string,
	requestID string,
) {

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	response :=
		ErrorResponse{
			Code:      code,
			Message:   message,
			RequestID: requestID,
		}

	_ = json.NewEncoder(w).Encode(response)
}
