package api

type AnalyzeRequest struct {
	Namespace     string `json:"namespace"`
	LookbackHours int    `json:"lookbackHours"`
	StepSeconds   int    `json:"stepSeconds"`
}

type ErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type ReadinessResponse struct {
	Status       string            `json:"status"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
}
