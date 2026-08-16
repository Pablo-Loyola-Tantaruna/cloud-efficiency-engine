package actions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type VerificationService struct {
	reader domain.StateReader
	now    func() time.Time
}

func NewVerificationService(reader domain.StateReader) *VerificationService {
	return &VerificationService{
		reader: reader,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

func (s *VerificationService) Verify(ctx context.Context, execution domain.ExecutionRecord, action domain.Action) (domain.VerificationResult, error) {
	if err := execution.Validate(); err != nil {
		return domain.VerificationResult{}, err
	}
	if execution.Status != domain.ExecutionStatusSucceeded {
		return domain.VerificationResult{}, fmt.Errorf("execution %q must be SUCCEEDED before verification", execution.ID)
	}
	if err := action.Validate(); err != nil {
		return domain.VerificationResult{}, err
	}
	if action.ID != execution.ActionID || action.Provider != execution.Provider || action.Cluster != execution.Cluster {
		return domain.VerificationResult{}, fmt.Errorf("action %q does not match execution %q", action.ID, execution.ID)
	}
	if s.reader == nil {
		return domain.VerificationResult{}, fmt.Errorf("verification state reader must not be nil")
	}
	observed, err := s.reader.ReadState(ctx, action)
	if err != nil {
		return newVerificationFailure(execution, s.now(), err), nil
	}
	if observed.CurrentValue <= 0 {
		return newVerificationFailure(execution, s.now(), fmt.Errorf("observed current value must be greater than zero")), nil
	}

	now := s.now().UTC()
	result := domain.VerificationResult{
		ID: verificationResultID(execution.ID), PlanID: execution.PlanID, ActionID: execution.ActionID,
		ExecutionID: execution.ID, Attempt: execution.Attempt, Provider: execution.Provider,
		Cluster: execution.Cluster, ExpectedValue: action.DesiredValue, ActualValue: observed.CurrentValue,
		VerifiedAt: now,
	}
	if observed.CurrentValue == action.DesiredValue {
		result.Status = domain.VerificationStatusVerified
		result.Message = fmt.Sprintf("verified desired value %d", action.DesiredValue)
	} else {
		result.Status = domain.VerificationStatusDrift
		result.Message = fmt.Sprintf("expected value %d but observed %d", action.DesiredValue, observed.CurrentValue)
	}
	if err := result.Validate(); err != nil {
		return domain.VerificationResult{}, err
	}
	return result, nil
}

func newVerificationFailure(execution domain.ExecutionRecord, now time.Time, verificationErr error) domain.VerificationResult {
	return domain.VerificationResult{
		ID: verificationResultID(execution.ID), PlanID: execution.PlanID, ActionID: execution.ActionID,
		ExecutionID: execution.ID, Attempt: execution.Attempt, Provider: execution.Provider,
		Cluster: execution.Cluster, Status: domain.VerificationStatusFailed,
		ExpectedValue: execution.DesiredValue, ActualValue: execution.CurrentValue,
		VerifiedAt: now.UTC(), Message: "verification could not determine actual state", Error: verificationErr.Error(),
	}
}

func verificationResultID(executionID string) string {
	sum := sha256.Sum256([]byte(executionID))
	return "verify-" + hex.EncodeToString(sum[:8])
}
