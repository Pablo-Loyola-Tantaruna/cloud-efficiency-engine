package aws

import (
	"context"
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func TestEKSStateReader_ShouldReadDesiredSize(t *testing.T) {
	client := &fakeEKSClient{observed: 6}
	reader := NewEKSStateReader(client)
	state, err := reader.ReadState(context.Background(), domain.Action{
		ID: "a1", Type: domain.ActionReduceNodeGroup, Provider: domain.CloudProviderAWS,
		Cluster: "production", NodeGroup: "workers", CurrentValue: 8, DesiredValue: 6,
		MonthlySavingsUSD: 1, AnnualizedSavingsUSD: 12, Risk: domain.ActionRiskLow, RequiresApproval: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if state.CurrentValue != 6 {
		t.Fatalf("expected 6, got %d", state.CurrentValue)
	}
}

type fakeEKSClient struct {
	observed int64
}

func (f *fakeEKSClient) DescribeDesiredSize(context.Context, string, string) (int64, error) {
	return f.observed, nil
}
func (f *fakeEKSClient) UpdateDesiredSize(context.Context, string, string, int64, string) (string, error) {
	return "update-1", nil
}
func (f *fakeEKSClient) WaitForUpdate(context.Context, string, string, string) error { return nil }
