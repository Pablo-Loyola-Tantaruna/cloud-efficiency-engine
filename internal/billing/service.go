package billing

import (
	"context"
	"fmt"

	"cloud-efficiency-engine/internal/domain"
)

type Service struct {
	provider Provider
}

func NewService(
	provider Provider,
) *Service {

	return &Service{
		provider: provider,
	}
}

func (s *Service) GetCost(
	ctx context.Context,
	query CostQuery,
) (CostReport, error) {

	if s.provider == nil {

		return CostReport{},
			fmt.Errorf(
				"billing provider is not configured",
			)
	}

	return s.provider.GetCost(
		ctx,
		query,
	)
}

type ContextAwareService struct {
	provider Provider
}

func NewContextAwareService(
	provider Provider,
) *ContextAwareService {

	return &ContextAwareService{
		provider: provider,
	}
}

func (s *ContextAwareService) GetCost(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	query CostQuery,
) (CostReport, error) {

	if s.provider == nil {

		return CostReport{},
			fmt.Errorf(
				"billing provider is not configured",
			)
	}

	if provider, ok :=
		s.provider.(ContextAwareProvider); ok {

		return provider.GetCostWithContext(
			ctx,
			analysisContext,
			query,
		)
	}

	return s.provider.GetCost(
		ctx,
		query,
	)
}
