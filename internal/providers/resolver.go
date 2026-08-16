package providers

import (
	"fmt"

	"cloud-efficiency-engine/internal/domain"
)

type StaticExecutorResolver struct {
	executors map[domain.CloudProvider]domain.ProviderExecutor
}

func NewStaticExecutorResolver(executors map[domain.CloudProvider]domain.ProviderExecutor) *StaticExecutorResolver {
	copyExecutors := make(map[domain.CloudProvider]domain.ProviderExecutor, len(executors))
	for provider, executor := range executors {
		copyExecutors[provider] = executor
	}
	return &StaticExecutorResolver{executors: copyExecutors}
}

func (r *StaticExecutorResolver) Resolve(provider domain.CloudProvider) (domain.ProviderExecutor, error) {
	if r == nil {
		return nil, fmt.Errorf("executor resolver must not be nil")
	}
	executor, ok := r.executors[provider]
	if !ok || executor == nil {
		return nil, fmt.Errorf("no executor registered for provider %q", provider)
	}
	return executor, nil
}
