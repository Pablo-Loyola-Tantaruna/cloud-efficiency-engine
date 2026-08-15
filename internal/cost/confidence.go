package cost

type CostConfidence string

const (
	CostConfidenceHigh CostConfidence = "HIGH"

	CostConfidenceMedium CostConfidence = "MEDIUM"

	CostConfidenceLow CostConfidence = "LOW"
)

type CostQuality struct {
	Confidence CostConfidence `json:"confidence"`

	HasActualBillingData bool `json:"hasActualBillingData"`

	HasNodeLevelAttribution bool `json:"hasNodeLevelAttribution"`

	HasWorkloadRequests bool `json:"hasWorkloadRequests"`
}
