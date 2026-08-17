package actions

import (
	"context"
	"fmt"
	"time"

	"cloud-efficiency-engine/internal/domain"
	"cloud-efficiency-engine/internal/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type ExecutionTelemetry interface {
	RecordExecution(action domain.Action, outcome string, duration time.Duration, realized bool)
	RecordVerification(provider domain.CloudProvider, outcome string, duration time.Duration)
	RecordProviderOperation(provider domain.CloudProvider, operation, outcome string, duration time.Duration)
}

type ExecutionEngine struct {
	executions *ExecutionService
	resolver   domain.ExecutorResolver
	verifier   *VerificationService
	telemetry  ExecutionTelemetry
}

func NewExecutionEngine(executions *ExecutionService, resolver domain.ExecutorResolver, verifier *VerificationService, telemetry ...ExecutionTelemetry) *ExecutionEngine {
	var runtimeTelemetry ExecutionTelemetry
	if len(telemetry) > 0 {
		runtimeTelemetry = telemetry[0]
	} else {
		runtimeTelemetry = observability.DefaultRuntimeMetrics()
	}
	return &ExecutionEngine{executions: executions, resolver: resolver, verifier: verifier, telemetry: runtimeTelemetry}
}

func (e *ExecutionEngine) Execute(ctx context.Context, plan domain.ActionPlan, action domain.Action) (domain.ExecutionRecord, *domain.VerificationResult, error) {
	tracer := otel.Tracer("cloud-efficiency-engine/actions")
	ctx, span := tracer.Start(ctx, "finops.action.execute")
	defer span.End()
	span.SetAttributes(attribute.String("finops.action.id", action.ID), attribute.String("finops.action.type", string(action.Type)), attribute.String("cloud.provider", string(action.Provider)), attribute.String("cloud.cluster", action.Cluster))

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
			e.telemetry.RecordExecution(action, "idempotent", 0, false)
			return record, nil, nil
		}
		verificationStartedAt := time.Now()
		verification, verifyErr := e.verifier.Verify(ctx, record, action)
		outcome := "success"
		if verifyErr != nil {
			outcome = "error"
		}
		e.telemetry.RecordExecution(action, "idempotent", 0, false)
		e.telemetry.RecordVerification(record.Provider, outcome, time.Since(verificationStartedAt))
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
		e.telemetry.RecordExecution(action, "resolver_error", 0, false)
		return failed, nil, err
	}

	providerStartedAt := time.Now()
	result, err := executor.Execute(ctx, action, record)
	providerOutcome := "success"
	if err != nil {
		providerOutcome = "error"
	}
	e.telemetry.RecordProviderOperation(action.Provider, "execute", providerOutcome, time.Since(providerStartedAt))
	if err != nil {
		failed, failErr := e.executions.Fail(record, err)
		if failErr != nil {
			return domain.ExecutionRecord{}, nil, fmt.Errorf("execute action: %v; fail execution: %w", err, failErr)
		}
		e.telemetry.RecordExecution(action, "error", time.Since(providerStartedAt), false)
		return failed, nil, err
	}
	if result.ExecutionID != record.ID || result.ActionID != record.ActionID || result.Provider != record.Provider || result.Cluster != record.Cluster {
		validationErr := fmt.Errorf("executor returned a result that does not match execution %q", record.ID)
		failed, failErr := e.executions.Fail(record, validationErr)
		if failErr != nil {
			return domain.ExecutionRecord{}, nil, fmt.Errorf("validate execution result: %v; fail execution: %w", validationErr, failErr)
		}
		e.telemetry.RecordExecution(action, "invalid_result", time.Since(providerStartedAt), false)
		return failed, nil, validationErr
	}
	if result.Status != domain.ExecutionResultSucceeded {
		executionErr := fmt.Errorf("executor returned unsuccessful result: %s", result.Status)
		failed, failErr := e.executions.Fail(record, executionErr)
		if failErr != nil {
			return domain.ExecutionRecord{}, nil, fmt.Errorf("handle unsuccessful result: %v; fail execution: %w", executionErr, failErr)
		}
		e.telemetry.RecordExecution(action, "unsuccessful", time.Since(providerStartedAt), false)
		return failed, nil, executionErr
	}

	record, err = e.executions.Complete(record, result.Message)
	if err != nil {
		e.telemetry.RecordExecution(action, "complete_error", time.Since(providerStartedAt), false)
		return domain.ExecutionRecord{}, nil, err
	}
	if e.verifier == nil {
		e.telemetry.RecordExecution(action, "success", time.Since(providerStartedAt), created)
		return record, nil, nil
	}

	verificationStartedAt := time.Now()
	verification, verifyErr := e.verifier.Verify(ctx, record, action)
	verificationOutcome := "success"
	if verifyErr != nil {
		verificationOutcome = "error"
	}
	e.telemetry.RecordExecution(action, "success", time.Since(providerStartedAt), created)
	e.telemetry.RecordVerification(record.Provider, verificationOutcome, time.Since(verificationStartedAt))
	return record, &verification, verifyErr
}
