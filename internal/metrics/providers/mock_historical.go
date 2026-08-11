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
	start time.Time,
	end time.Time,
	step time.Duration,
) ([]domain.WorkloadHistory, error) {

	if end.Before(start) {
		return nil, nil
	}

	result :=
		make(
			[]domain.WorkloadHistory,
			0,
			len(p.histories),
		)

	for _, history := range p.histories {

		filtered := domain.WorkloadHistory{
			Namespace: history.Namespace,
			Name:      history.Name,
		}

		for _, sample := range history.CPUUsageMillicores {

			if !sample.Timestamp.Before(start) &&
				!sample.Timestamp.After(end) {

				filtered.CPUUsageMillicores =
					append(
						filtered.CPUUsageMillicores,
						sample,
					)
			}
		}

		for _, sample := range history.MemoryUsageBytes {

			if !sample.Timestamp.Before(start) &&
				!sample.Timestamp.After(end) {

				filtered.MemoryUsageBytes =
					append(
						filtered.MemoryUsageBytes,
						sample,
					)
			}
		}

		result = append(
			result,
			filtered,
		)
	}

	return result, nil
}
