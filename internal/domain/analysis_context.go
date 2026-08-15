package domain

type AnalysisContext struct {
	Provider    CloudProvider `json:"provider"`
	Environment string        `json:"environment"`

	AccountID string `json:"accountId,omitempty"`

	Region string `json:"region,omitempty"`

	ClusterName string `json:"clusterName,omitempty"`
}

func DefaultKubernetesAnalysisContext(
	environment string,
	clusterName string,
) AnalysisContext {

	if environment == "" {
		environment = "unknown"
	}

	return AnalysisContext{
		Provider: CloudProviderKubernetes,

		Environment: environment,

		ClusterName: clusterName,
	}
}

func NormalizeAnalysisContext(
	value AnalysisContext,
) AnalysisContext {

	if value.Provider == "" {

		value.Provider =
			CloudProviderKubernetes
	}

	if value.Environment == "" {

		value.Environment =
			"unknown"
	}

	return value
}
