package actions

import (
	"fmt"
	"sync"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type ExecutionRecordRepository interface {
	GetByID(id string) (domain.ExecutionRecord, bool)
	GetByIdempotencyKey(key string) (domain.ExecutionRecord, bool)
	ListByIdempotencyKey(key string) []domain.ExecutionRecord
	CreateIfAbsent(record domain.ExecutionRecord) (bool, error)
	Update(record domain.ExecutionRecord) error
}

type InMemoryExecutionRecordStore struct {
	mu      sync.RWMutex
	records map[string]domain.ExecutionRecord
	latest  map[string]string
}

func NewInMemoryExecutionRecordStore() *InMemoryExecutionRecordStore {
	return &InMemoryExecutionRecordStore{records: make(map[string]domain.ExecutionRecord), latest: make(map[string]string)}
}

func (s *InMemoryExecutionRecordStore) GetByID(id string) (domain.ExecutionRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[id]
	return record, ok
}

func (s *InMemoryExecutionRecordStore) GetByIdempotencyKey(key string) (domain.ExecutionRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.latest[key]
	if !ok {
		return domain.ExecutionRecord{}, false
	}
	record, ok := s.records[id]
	return record, ok
}

func (s *InMemoryExecutionRecordStore) ListByIdempotencyKey(key string) []domain.ExecutionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.ExecutionRecord, 0)
	for _, record := range s.records {
		if record.IdempotencyKey == key {
			result = append(result, record)
		}
	}
	return result
}

func (s *InMemoryExecutionRecordStore) CreateIfAbsent(record domain.ExecutionRecord) (bool, error) {
	if err := record.Validate(); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.records[record.ID]; ok {
		if existing.IdempotencyKey != record.IdempotencyKey || existing.Attempt != record.Attempt {
			return false, fmt.Errorf("execution record id %q already exists with different identity", record.ID)
		}
		return false, nil
	}
	s.records[record.ID] = record
	latestID, ok := s.latest[record.IdempotencyKey]
	if !ok || s.records[latestID].Attempt < record.Attempt {
		s.latest[record.IdempotencyKey] = record.ID
	}
	return true, nil
}

func (s *InMemoryExecutionRecordStore) Update(record domain.ExecutionRecord) error {
	if err := record.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.records[record.ID]
	if !ok {
		return fmt.Errorf("execution record %q not found", record.ID)
	}
	if existing.IdempotencyKey != record.IdempotencyKey || existing.Attempt != record.Attempt {
		return fmt.Errorf("execution record %q identity cannot change", record.ID)
	}
	if existing.Status != record.Status && !domain.CanTransitionExecution(existing.Status, record.Status) {
		return fmt.Errorf("invalid execution transition: %s -> %s", existing.Status, record.Status)
	}
	s.records[record.ID] = record
	latestID, ok := s.latest[record.IdempotencyKey]
	if !ok || s.records[latestID].Attempt <= record.Attempt {
		s.latest[record.IdempotencyKey] = record.ID
	}
	return nil
}

func (s *InMemoryExecutionRecordStore) Get(idempotencyKey string) (domain.ExecutionRecord, bool) {
	return s.GetByIdempotencyKey(idempotencyKey)
}
func (s *InMemoryExecutionRecordStore) Put(record domain.ExecutionRecord) error {
	if _, ok := s.GetByID(record.ID); ok {
		return s.Update(record)
	}
	_, err := s.CreateIfAbsent(record)
	return err
}

type ExecutionRecordStore = ExecutionRecordRepository

type ExecutionService struct {
	repository ExecutionRecordRepository
	now        func() time.Time
}

func NewExecutionService(repository ExecutionRecordRepository) *ExecutionService {
	return &ExecutionService{repository: repository, now: func() time.Time { return time.Now().UTC() }}
}

func (s *ExecutionService) Start(plan domain.ActionPlan, action domain.Action) (domain.ExecutionRecord, bool, error) {
	record, err := NewExecutionRecord(plan, action, s.now())
	if err != nil {
		return domain.ExecutionRecord{}, false, err
	}
	if existing, ok := s.repository.GetByIdempotencyKey(record.IdempotencyKey); ok {
		switch existing.Status {
		case domain.ExecutionStatusSucceeded, domain.ExecutionStatusRunning, domain.ExecutionStatusPending:
			return existing, false, nil
		default:
			return domain.ExecutionRecord{}, false, fmt.Errorf("execution %q is already %s; explicit retry is required", existing.ID, existing.Status)
		}
	}
	created, err := s.repository.CreateIfAbsent(record)
	if err != nil {
		return domain.ExecutionRecord{}, false, err
	}
	if !created {
		existing, ok := s.repository.GetByIdempotencyKey(record.IdempotencyKey)
		if !ok {
			return domain.ExecutionRecord{}, false, fmt.Errorf("execution record was not created and no existing record was found")
		}
		return existing, false, nil
	}
	return record, true, nil
}

func (s *ExecutionService) MarkRunning(record domain.ExecutionRecord) (domain.ExecutionRecord, error) {
	updated, err := TransitionExecution(record, domain.ExecutionStatusRunning, s.now(), nil)
	if err != nil {
		return domain.ExecutionRecord{}, err
	}
	if err := s.repository.Update(updated); err != nil {
		return domain.ExecutionRecord{}, err
	}
	return updated, nil
}

func (s *ExecutionService) Complete(record domain.ExecutionRecord, result string) (domain.ExecutionRecord, error) {
	updated, err := TransitionExecution(record, domain.ExecutionStatusSucceeded, s.now(), nil)
	if err != nil {
		return domain.ExecutionRecord{}, err
	}
	updated.Result = result
	if err := updated.Validate(); err != nil {
		return domain.ExecutionRecord{}, err
	}
	if err := s.repository.Update(updated); err != nil {
		return domain.ExecutionRecord{}, err
	}
	return updated, nil
}

func (s *ExecutionService) Fail(record domain.ExecutionRecord, executionErr error) (domain.ExecutionRecord, error) {
	updated, err := TransitionExecution(record, domain.ExecutionStatusFailed, s.now(), executionErr)
	if err != nil {
		return domain.ExecutionRecord{}, err
	}
	if err := s.repository.Update(updated); err != nil {
		return domain.ExecutionRecord{}, err
	}
	return updated, nil
}

func (s *ExecutionService) Skip(record domain.ExecutionRecord, result string) (domain.ExecutionRecord, error) {
	updated, err := TransitionExecution(record, domain.ExecutionStatusSkipped, s.now(), nil)
	if err != nil {
		return domain.ExecutionRecord{}, err
	}
	updated.Result = result
	if err := updated.Validate(); err != nil {
		return domain.ExecutionRecord{}, err
	}
	if err := s.repository.Update(updated); err != nil {
		return domain.ExecutionRecord{}, err
	}
	return updated, nil
}

func (s *ExecutionService) Retry(plan domain.ActionPlan, action domain.Action) (domain.ExecutionRecord, bool, error) {
	if err := plan.Validate(); err != nil {
		return domain.ExecutionRecord{}, false, err
	}
	if err := action.Validate(); err != nil {
		return domain.ExecutionRecord{}, false, err
	}
	if plan.Status != domain.ActionPlanReadyToApply {
		return domain.ExecutionRecord{}, false, fmt.Errorf("action plan %q must be READY_TO_APPLY", plan.ID)
	}
	if action.Provider != plan.Provider || action.Cluster != plan.Cluster {
		return domain.ExecutionRecord{}, false, fmt.Errorf("action %q does not belong to plan %q", action.ID, plan.ID)
	}
	key := BuildIdempotencyKey(plan.ID, action.ID)
	existing, ok := s.repository.GetByIdempotencyKey(key)
	if !ok {
		return domain.ExecutionRecord{}, false, fmt.Errorf("execution for plan %q and action %q does not exist", plan.ID, action.ID)
	}
	if existing.Status != domain.ExecutionStatusFailed {
		return domain.ExecutionRecord{}, false, fmt.Errorf("execution %q must be FAILED before retry", existing.ID)
	}
	retry, err := RetryExecutionRecord(existing, s.now())
	if err != nil {
		return domain.ExecutionRecord{}, false, err
	}
	created, err := s.repository.CreateIfAbsent(retry)
	if err != nil {
		return domain.ExecutionRecord{}, false, err
	}
	if !created {
		latest, ok := s.repository.GetByIdempotencyKey(key)
		if !ok {
			return domain.ExecutionRecord{}, false, fmt.Errorf("retry was not created and no latest execution exists")
		}
		return latest, false, nil
	}
	return retry, true, nil
}

func (s *ExecutionService) IsAlreadyExecuted(planID, actionID string) bool {
	record, ok := s.repository.GetByIdempotencyKey(BuildIdempotencyKey(planID, actionID))
	return ok && record.Status == domain.ExecutionStatusSucceeded
}

func StartExecutionRecord(store ExecutionRecordStore, plan domain.ActionPlan, actionID string, now time.Time) (domain.ExecutionRecord, bool, error) {
	var action *domain.Action
	for i := range plan.Actions {
		if plan.Actions[i].ID == actionID {
			action = &plan.Actions[i]
			break
		}
	}
	if action == nil {
		return domain.ExecutionRecord{}, false, fmt.Errorf("action %q does not belong to plan %q", actionID, plan.ID)
	}
	service := NewExecutionService(store)
	service.now = func() time.Time { return now }
	return service.Start(plan, *action)
}

func MarkExecutionRunning(store ExecutionRecordStore, record domain.ExecutionRecord) (domain.ExecutionRecord, error) {
	service := NewExecutionService(store)
	service.now = func() time.Time { return record.StartedAt }
	return service.MarkRunning(record)
}

func FinishExecution(store ExecutionRecordStore, record domain.ExecutionRecord, status domain.ExecutionStatus, executionErr error, completedAt time.Time) (domain.ExecutionRecord, error) {
	if status != domain.ExecutionStatusSucceeded && status != domain.ExecutionStatusFailed && status != domain.ExecutionStatusSkipped {
		return domain.ExecutionRecord{}, fmt.Errorf("terminal status must be SUCCEEDED, FAILED or SKIPPED")
	}
	if record.Status != domain.ExecutionStatusRunning {
		return domain.ExecutionRecord{}, fmt.Errorf("execution %q is not RUNNING", record.ID)
	}
	updated, err := TransitionExecution(record, status, completedAt, executionErr)
	if err != nil {
		return domain.ExecutionRecord{}, err
	}
	if err := store.Update(updated); err != nil {
		return domain.ExecutionRecord{}, err
	}
	return updated, nil
}
