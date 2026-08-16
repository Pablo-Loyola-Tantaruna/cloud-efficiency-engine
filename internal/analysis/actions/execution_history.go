package actions

import (
	"fmt"
	"sort"

	"cloud-efficiency-engine/internal/domain"
)

type ExecutionHistoryService struct {
	repository ExecutionRecordRepository
}

func NewExecutionHistoryService(repository ExecutionRecordRepository) *ExecutionHistoryService {
	return &ExecutionHistoryService{repository: repository}
}

func (s *ExecutionHistoryService) Get(planID, actionID string) (domain.ExecutionHistory, error) {
	if planID == "" {
		return domain.ExecutionHistory{}, fmt.Errorf("plan id must not be empty")
	}
	if actionID == "" {
		return domain.ExecutionHistory{}, fmt.Errorf("action id must not be empty")
	}
	key := BuildIdempotencyKey(planID, actionID)
	records := s.repository.ListByIdempotencyKey(key)
	if len(records) == 0 {
		return domain.ExecutionHistory{}, fmt.Errorf("no execution history found for plan %q and action %q", planID, actionID)
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Attempt == records[j].Attempt {
			return records[i].StartedAt.Before(records[j].StartedAt)
		}
		return records[i].Attempt < records[j].Attempt
	})
	return domain.NewExecutionHistory(records), nil
}
