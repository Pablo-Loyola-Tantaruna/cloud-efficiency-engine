package gcp

import (
	"context"
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func TestGKEStateReader_ShouldReadNodePoolState(t *testing.T) {
	reader := NewGKEStateReader(&fakeGKENodePoolClient{current: 6}, "gcp-project", "us-central1")
	action := domain.Action{
		ID:        "action-gcp-state",
		Provider:  domain.CloudProviderGCP,
		Cluster:   "gke-prod",
		NodeGroup: "default-pool",
	}

	state, err := reader.ReadState(context.Background(), action)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if state.CurrentValue != 6 {
		t.Fatalf("expected current value 6, got %d", state.CurrentValue)
	}
}

func TestGKEStateReader_ShouldRejectWrongProvider(t *testing.T) {
	reader := NewGKEStateReader(&fakeGKENodePoolClient{current: 6}, "gcp-project", "us-central1")
	action := domain.Action{
		Provider:  domain.CloudProviderAzure,
		Cluster:   "gke-prod",
		NodeGroup: "default-pool",
	}

	if _, err := reader.ReadState(context.Background(), action); err == nil {
		t.Fatal("expected provider validation error")
	}
}
