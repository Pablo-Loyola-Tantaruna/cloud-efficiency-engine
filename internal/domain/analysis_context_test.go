package domain

import "testing"

func TestCloudProviderIsValid_ShouldAcceptSupportedProviders(
	t *testing.T,
) {

	testCases := []struct {
		name     string
		provider CloudProvider
		expected bool
	}{
		{
			name: "kubernetes",

			provider: CloudProviderKubernetes,

			expected: true,
		},
		{
			name: "aws",

			provider: CloudProviderAWS,

			expected: true,
		},
		{
			name: "azure",

			provider: CloudProviderAzure,

			expected: true,
		},
		{
			name: "gcp",

			provider: CloudProviderGCP,

			expected: true,
		},
		{
			name: "unknown",

			provider: CloudProviderUnknown,

			expected: false,
		},
	}

	for _, testCase := range testCases {

		t.Run(
			testCase.name,
			func(t *testing.T) {

				actual :=
					testCase.provider.IsValid()

				if actual !=
					testCase.expected {

					t.Fatalf(
						"expected %v, got %v",
						testCase.expected,
						actual,
					)
				}
			},
		)
	}
}

func TestDefaultKubernetesAnalysisContext_ShouldCreateExpectedContext(
	t *testing.T,
) {

	value :=
		DefaultKubernetesAnalysisContext(
			"production",
			"kind-cloud-efficiency",
		)

	if value.Provider !=
		CloudProviderKubernetes {

		t.Fatalf(
			"expected kubernetes provider, got %s",
			value.Provider,
		)
	}

	if value.Environment !=
		"production" {

		t.Fatalf(
			"expected production environment, got %s",
			value.Environment,
		)
	}

	if value.ClusterName !=
		"kind-cloud-efficiency" {

		t.Fatalf(
			"expected cluster name kind-cloud-efficiency, got %s",
			value.ClusterName,
		)
	}
}

func TestNormalizeAnalysisContext_ShouldDefaultMissingProvider(
	t *testing.T,
) {

	value :=
		NormalizeAnalysisContext(
			AnalysisContext{},
		)

	if value.Provider !=
		CloudProviderKubernetes {

		t.Fatalf(
			"expected kubernetes provider, got %s",
			value.Provider,
		)
	}

	if value.Environment !=
		"unknown" {

		t.Fatalf(
			"expected unknown environment, got %s",
			value.Environment,
		)
	}
}

func TestNormalizeAnalysisContext_ShouldPreserveExistingValues(
	t *testing.T,
) {

	value :=
		NormalizeAnalysisContext(
			AnalysisContext{
				Provider: CloudProviderAWS,

				Environment: "production",

				AccountID: "123456789",

				Region: "us-east-1",

				ClusterName: "prod-eks",
			},
		)

	if value.Provider !=
		CloudProviderAWS {

		t.Fatalf(
			"expected aws provider, got %s",
			value.Provider,
		)
	}

	if value.Environment !=
		"production" {

		t.Fatalf(
			"expected production environment, got %s",
			value.Environment,
		)
	}

	if value.AccountID !=
		"123456789" {

		t.Fatalf(
			"expected account id 123456789, got %s",
			value.AccountID,
		)
	}

	if value.Region !=
		"us-east-1" {

		t.Fatalf(
			"expected region us-east-1, got %s",
			value.Region,
		)
	}

	if value.ClusterName !=
		"prod-eks" {

		t.Fatalf(
			"expected cluster prod-eks, got %s",
			value.ClusterName,
		)
	}
}
