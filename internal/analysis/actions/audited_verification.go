package actions

import (
	"context"
	"fmt"

	"cloud-efficiency-engine/internal/domain"
)

type AuditedVerificationService struct {
	verification *VerificationService
	repository   VerificationResultRepository
	audit        *AuditService
}

func NewAuditedVerificationService(
	verification *VerificationService,
	repository VerificationResultRepository,
	audit *AuditService,
) *AuditedVerificationService {
	return &AuditedVerificationService{
		verification: verification,
		repository:   repository,
		audit:        audit,
	}
}

func (s *AuditedVerificationService) Verify(ctx context.Context, execution domain.ExecutionRecord, action domain.Action) (domain.VerificationResult, error) {
	if s.verification == nil || s.repository == nil || s.audit == nil {
		return domain.VerificationResult{}, fmt.Errorf("audited verification dependencies must not be nil")
	}
	result, err := s.verification.Verify(ctx, execution, action)
	if err != nil {
		return domain.VerificationResult{}, err
	}
	if err := s.repository.Save(result); err != nil {
		return domain.VerificationResult{}, err
	}

	eventType := domain.AuditVerificationVerified
	switch result.Status {
	case domain.VerificationStatusDrift:
		eventType = domain.AuditVerificationDrift
	case domain.VerificationStatusFailed:
		eventType = domain.AuditVerificationFailed
	}
	event := domain.AuditEvent{
		PlanID:      result.PlanID,
		ActionID:    result.ActionID,
		ExecutionID: result.ExecutionID,
		Attempt:     result.Attempt,
		EventType:   eventType,
		Provider:    result.Provider,
		Cluster:     result.Cluster,
		NewState:    string(result.Status),
		Message:     result.Message,
		Metadata: map[string]string{
			"expectedValue": fmt.Sprintf("%d", result.ExpectedValue),
			"actualValue":   fmt.Sprintf("%d", result.ActualValue),
		},
	}
	if result.Error != "" {
		event.Metadata["error"] = result.Error
	}
	if err := s.audit.Record(event); err != nil {
		return domain.VerificationResult{}, err
	}
	return result, nil
}
