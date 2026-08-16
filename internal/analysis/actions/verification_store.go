package actions

import (
	"fmt"
	"sort"
	"sync"

	"cloud-efficiency-engine/internal/domain"
)

type VerificationResultRepository interface {
	Save(result domain.VerificationResult) error
	GetByExecutionID(executionID string) (domain.VerificationResult, bool)
	ListByPlan(planID string) []domain.VerificationResult
}

type InMemoryVerificationResultRepository struct {
	mu      sync.RWMutex
	results map[string]domain.VerificationResult
}

func NewInMemoryVerificationResultRepository() *InMemoryVerificationResultRepository {
	return &InMemoryVerificationResultRepository{results: make(map[string]domain.VerificationResult)}
}

func (r *InMemoryVerificationResultRepository) Save(result domain.VerificationResult) error {
	if err := result.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.results[result.ExecutionID]; ok && existing.ID != result.ID {
		return fmt.Errorf("verification for execution %q already exists", result.ExecutionID)
	}
	r.results[result.ExecutionID] = result
	return nil
}

func (r *InMemoryVerificationResultRepository) GetByExecutionID(executionID string) (domain.VerificationResult, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result, ok := r.results[executionID]
	return result, ok
}

func (r *InMemoryVerificationResultRepository) ListByPlan(planID string) []domain.VerificationResult {
	r.mu.RLock()
	defer r.mu.RUnlock()
	results := make([]domain.VerificationResult, 0)
	for _, result := range r.results {
		if result.PlanID == planID {
			results = append(results, result)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].VerifiedAt.Before(results[j].VerifiedAt)
	})
	return results
}
