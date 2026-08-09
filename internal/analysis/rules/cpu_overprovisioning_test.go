package rules

import (
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func TestCPUOverprovisioningRule_WhenCPUUtilizationIsBelowThreshold_ShouldRecommendOptimization(t *testing.T) {
	// Arrange
	rule := NewCPUOverprovisioningRule()

	workload := domain.WorkloadMetrics{
		Namespace:            "payments",
		Name:                 "payments-api",
		Replicas:             3,
		CPURequestMillicores: 1000,
		CPUUsageMillicores:   180,
		MemoryRequestBytes:   2 * 1024 * 1024 * 1024,
		MemoryUsageBytes:     640 * 1024 * 1024,
	}

	// Act
	result := rule.Evaluate(workload)

	// Assert
	if result == nil {
		t.Fatal("expected a recommendation, got nil")
	}

	if result.Rule != "CPU_OVERPROVISIONING" {
		t.Errorf(
			"expected rule CPU_OVERPROVISIONING, got %s",
			result.Rule,
		)
	}

	if result.Severity != SeverityWarning {
		t.Errorf(
			"expected severity %s, got %s",
			SeverityWarning,
			result.Severity,
		)
	}

	if result.Confidence != ConfidenceHigh {
		t.Errorf(
			"expected confidence %s, got %s",
			ConfidenceHigh,
			result.Confidence,
		)
	}
}

func TestCPUOverprovisioningRule_WhenCPUUtilizationIsAboveThreshold_ShouldNotRecommendOptimization(t *testing.T) {
	// Arrange
	rule := NewCPUOverprovisioningRule()

	workload := domain.WorkloadMetrics{
		Namespace:            "orders",
		Name:                 "orders-api",
		Replicas:             2,
		CPURequestMillicores: 500,
		CPUUsageMillicores:   350,
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
