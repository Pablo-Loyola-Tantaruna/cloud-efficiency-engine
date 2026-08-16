package providers

import (
	"context"
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

type resolverExecutor struct{}

func (resolverExecutor) Execute(context.Context, domain.Action, domain.ExecutionRecord) (domain.ExecutionResult, error) {
	return domain.ExecutionResult{}, nil
}

func TestStaticExecutorResolver_ShouldResolveRegisteredProvider(t *testing.T) {
	executor := resolverExecutor{}
	resolver := NewStaticExecutorResolver(map[domain.CloudProvider]domain.ProviderExecutor{
		domain.CloudProviderAWS: executor,
	})
	resolved, err := resolver.Resolve(domain.CloudProviderAWS)
	if err != nil {
		t.Fatal(err)
	}
	if resolved == nil {
		t.Fatal("expected executor")
	}
}

func TestStaticExecutorResolver_ShouldRejectUnknownProvider(t *testing.T) {
	resolver := NewStaticExecutorResolver(nil)
	if _, err := resolver.Resolve(domain.CloudProviderAWS); err == nil {
		t.Fatal("expected missing provider error")
	}
}
