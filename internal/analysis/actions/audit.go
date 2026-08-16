package actions

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type AuditEventRepository interface {
	Append(event domain.AuditEvent) error
	ListByPlan(planID string) []domain.AuditEvent
	ListByExecution(executionID string) []domain.AuditEvent
}

type InMemoryAuditEventRepository struct {
	mu     sync.RWMutex
	events map[string]domain.AuditEvent
	order  []string
}

func NewInMemoryAuditEventRepository() *InMemoryAuditEventRepository {
	return &InMemoryAuditEventRepository{events: make(map[string]domain.AuditEvent)}
}

func (r *InMemoryAuditEventRepository) Append(event domain.AuditEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.events[event.ID]; exists {
		return fmt.Errorf("audit event %q already exists", event.ID)
	}
	r.events[event.ID] = cloneAuditEvent(event)
	r.order = append(r.order, event.ID)
	return nil
}

func (r *InMemoryAuditEventRepository) ListByPlan(planID string) []domain.AuditEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.AuditEvent, 0)
	for _, id := range r.order {
		event := r.events[id]
		if event.PlanID == planID {
			result = append(result, cloneAuditEvent(event))
		}
	}
	return result
}

func (r *InMemoryAuditEventRepository) ListByExecution(executionID string) []domain.AuditEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.AuditEvent, 0)
	for _, id := range r.order {
		event := r.events[id]
		if event.ExecutionID == executionID {
			result = append(result, cloneAuditEvent(event))
		}
	}
	return result
}

func cloneAuditEvent(event domain.AuditEvent) domain.AuditEvent {
	copyEvent := event
	if event.Metadata != nil {
		copyEvent.Metadata = make(map[string]string, len(event.Metadata))
		for key, value := range event.Metadata {
			copyEvent.Metadata[key] = value
		}
	}
	return copyEvent
}

type AuditService struct {
	repository AuditEventRepository
	now        func() time.Time
	actor      string
}

func NewAuditService(repository AuditEventRepository, actor string) *AuditService {
	return &AuditService{repository: repository, actor: actor, now: func() time.Time { return time.Now().UTC() }}
}

func (s *AuditService) Record(event domain.AuditEvent) error {
	if event.Actor == "" {
		event.Actor = s.actor
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = s.now().UTC()
	}
	if event.ID == "" {
		event.ID = auditEventID(event)
	}
	return s.repository.Append(event)
}

func (s *AuditService) RecordExecution(eventType domain.AuditEventType, record domain.ExecutionRecord, message string) error {
	event := domain.AuditEvent{
		PlanID: record.PlanID, ActionID: record.ActionID, ExecutionID: record.ID,
		Attempt: record.Attempt, EventType: eventType, Provider: record.Provider,
		Cluster: record.Cluster, NewState: string(record.Status), Message: message,
		Timestamp: s.now().UTC(), Actor: s.actor,
	}
	return s.Record(event)
}

func (s *AuditService) RecordRetry(record domain.ExecutionRecord, previousAttempt int, message string) error {
	event := domain.AuditEvent{
		PlanID: record.PlanID, ActionID: record.ActionID, ExecutionID: record.ID,
		Attempt: record.Attempt, EventType: domain.AuditExecutionRetried, Provider: record.Provider,
		Cluster: record.Cluster, PreviousState: string(domain.ExecutionStatusFailed),
		NewState: string(record.Status), Message: message, Timestamp: s.now().UTC(), Actor: s.actor,
		Metadata: map[string]string{"previousAttempt": fmt.Sprintf("%d", previousAttempt)},
	}
	return s.Record(event)
}

func (s *AuditService) ListByPlan(planID string) []domain.AuditEvent {
	events := s.repository.ListByPlan(planID)
	sort.SliceStable(events, func(i, j int) bool { return events[i].Timestamp.Before(events[j].Timestamp) })
	return events
}

func (s *AuditService) ListByExecution(executionID string) []domain.AuditEvent {
	events := s.repository.ListByExecution(executionID)
	sort.SliceStable(events, func(i, j int) bool { return events[i].Timestamp.Before(events[j].Timestamp) })
	return events
}

func auditEventID(event domain.AuditEvent) string {
	payload := fmt.Sprintf("%s|%s|%s|%d|%s|%s", event.PlanID, event.ActionID, event.ExecutionID, event.Attempt, event.EventType, event.Timestamp.UTC().Format(time.RFC3339Nano))
	sum := sha256.Sum256([]byte(payload))
	return "audit-" + hex.EncodeToString(sum[:8])
}
