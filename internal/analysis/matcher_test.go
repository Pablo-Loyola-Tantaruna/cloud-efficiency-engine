package analysis

import (
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func TestMatchHistory_ShouldReturnMatchingWorkload(
	t *testing.T,
) {

	// Arrange

	workload := domain.WorkloadMetrics{
		Namespace: "payments",
		Name:      "payments-api",
	}

	histories := []domain.WorkloadHistory{
		{
			Namespace: "orders",
			Name:      "orders-api",
		},
		{
			Namespace: "payments",
			Name:      "payments-api",
		},
	}

	// Act

	result, err := MatchHistory(
		workload,
		histories,
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
			"expected matching history",
		)
	}

	if result.Namespace != "payments" {
		t.Errorf(
			"expected payments namespace, got %s",
			result.Namespace,
		)
	}

	if result.Name != "payments-api" {
		t.Errorf(
			"expected payments-api, got %s",
			result.Name,
		)
	}
}

func TestMatchHistory_ShouldReturnErrorWhenHistoryDoesNotExist(
	t *testing.T,
) {

	// Arrange

	workload := domain.WorkloadMetrics{
		Namespace: "payments",
		Name:      "payments-api",
	}

	histories := []domain.WorkloadHistory{
		{
			Namespace: "orders",
			Name:      "orders-api",
		},
	}

	// Act

	_, err := MatchHistory(
		workload,
		histories,
	)

	// Assert

	if err == nil {
		t.Fatal(
			"expected error when history does not exist",
		)
	}
}
