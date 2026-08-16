package actions

import "cloud-efficiency-engine/internal/domain"

type AuditedExecutionService struct {
	execution *ExecutionService
	audit     *AuditService
}

func NewAuditedExecutionService(repository ExecutionRecordRepository, auditRepository AuditEventRepository, actor string) *AuditedExecutionService {
	return &AuditedExecutionService{
		execution: NewExecutionService(repository),
		audit:     NewAuditService(auditRepository, actor),
	}
}

func (s *AuditedExecutionService) Start(plan domain.ActionPlan, action domain.Action) (domain.ExecutionRecord, bool, error) {
	record, created, err := s.execution.Start(plan, action)
	if err != nil || !created {
		return record, created, err
	}
	if err := s.audit.RecordExecution(domain.AuditExecutionCreated, record, "execution record created"); err != nil {
		return domain.ExecutionRecord{}, false, err
	}
	return record, true, nil
}

func (s *AuditedExecutionService) MarkRunning(record domain.ExecutionRecord) (domain.ExecutionRecord, error) {
	updated, err := s.execution.MarkRunning(record)
	if err != nil {
		return domain.ExecutionRecord{}, err
	}
	if err := s.audit.RecordExecution(domain.AuditExecutionStarted, updated, "execution started"); err != nil {
		return domain.ExecutionRecord{}, err
	}
	return updated, nil
}

func (s *AuditedExecutionService) Complete(record domain.ExecutionRecord, result string) (domain.ExecutionRecord, error) {
	updated, err := s.execution.Complete(record, result)
	if err != nil {
		return domain.ExecutionRecord{}, err
	}
	if err := s.audit.RecordExecution(domain.AuditExecutionSucceeded, updated, result); err != nil {
		return domain.ExecutionRecord{}, err
	}
	return updated, nil
}

func (s *AuditedExecutionService) Fail(record domain.ExecutionRecord, executionErr error) (domain.ExecutionRecord, error) {
	updated, err := s.execution.Fail(record, executionErr)
	if err != nil {
		return domain.ExecutionRecord{}, err
	}
	if err := s.audit.RecordExecution(domain.AuditExecutionFailed, updated, executionErr.Error()); err != nil {
		return domain.ExecutionRecord{}, err
	}
	return updated, nil
}

func (s *AuditedExecutionService) Skip(record domain.ExecutionRecord, result string) (domain.ExecutionRecord, error) {
	updated, err := s.execution.Skip(record, result)
	if err != nil {
		return domain.ExecutionRecord{}, err
	}
	if err := s.audit.RecordExecution(domain.AuditExecutionSkipped, updated, result); err != nil {
		return domain.ExecutionRecord{}, err
	}
	return updated, nil
}

func (s *AuditedExecutionService) Retry(plan domain.ActionPlan, action domain.Action) (domain.ExecutionRecord, bool, error) {
	before, _ := s.execution.repository.GetByIdempotencyKey(BuildIdempotencyKey(plan.ID, action.ID))
	retry, created, err := s.execution.Retry(plan, action)
	if err != nil || !created {
		return retry, created, err
	}
	if err := s.audit.RecordRetry(retry, before.Attempt, "execution retry created after failure"); err != nil {
		return domain.ExecutionRecord{}, false, err
	}
	return retry, true, nil
}

func (s *AuditedExecutionService) History(planID, actionID string) (domain.ExecutionHistory, error) {
	return NewExecutionHistoryService(s.execution.repository).Get(planID, actionID)
}

func (s *AuditedExecutionService) AuditByPlan(planID string) []domain.AuditEvent {
	return s.audit.ListByPlan(planID)
}

func (s *AuditedExecutionService) AuditByExecution(executionID string) []domain.AuditEvent {
	return s.audit.ListByExecution(executionID)
}
