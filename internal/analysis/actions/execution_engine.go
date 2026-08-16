package actions

import (
	"context"
	"fmt"

	"cloud-efficiency-engine/internal/domain"
)

type ExecutionEngine struct {
	executions *ExecutionService
	resolver   domain.ExecutorResolver
	verifier   *VerificationService
}

func NewExecutionEngine(executions *ExecutionService, resolver domain.ExecutorResolver, verifier *VerificationService) *ExecutionEngine {
	return &ExecutionEngine{executions: executions, resolver: resolver, verifier: verifier}
}

func (e *ExecutionEngine) Execute(ctx context.Context, plan domain.ActionPlan, action domain.Action) (domain.ExecutionRecord, *domain.VerificationResult, error) {
	if e.executions == nil {
		return domain.ExecutionRecord{}, nil, fmt.Errorf("execution service must not be nil")
	}
	if e.resolver == nil {
		return domain.ExecutionRecord{}, nil, fmt.Errorf("executor resolver must not be nil")
	}
	if err := plan.Validate(); err != nil {
		return domain.ExecutionRecord{}, nil, err
	}
	if err := action.Validate(); err != nil {
		return domain.ExecutionRecord{}, nil, err
	}

	record, created, err := e.executions.Start(plan, action)
	if err != nil {
		return domain.ExecutionRecord{}, nil, err
	}
	if !created && record.Status == domain.ExecutionStatusSucceeded {
		if e.verifier == nil {
			return record, nil, nil
		}
		verification, verifyErr := e.verifier.Verify(ctx, record, action)
		return record, &verification, verifyErr
	}
	if record.Status == domain.ExecutionStatusFailed || record.Status == domain.ExecutionStatusSkipped {
		return domain.ExecutionRecord{}, nil, fmt.Errorf("execution %q requires explicit retry", record.ID)
	}

	record, err = e.executions.MarkRunning(record)
	if err != nil {
		return domain.ExecutionRecord{}, nil, err
	}

	executor, err := e.resolver.Resolve(action.Provider)
	if err != nil {
		failed, failErr := e.executions.Fail(record, err)
		if failErr != nil {
			return domain.ExecutionRecord{}, nil, fmt.Errorf("resolve executor: %v; fail execution: %w", err, failErr)
		}
		return failed, nil, err
	}

	result, err := executor.Execute(ctx, action, record)
	if err != nil {
		failed, failErr := e.executions.Fail(record, err)
		if failErr != nil {
			return domain.ExecutionRecord{}, nil, fmt.Errorf("execute action: %v; fail execution: %w", err, failErr)
		}
		return failed, nil, err
	}
	if result.ExecutionID != record.ID || result.ActionID != record.ActionID || result.Provider != record.Provider || result.Cluster != record.Cluster {
		validationErr := fmt.Errorf("executor returned a result that does not match execution %q", record.ID)
		failed, failErr := e.executions.Fail(record, validationErr)
		if failErr != nil {
			return domain.ExecutionRecord{}, nil, fmt.Errorf("validate execution result: %v; fail execution: %w", validationErr, failErr)
		}
		return failed, nil, validationErr
	}
	if result.Status != domain.ExecutionResultSucceeded {
		executionErr := fmt.Errorf("executor returned unsuccessful result: %s", result.Status)
		failed, failErr := e.executions.Fail(record, executionErr)
		if failErr != nil {
			return domain.ExecutionRecord{}, nil, fmt.Errorf("handle unsuccessful result: %v; fail execution: %w", executionErr, failErr)
		}
		return failed, nil, executionErr
	}

	record, err = e.executions.Complete(record, result.Message)
	if err != nil {
		return domain.ExecutionRecord{}, nil, err
	}
	if e.verifier == nil {
		return record, nil, nil
	}
	verification, verifyErr := e.verifier.Verify(ctx, record, action)
	return record, &verification, verifyErr
}
