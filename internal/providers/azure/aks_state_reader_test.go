package azure

import (
	"context"
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func TestAKSStateReader_ShouldReadNodePoolState(t *testing.T) {
	reader := NewAKSStateReader(&fakeAKSNodePoolClient{current: 6}, "rg-finops")
	action := domain.Action{
		ID:        "action-azure-state",
		Provider:  domain.CloudProviderAzure,
		Cluster:   "aks-prod",
		NodeGroup: "system",
	}

	state, err := reader.ReadState(context.Background(), action)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if state.CurrentValue != 6 {
		t.Fatalf("expected current value 6, got %d", state.CurrentValue)
	}
}

func TestAKSStateReader_ShouldRejectWrongProvider(t *testing.T) {
	reader := NewAKSStateReader(&fakeAKSNodePoolClient{current: 6}, "rg-finops")
	action := domain.Action{
		Provider:  domain.CloudProviderGCP,
		Cluster:   "aks-prod",
		NodeGroup: "system",
	}

	if _, err := reader.ReadState(context.Background(), action); err == nil {
		t.Fatal("expected provider validation error")
	}
}
