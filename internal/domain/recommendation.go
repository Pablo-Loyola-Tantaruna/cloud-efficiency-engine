package domain

type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityWarning  Severity = "WARNING"
	SeverityCritical Severity = "CRITICAL"
)

type Confidence string

const (
	ConfidenceLow    Confidence = "LOW"
	ConfidenceMedium Confidence = "MEDIUM"
	ConfidenceHigh   Confidence = "HIGH"
)

type Recommendation struct {
	Rule        string     `json:"rule"`
	Workload    string     `json:"workload"`
	Description string     `json:"description"`
	Severity    Severity   `json:"severity"`
	Confidence  Confidence `json:"confidence"`

	CurrentCPURequestMillicores   int64 `json:"currentCpuRequestMillicores,omitempty"`
	SuggestedCPURequestMillicores int64 `json:"suggestedCpuRequestMillicores,omitempty"`

	CurrentMemoryRequestBytes   int64 `json:"currentMemoryRequestBytes,omitempty"`
	SuggestedMemoryRequestBytes int64 `json:"suggestedMemoryRequestBytes,omitempty"`
}
