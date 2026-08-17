package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cloud-efficiency-engine/internal/analysis/actions"
	"cloud-efficiency-engine/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repositories struct {
	Execution    *ExecutionRepository
	Audit        *AuditRepository
	Verification *VerificationRepository
	ActionPlan   *ActionPlanRepository
	Approval     *ApprovalRepository
	Recovery     *RecoveryRepository
}

func NewRepositories(pool *pgxpool.Pool) (*Repositories, error) {
	if pool == nil {
		return nil, errors.New("postgres pool must not be nil")
	}
	return &Repositories{
		Execution:    &ExecutionRepository{pool: pool},
		Audit:        &AuditRepository{pool: pool},
		Verification: &VerificationRepository{pool: pool},
		ActionPlan:   &ActionPlanRepository{pool: pool},
		Approval:     &ApprovalRepository{pool: pool},
		Recovery:     &RecoveryRepository{pool: pool},
	}, nil
}

type ExecutionRepository struct{ pool *pgxpool.Pool }

func (r *ExecutionRepository) GetByID(id string) (domain.ExecutionRecord, bool) {
	ctx, cancel := dbContext()
	defer cancel()
	var payload []byte
	if err := r.pool.QueryRow(ctx, `SELECT payload FROM execution_records WHERE id=$1`, id).Scan(&payload); err != nil {
		return domain.ExecutionRecord{}, false
	}
	var record domain.ExecutionRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return domain.ExecutionRecord{}, false
	}
	return record, true
}

func (r *ExecutionRepository) GetByIdempotencyKey(key string) (domain.ExecutionRecord, bool) {
	ctx, cancel := dbContext()
	defer cancel()
	var payload []byte
	if err := r.pool.QueryRow(ctx, `SELECT payload FROM execution_records WHERE idempotency_key=$1 ORDER BY attempt DESC LIMIT 1`, key).Scan(&payload); err != nil {
		return domain.ExecutionRecord{}, false
	}
	var record domain.ExecutionRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return domain.ExecutionRecord{}, false
	}
	return record, true
}

func (r *ExecutionRepository) ListByIdempotencyKey(key string) []domain.ExecutionRecord {
	ctx, cancel := dbContext()
	defer cancel()
	rows, err := r.pool.Query(ctx, `SELECT payload FROM execution_records WHERE idempotency_key=$1 ORDER BY attempt ASC`, key)
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := make([]domain.ExecutionRecord, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			continue
		}
		var record domain.ExecutionRecord
		if err := json.Unmarshal(payload, &record); err == nil {
			result = append(result, record)
		}
	}
	return result
}

func (r *ExecutionRepository) CreateIfAbsent(record domain.ExecutionRecord) (bool, error) {
	if err := record.Validate(); err != nil {
		return false, err
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return false, fmt.Errorf("marshal execution record: %w", err)
	}
	ctx, cancel := dbContext()
	defer cancel()
	result, err := r.pool.Exec(ctx, `INSERT INTO execution_records (id,idempotency_key,plan_id,action_id,provider,cluster,status,attempt,current_value,desired_value,started_at,completed_at,error,result,payload) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) ON CONFLICT DO NOTHING`, record.ID, record.IdempotencyKey, record.PlanID, record.ActionID, record.Provider, record.Cluster, record.Status, record.Attempt, record.CurrentValue, record.DesiredValue, record.StartedAt, record.CompletedAt, record.Error, record.Result, payload)
	if err != nil {
		return false, fmt.Errorf("insert execution record: %w", err)
	}
	if result.RowsAffected() == 1 {
		return true, nil
	}
	var existingID string
	var existingAttempt int
	if err := r.pool.QueryRow(ctx, `SELECT id,attempt FROM execution_records WHERE idempotency_key=$1 AND attempt=$2`, record.IdempotencyKey, record.Attempt).Scan(&existingID, &existingAttempt); err != nil {
		return false, fmt.Errorf("resolve existing execution record: %w", err)
	}
	if existingID != record.ID || existingAttempt != record.Attempt {
		return false, fmt.Errorf("execution attempt already exists with different identity")
	}
	return false, nil
}

func (r *ExecutionRepository) Update(record domain.ExecutionRecord) error {
	if err := record.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal execution record: %w", err)
	}
	ctx, cancel := dbContext()
	defer cancel()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin execution update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var previousStatus domain.ExecutionStatus
	var previousKey string
	var previousAttempt int
	if err := tx.QueryRow(ctx, `SELECT status,idempotency_key,attempt FROM execution_records WHERE id=$1 FOR UPDATE`, record.ID).Scan(&previousStatus, &previousKey, &previousAttempt); err != nil {
		return fmt.Errorf("load execution record: %w", err)
	}
	if previousKey != record.IdempotencyKey || previousAttempt != record.Attempt {
		return errors.New("execution record identity cannot change")
	}
	if previousStatus != record.Status && !domain.CanTransitionExecution(previousStatus, record.Status) {
		return fmt.Errorf("invalid execution transition: %s -> %s", previousStatus, record.Status)
	}
	if _, err := tx.Exec(ctx, `UPDATE execution_records SET status=$2,completed_at=$3,error=$4,result=$5,payload=$6 WHERE id=$1`, record.ID, record.Status, record.CompletedAt, record.Error, record.Result, payload); err != nil {
		return fmt.Errorf("update execution record: %w", err)
	}
	return tx.Commit(ctx)
}

var _ actions.ExecutionRecordRepository = (*ExecutionRepository)(nil)

type AuditRepository struct{ pool *pgxpool.Pool }

func (r *AuditRepository) Append(event domain.AuditEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}
	ctx, cancel := dbContext()
	defer cancel()
	_, err = r.pool.Exec(ctx, `INSERT INTO audit_events (id,plan_id,action_id,execution_id,attempt,event_type,actor,timestamp,provider,cluster,previous_state,new_state,message,metadata) VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, event.ID, event.PlanID, event.ActionID, event.ExecutionID, event.Attempt, event.EventType, event.Actor, event.Timestamp, event.Provider, event.Cluster, event.PreviousState, event.NewState, event.Message, metadata)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

func (r *AuditRepository) ListByPlan(planID string) []domain.AuditEvent {
	return r.list(`SELECT id,plan_id,action_id,COALESCE(execution_id,''),attempt,event_type,actor,timestamp,provider,cluster,previous_state,new_state,message,metadata FROM audit_events WHERE plan_id=$1 ORDER BY timestamp ASC`, planID)
}

func (r *AuditRepository) ListByExecution(executionID string) []domain.AuditEvent {
	return r.list(`SELECT id,plan_id,action_id,COALESCE(execution_id,''),attempt,event_type,actor,timestamp,provider,cluster,previous_state,new_state,message,metadata FROM audit_events WHERE execution_id=$1 ORDER BY timestamp ASC`, executionID)
}

func (r *AuditRepository) list(query, arg string) []domain.AuditEvent {
	ctx, cancel := dbContext()
	defer cancel()
	rows, err := r.pool.Query(ctx, query, arg)
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := make([]domain.AuditEvent, 0)
	for rows.Next() {
		var event domain.AuditEvent
		var metadata []byte
		if err := rows.Scan(&event.ID, &event.PlanID, &event.ActionID, &event.ExecutionID, &event.Attempt, &event.EventType, &event.Actor, &event.Timestamp, &event.Provider, &event.Cluster, &event.PreviousState, &event.NewState, &event.Message, &metadata); err != nil {
			continue
		}
		_ = json.Unmarshal(metadata, &event.Metadata)
		result = append(result, event)
	}
	return result
}

var _ actions.AuditEventRepository = (*AuditRepository)(nil)

type VerificationRepository struct{ pool *pgxpool.Pool }

func (r *VerificationRepository) Save(result domain.VerificationResult) error {
	if err := result.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal verification result: %w", err)
	}
	ctx, cancel := dbContext()
	defer cancel()
	var existingID string
	err = r.pool.QueryRow(ctx, `SELECT id FROM verification_results WHERE execution_id=$1`, result.ExecutionID).Scan(&existingID)
	if err == nil && existingID != result.ID {
		return fmt.Errorf("verification for execution %q already exists", result.ExecutionID)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("check verification result: %w", err)
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO verification_results (execution_id,id,plan_id,action_id,attempt,provider,cluster,status,expected_value,actual_value,verified_at,message,error,payload) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT (execution_id) DO UPDATE SET status=EXCLUDED.status,expected_value=EXCLUDED.expected_value,actual_value=EXCLUDED.actual_value,verified_at=EXCLUDED.verified_at,message=EXCLUDED.message,error=EXCLUDED.error,payload=EXCLUDED.payload`, result.ExecutionID, result.ID, result.PlanID, result.ActionID, result.Attempt, result.Provider, result.Cluster, result.Status, result.ExpectedValue, result.ActualValue, result.VerifiedAt, result.Message, result.Error, payload)
	if err != nil {
		return fmt.Errorf("save verification result: %w", err)
	}
	return nil
}

func (r *VerificationRepository) GetByExecutionID(executionID string) (domain.VerificationResult, bool) {
	ctx, cancel := dbContext()
	defer cancel()
	var payload []byte
	if err := r.pool.QueryRow(ctx, `SELECT payload FROM verification_results WHERE execution_id=$1`, executionID).Scan(&payload); err != nil {
		return domain.VerificationResult{}, false
	}
	var result domain.VerificationResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return domain.VerificationResult{}, false
	}
	return result, true
}

func (r *VerificationRepository) ListByPlan(planID string) []domain.VerificationResult {
	ctx, cancel := dbContext()
	defer cancel()
	rows, err := r.pool.Query(ctx, `SELECT payload FROM verification_results WHERE plan_id=$1 ORDER BY verified_at ASC`, planID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := make([]domain.VerificationResult, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			continue
		}
		var item domain.VerificationResult
		if err := json.Unmarshal(payload, &item); err == nil {
			result = append(result, item)
		}
	}
	return result
}

var _ actions.VerificationResultRepository = (*VerificationRepository)(nil)

type ActionPlanRepository struct{ pool *pgxpool.Pool }

func (r *ActionPlanRepository) GetActionPlanByID(id string) (domain.ActionPlan, bool, error) {
	ctx, cancel := dbContext()
	defer cancel()
	var payload []byte
	if err := r.pool.QueryRow(ctx, `SELECT payload FROM action_plans WHERE id=$1`, id).Scan(&payload); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ActionPlan{}, false, nil
		}
		return domain.ActionPlan{}, false, fmt.Errorf("get action plan: %w", err)
	}
	var plan domain.ActionPlan
	if err := json.Unmarshal(payload, &plan); err != nil {
		return domain.ActionPlan{}, false, fmt.Errorf("decode action plan: %w", err)
	}
	return plan, true, nil
}

func (r *ActionPlanRepository) CreateActionPlanIfAbsent(plan domain.ActionPlan) (bool, error) {
	if err := plan.Validate(); err != nil {
		return false, err
	}
	payload, err := json.Marshal(plan)
	if err != nil {
		return false, fmt.Errorf("marshal action plan: %w", err)
	}
	ctx, cancel := dbContext()
	defer cancel()
	result, err := r.pool.Exec(ctx, `INSERT INTO action_plans (id,provider,cluster,status,total_monthly_savings_usd,total_annualized_savings_usd,requires_approval,payload) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO NOTHING`, plan.ID, plan.Provider, plan.Cluster, plan.Status, plan.TotalMonthlySavingsUSD, plan.TotalAnnualizedSavingsUSD, plan.RequiresApproval, payload)
	if err != nil {
		return false, fmt.Errorf("insert action plan: %w", err)
	}
	return result.RowsAffected() == 1, nil
}

func (r *ActionPlanRepository) UpdateActionPlan(plan domain.ActionPlan) error {
	if err := plan.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("marshal action plan: %w", err)
	}
	ctx, cancel := dbContext()
	defer cancel()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin action plan update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var previous domain.ActionPlanStatus
	if err := tx.QueryRow(ctx, `SELECT status FROM action_plans WHERE id=$1 FOR UPDATE`, plan.ID).Scan(&previous); err != nil {
		return fmt.Errorf("load action plan: %w", err)
	}
	if previous != plan.Status && !domain.CanTransitionActionPlan(previous, plan.Status) {
		return fmt.Errorf("invalid action plan transition: %s -> %s", previous, plan.Status)
	}
	if _, err := tx.Exec(ctx, `UPDATE action_plans SET provider=$2,cluster=$3,status=$4,total_monthly_savings_usd=$5,total_annualized_savings_usd=$6,requires_approval=$7,payload=$8,updated_at=NOW() WHERE id=$1`, plan.ID, plan.Provider, plan.Cluster, plan.Status, plan.TotalMonthlySavingsUSD, plan.TotalAnnualizedSavingsUSD, plan.RequiresApproval, payload); err != nil {
		return fmt.Errorf("update action plan: %w", err)
	}
	return tx.Commit(ctx)
}

var _ actions.ActionPlanRepository = (*ActionPlanRepository)(nil)

type ApprovalRepository struct{ pool *pgxpool.Pool }

func (r *ApprovalRepository) GetApprovalByPlanID(planID string) (domain.ActionApproval, bool, error) {
	ctx, cancel := dbContext()
	defer cancel()
	var approval domain.ActionApproval
	if err := r.pool.QueryRow(ctx, `SELECT plan_id,approved_by,approved_at,comment FROM action_approvals WHERE plan_id=$1`, planID).Scan(&approval.PlanID, &approval.ApprovedBy, &approval.ApprovedAt, &approval.Comment); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ActionApproval{}, false, nil
		}
		return domain.ActionApproval{}, false, fmt.Errorf("get approval: %w", err)
	}
	return approval, true, nil
}

func (r *ApprovalRepository) SaveApproval(approval domain.ActionApproval) error {
	if err := approval.Validate(); err != nil {
		return err
	}
	ctx, cancel := dbContext()
	defer cancel()
	_, err := r.pool.Exec(ctx, `INSERT INTO action_approvals(plan_id,approved_by,approved_at,comment) VALUES($1,$2,$3,$4) ON CONFLICT(plan_id) DO UPDATE SET approved_by=EXCLUDED.approved_by,approved_at=EXCLUDED.approved_at,comment=EXCLUDED.comment`, approval.PlanID, approval.ApprovedBy, approval.ApprovedAt, approval.Comment)
	if err != nil {
		return fmt.Errorf("save approval: %w", err)
	}
	return nil
}

var _ actions.ActionApprovalRepository = (*ApprovalRepository)(nil)

type RecoveryRepository struct{ pool *pgxpool.Pool }

func (r *RecoveryRepository) GetRecoveryByID(id string) (domain.RecoveryAction, bool, error) {
	ctx, cancel := dbContext()
	defer cancel()
	var payload []byte
	if err := r.pool.QueryRow(ctx, `SELECT payload FROM recovery_actions WHERE id=$1`, id).Scan(&payload); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.RecoveryAction{}, false, nil
		}
		return domain.RecoveryAction{}, false, fmt.Errorf("get recovery action: %w", err)
	}
	var action domain.RecoveryAction
	if err := json.Unmarshal(payload, &action); err != nil {
		return domain.RecoveryAction{}, false, fmt.Errorf("decode recovery action: %w", err)
	}
	return action, true, nil
}

func (r *RecoveryRepository) ListRecoveryByPlan(planID string) ([]domain.RecoveryAction, error) {
	ctx, cancel := dbContext()
	defer cancel()
	rows, err := r.pool.Query(ctx, `SELECT payload FROM recovery_actions WHERE plan_id=$1 ORDER BY created_at ASC`, planID)
	if err != nil {
		return nil, fmt.Errorf("list recovery actions: %w", err)
	}
	defer rows.Close()
	result := make([]domain.RecoveryAction, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var action domain.RecoveryAction
		if err := json.Unmarshal(payload, &action); err != nil {
			return nil, fmt.Errorf("decode recovery action: %w", err)
		}
		result = append(result, action)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *RecoveryRepository) SaveRecovery(action domain.RecoveryAction) error {
	if err := action.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(action)
	if err != nil {
		return fmt.Errorf("marshal recovery action: %w", err)
	}
	ctx, cancel := dbContext()
	defer cancel()
	_, err = r.pool.Exec(ctx, `INSERT INTO recovery_actions (id,plan_id,action_id,execution_id,provider,cluster,resource,from_value,to_value,reason,status,requires_approval,created_at,payload) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT(id) DO UPDATE SET status=EXCLUDED.status,payload=EXCLUDED.payload`, action.ID, action.PlanID, action.ActionID, action.ExecutionID, action.Provider, action.Cluster, action.Resource, action.FromValue, action.ToValue, action.Reason, action.Status, action.RequiresApproval, action.CreatedAt, payload)
	if err != nil {
		return fmt.Errorf("save recovery action: %w", err)
	}
	return nil
}

var _ actions.RecoveryActionRepository = (*RecoveryRepository)(nil)

func dbContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
