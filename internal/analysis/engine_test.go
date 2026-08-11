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

type testPricingProvider struct {
	prices pricing.ResourcePricing
}

func (p *testPricingProvider) GetPricing(
	ctx context.Context,
) (pricing.ResourcePricing, error) {

	return p.prices, nil
}

func TestEngine_Analyze_ShouldCalculateCostUsingPricingProvider(
	t *testing.T,
) {

	// Arrange

	workload :=
		domain.WorkloadMetrics{
			Namespace: "payments",
			Name:      "payments-api",

			Type: domain.WorkloadDeployment,

			Replicas: 3,

			CPURequestMillicores: 1000,

			MemoryRequestBytes: 2 *
				1024 *
				1024 *
				1024,
		}

	provider :=
		providers.NewMockProvider(
			[]domain.WorkloadMetrics{
				workload,
			},
		)

	now :=
		time.Now().UTC()

	historicalProvider :=
		providers.NewMockHistoricalProvider(
			[]domain.WorkloadHistory{
				{
					Namespace: "payments",
					Name:      "payments-api",

					CPUUsageMillicores: buildEngineCPUHistory(
						now,
					),

					MemoryUsageBytes: buildEngineMemoryHistory(
						now,
					),
				},
			},
		)

	pricingProvider :=
		&testPricingProvider{
			prices: pricing.ResourcePricing{
				CPUPerCoreHour:  0.04,
				MemoryPerGBHour: 0.005,
			},
		}

	calculator :=
		cost.NewCalculator(730)

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

	options :=
		AnalysisOptions{
			Start: now.Add(
				-7 * 24 * time.Hour,
			),

			End: now,

			Step: 5 * time.Minute,
		}

	// Act

	result, err :=
		engine.Analyze(
			context.Background(),
			options,
		)

	// Assert

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

	if result.Summary.
		CurrentMonthlyCostUSD <= 0 {

		t.Fatalf(
			"expected current monthly cost greater than zero",
		)
	}

	if result.Summary.
		OptimizedMonthlyCostUSD >=
		result.Summary.CurrentMonthlyCostUSD {

		t.Fatalf(
			"expected optimized monthly cost to be lower",
		)
	}

	if result.Summary.
		PotentialSavingsUSD <= 0 {

		t.Fatalf(
			"expected potential savings greater than zero",
		)
	}
}

func buildEngineCPUHistory(
	now time.Time,
) []domain.MetricSample {

	values :=
		[]float64{
			100,
			120,
			150,
			180,
			220,
		}

	result :=
		make(
			[]domain.MetricSample,
			0,
			2016,
		)

	for i := 0; i < 2016; i++ {

		result =
			append(
				result,
				domain.MetricSample{
					Timestamp: now.Add(
						-time.Duration(
							2016-i,
						) * 5 * time.Minute,
					),

					Value: values[i%len(values)],
				},
			)
	}

	return result
}

func buildEngineMemoryHistory(
	now time.Time,
) []domain.MetricSample {

	values :=
		[]float64{
			500 * 1024 * 1024,
			550 * 1024 * 1024,
			600 * 1024 * 1024,
			650 * 1024 * 1024,
			700 * 1024 * 1024,
		}

	result :=
		make(
			[]domain.MetricSample,
			0,
			2016,
		)

	for i := 0; i < 2016; i++ {

		result =
			append(
				result,
				domain.MetricSample{
					Timestamp: now.Add(
						-time.Duration(
							2016-i,
						) * 5 * time.Minute,
					),

					Value: values[i%len(values)],
				},
			)
	}

	return result
}
