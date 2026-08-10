package rules

import (
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func TestMemoryOverprovisioningRule_WhenMemoryUtilizationIsBelowThreshold_ShouldRecommendOptimization(t *testing.T) {
	// Arrange
	rule := NewMemoryOverprovisioningRule()

	workload := domain.WorkloadMetrics{
		Namespace:          "payments",
		Name:               "payments-api",
		Replicas:           3,
		MemoryRequestBytes: 2 * 1024 * 1024 * 1024,
		MemoryUsageBytes:   640 * 1024 * 1024,
	}

	// Act
	result := rule.Evaluate(workload)

	// Assert
	if result == nil {
		t.Fatal("expected a recommendation, got nil")
	}

	if result.Rule != "MEMORY_OVERPROVISIONING" {
		t.Errorf(
			"expected rule MEMORY_OVERPROVISIONING, got %s",
			result.Rule,
		)
	}

	if result.Severity != domain.SeverityWarning {
		t.Errorf(
			"expected severity %s, got %s",
			domain.SeverityWarning,
			result.Severity,
		)
	}
}

func TestMemoryOverprovisioningRule_WhenMemoryUtilizationIsAboveThreshold_ShouldNotRecommendOptimization(t *testing.T) {
	// Arrange
	rule := NewMemoryOverprovisioningRule()

	workload := domain.WorkloadMetrics{
		Namespace:          "orders",
		Name:               "orders-api",
		Replicas:           2,
		MemoryRequestBytes: 1024 * 1024 * 1024,
		MemoryUsageBytes:   805 * 1024 * 1024,
	}

	// Act
	result := rule.Evaluate(workload)

	// Assert
	if result != nil {
		t.Fatalf(
			"expected no recommendation, got %+v",
			result,
		)
	}
}
