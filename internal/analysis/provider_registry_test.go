package analysis

import (
	"context"
	"strings"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/analysis/optimizer"
	"cloud-efficiency-engine/internal/analysis/resolver"
	"cloud-efficiency-engine/internal/analysis/rules"
	"cloud-efficiency-engine/internal/cost"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics"
	"cloud-efficiency-engine/internal/pricing"
	providerregistry "cloud-efficiency-engine/internal/providers"
)

type testContextPricingProvider struct{}

func (
	p *testContextPricingProvider,
) GetPricing(
	ctx context.Context,
) (pricing.ResourcePricing, error) {

	return pricing.ResourcePricing{
		CPUPerCoreHour: 0.04,

		MemoryPerGBHour: 0.005,
	}, nil
}

type testRegistryMetricsProvider struct{}

func (
	p *testRegistryMetricsProvider,
) GetWorkloads(
	ctx context.Context,
	namespace string,
) ([]domain.WorkloadMetrics, error) {

	return []domain.WorkloadMetrics{}, nil
}

type testRegistryHistoricalProvider struct{}

func (
	p *testRegistryHistoricalProvider,
) GetWorkloadHistory(
	ctx context.Context,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {

	return []domain.WorkloadHistory{}, nil
}

func TestEngineWithRegistry_ShouldResolveKubernetesProvider(
	t *testing.T,
) {

	registry :=
		providerregistry.NewRegistry()

	registry.RegisterMetricsProvider(
		domain.CloudProviderKubernetes,
		func(
			analysisContext domain.AnalysisContext,
		) (
			metrics.Provider,
			metrics.HistoricalProvider,
			error,
		) {

			return &testRegistryMetricsProvider{},
				&testRegistryHistoricalProvider{},
				nil
		},
	)

	registry.RegisterPricingProvider(
		domain.CloudProviderKubernetes,
		func(
			analysisContext domain.AnalysisContext,
		) (
			pricing.Provider,
			error,
		) {

			return &testContextPricingProvider{},
				nil
		},
	)

	engine :=
		NewEngineWithRegistry(
			registry,
			[]rules.Rule{},
			optimizer.DefaultOptimizationPolicy(),
			resolver.NewResolver(),
			cost.NewCalculator(
				730,
			),
		)

	_, err :=
		engine.Analyze(
			context.Background(),
			AnalysisOptions{
				Namespace: "payments",

				Start: time.Now().UTC().
					Add(
						-1 * time.Hour,
					),

				End: time.Now().UTC(),

				Step: 5 * time.Minute,

				Context: domain.AnalysisContext{
					Provider: domain.CloudProviderKubernetes,
				},
			},
		)

	if err != nil {

		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}
}

func TestEngineWithRegistry_ShouldRejectUnregisteredProvider(
	t *testing.T,
) {

	registry :=
		providerregistry.NewRegistry()

	engine :=
		NewEngineWithRegistry(
			registry,
			nil,
			optimizer.DefaultOptimizationPolicy(),
			resolver.NewResolver(),
			cost.NewCalculator(
				730,
			),
		)

	_, err :=
		engine.Analyze(
			context.Background(),
			AnalysisOptions{
				Namespace: "payments",

				Start: time.Now().UTC().
					Add(
						-1 * time.Hour,
					),

				End: time.Now().UTC(),

				Step: 5 * time.Minute,

				Context: domain.AnalysisContext{
					Provider: domain.CloudProviderAWS,
				},
			},
		)

	if err == nil {

		t.Fatal(
			"expected error for unregistered provider",
		)
	}

	if !strings.Contains(
		err.Error(),
		"not registered",
	) {

		t.Fatalf(
			"expected provider registration error, got %v",
			err,
		)
	}
}
