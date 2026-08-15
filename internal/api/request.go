package api

type AnalyzeRequest struct {
	Namespace string `json:"namespace"`

	Provider string `json:"provider"`

	Environment string `json:"environment"`

	AccountID string `json:"accountId,omitempty"`

	Region string `json:"region,omitempty"`

	ClusterName string `json:"clusterName,omitempty"`

	LookbackHours int `json:"lookbackHours"`

	StepSeconds int `json:"stepSeconds"`
}
