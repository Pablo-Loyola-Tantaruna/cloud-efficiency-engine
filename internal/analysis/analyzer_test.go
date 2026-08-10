package analysis

import (
	"cloud-efficiency-engine/internal/cost"
	"context"
	"testing"

	"cloud-efficiency-engine/internal/analysis/rules"
	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/metrics/providers"
)

func TestAnalyzer_Analyze_ShouldReturnRecommendationsForOverprovisionedWorkloads(t *testing.T) {
	// Arrange
	workloads := []domain.WorkloadMetrics{
		{
			Namespace:            "payments",
			Name:                 "payments-api",
			Replicas:             3,
			CPURequestMillicores: 1000,
			CPUUsageMillicores:   180,
			MemoryRequestBytes:   2 * 1024 * 1024 * 1024,
			MemoryUsageBytes:     640 * 1024 * 1024,
		},
		{
			Namespace:            "orders",
			Name:                 "orders-api",
			Replicas:             2,
			CPURequestMillicores: 500,
			CPUUsageMillicores:   350,
			MemoryRequestBytes:   1024 * 1024 * 1024,
			MemoryUsageBytes:     805 * 1024 * 1024,
		},
	}

	provider := providers.NewMockProvider(workloads)

	optimizationRules := []rules.Rule{
		rules.NewCPUOverprovisioningRule(),
		rules.NewMemoryOverprovisioningRule(),
	}

	pricing := cost.Pricing{
		CPUPerCoreHour:  0.04,
		MemoryPerGBHour: 0.005,
		HoursPerMonth:   730,
	}

	calculator := cost.NewCalculator(pricing)

	analyzer := NewAnalyzer(
		provider,
		optimizationRules,
		calculator,
	)

	// Act
	result, err := analyzer.Analyze(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.WorkloadsAnalyzed != 2 {
		t.Fatalf(
			"expected 2 workloads analyzed, got %d",
			result.WorkloadsAnalyzed,
		)
	}

	if len(result.Recommendations) != 2 {
		t.Fatalf(
			"expected 2 recommendations, got %d",
			len(result.Recommendations),
		)
	}
}
