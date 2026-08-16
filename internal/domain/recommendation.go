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

type RecommendationPriority string

const (
	RecommendationPriorityLow      RecommendationPriority = "LOW"
	RecommendationPriorityMedium   RecommendationPriority = "MEDIUM"
	RecommendationPriorityHigh     RecommendationPriority = "HIGH"
	RecommendationPriorityCritical RecommendationPriority = "CRITICAL"
)

type SavingsSource string

const (
	SavingsSourceEstimated      SavingsSource = "ESTIMATED"
	SavingsSourceProviderPriced SavingsSource = "PROVIDER_PRICED"
	SavingsSourceActual         SavingsSource = "ACTUAL"
)

func SafetyScoreForConfidence(confidence Confidence) int {
	switch confidence {
	case ConfidenceHigh:
		return 90
	case ConfidenceMedium:
		return 70
	case ConfidenceLow:
		return 40
	default:
		return 0
	}
}

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

	CurrentNodeCount   int64 `json:"currentNodeCount,omitempty"`
	SuggestedNodeCount int64 `json:"suggestedNodeCount,omitempty"`

	MonthlySavingsUSD    float64                `json:"monthlySavingsUsd,omitempty"`
	AnnualizedSavingsUSD float64                `json:"annualizedSavingsUsd,omitempty"`
	SavingsPercentage    float64                `json:"savingsPercentage,omitempty"`
	Priority             RecommendationPriority `json:"priority,omitempty"`
	Actionable           bool                   `json:"actionable"`

	SafetyScore   int           `json:"safetyScore,omitempty"`
	SavingsSource SavingsSource `json:"savingsSource,omitempty"`
}
