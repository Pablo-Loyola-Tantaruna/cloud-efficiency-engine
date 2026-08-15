package domain

type CloudProvider string

const (
	CloudProviderUnknown    CloudProvider = "unknown"
	CloudProviderKubernetes CloudProvider = "kubernetes"
	CloudProviderAWS        CloudProvider = "aws"
	CloudProviderAzure      CloudProvider = "azure"
	CloudProviderGCP        CloudProvider = "gcp"
)

func (p CloudProvider) IsValid() bool {

	switch p {

	case CloudProviderKubernetes,
		CloudProviderAWS,
		CloudProviderAzure,
		CloudProviderGCP:

		return true

	default:

		return false
	}
}
