package providers

import (
	"context"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type MockHistoricalProvider struct {
	histories []domain.WorkloadHistory
}

func NewMockHistoricalProvider(
	histories []domain.WorkloadHistory,
) *MockHistoricalProvider {

	return &MockHistoricalProvider{
		histories: histories,
	}
}

func (p *MockHistoricalProvider) GetWorkloadHistory(
	ctx context.Context,
	namespace string,
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {

	result :=
		make(
			[]domain.WorkloadHistory,
			0,
			len(p.histories),
		)

	for _, history := range p.histories {

		if history.Namespace != namespace {
			continue
		}

		result =
			append(
				result,
				history,
			)
	}

	return result, nil
}
