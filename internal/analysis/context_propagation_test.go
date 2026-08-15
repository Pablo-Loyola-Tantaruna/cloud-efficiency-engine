package analysis

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/analysis/optimizer"
	"cloud-efficiency-engine/internal/analysis/resolver"
	"cloud-efficiency-engine/internal/analysis/rules"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics/providers"
	"cloud-efficiency-engine/internal/pricing"
)

func TestEngineAnalyze_ShouldPropagateAnalysisContext(
	t *testing.T,
) {

	now :=
		time.Now().UTC()

	provider :=
		providers.NewMockProvider(
			[]domain.WorkloadMetrics{
				{
					Namespace: "payments",

					Name: "payments-api",

					Type: domain.WorkloadDeployment,

					Replicas: 2,

					CPURequestMillicores: 500,

					MemoryRequestBytes: 1024 *
						1024 *
						1024,
				},
			},
		)

	historicalProvider :=
		providers.NewMockHistoricalProvider(
			[]domain.WorkloadHistory{
				{
					Namespace: "payments",

					Name: "payments-api",

					CPUUsageMillicores: []domain.MetricSample{
						{
							Timestamp: now.Add(
								-1 * time.Hour,
							),

							Value: 100,
						},
					},

					MemoryUsageBytes: []domain.MetricSample{
						{
							Timestamp: now.Add(
								-1 * time.Hour,
							),

							Value: 512 *
								1024 *
								1024,
						},
					},
				},
			},
		)

	pricingProvider :=
		&testPricingProvider{
			prices: pricing.ResourcePricing{
				CPUPerCoreHour: 0.04,

				MemoryPerGBHour: 0.005,
			},
		}

	calculator :=
		cost.NewCalculator(
			730,
		)

	engine :=
		NewEngine(
			provider,
			historicalProvider,
			pricingProvider,

			[]rules.Rule{
				rules.NewCPUOverprovisioningRule(),
				rules.NewMemoryOverprovisioningRule(),
			},

			optimizer.DefaultOptimizationPolicy(),

			resolver.NewResolver(),

			calculator,
		)

	expectedContext :=
		domain.AnalysisContext{
			Provider: domain.CloudProviderAWS,

			Environment: "production",

			AccountID: "123456789",

			Region: "us-east-1",

			ClusterName: "prod-eks",
		}

	result, err :=
		engine.Analyze(
			context.Background(),
			AnalysisOptions{
				Namespace: "payments",

				Start: now.Add(
					-2 * time.Hour,
				),

				End: now,

				Step: 5 * time.Minute,

				Context: expectedContext,
			},
		)

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if result == nil {

		t.Fatal(
			"expected analysis report",
		)
	}

	if result.Context.Provider !=
		expectedContext.Provider {

		t.Fatalf(
			"expected provider %s, got %s",
			expectedContext.Provider,
			result.Context.Provider,
		)
	}

	if result.Context.Environment !=
		expectedContext.Environment {

		t.Fatalf(
			"expected environment %s, got %s",
			expectedContext.Environment,
			result.Context.Environment,
		)
	}

	if result.Context.AccountID !=
		expectedContext.AccountID {

		t.Fatalf(
			"expected account %s, got %s",
			expectedContext.AccountID,
			result.Context.AccountID,
		)
	}

	if result.Context.Region !=
		expectedContext.Region {

		t.Fatalf(
			"expected region %s, got %s",
			expectedContext.Region,
			result.Context.Region,
		)
	}

	if result.Context.ClusterName !=
		expectedContext.ClusterName {

		t.Fatalf(
			"expected cluster %s, got %s",
			expectedContext.ClusterName,
			result.Context.ClusterName,
		)
	}
}

func TestEngineAnalyze_ShouldDefaultMissingContextToKubernetes(
	t *testing.T,
) {

	provider :=
		providers.NewMockProvider(
			[]domain.WorkloadMetrics{},
		)

	engine :=
		NewEngine(
			provider,
			nil,
			nil,
			nil,
			optimizer.DefaultOptimizationPolicy(),
			nil,
			cost.NewCalculator(
				730,
			),
		)

	now :=
		time.Now().UTC()

	result, err :=
		engine.Analyze(
			context.Background(),
			AnalysisOptions{
				Namespace: "cloud-efficiency-engine",

				Start: now.Add(
					-1 * time.Hour,
				),

				End: now,

				Step: 5 * time.Minute,
			},
		)

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if result == nil {

		t.Fatal(
			"expected analysis report",
		)
	}

	if result.Context.Provider !=
		domain.CloudProviderKubernetes {

		t.Fatalf(
			"expected kubernetes provider, got %s",
			result.Context.Provider,
		)
	}

	if result.Context.Environment !=
		"unknown" {

		t.Fatalf(
			"expected unknown environment, got %s",
			result.Context.Environment,
		)
	}
}
