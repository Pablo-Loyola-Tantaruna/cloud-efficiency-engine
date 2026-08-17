CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS action_plans (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    cluster TEXT NOT NULL,
    status TEXT NOT NULL,
    total_monthly_savings_usd DOUBLE PRECISION NOT NULL,
    total_annualized_savings_usd DOUBLE PRECISION NOT NULL,
    requires_approval BOOLEAN NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_action_plans_provider_cluster
    ON action_plans(provider, cluster);
CREATE INDEX IF NOT EXISTS idx_action_plans_status
    ON action_plans(status);

CREATE TABLE IF NOT EXISTS action_approvals (
    plan_id TEXT PRIMARY KEY REFERENCES action_plans(id) ON DELETE CASCADE,
    approved_by TEXT NOT NULL,
    approved_at TIMESTAMPTZ NOT NULL,
    comment TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS execution_records (
    id TEXT PRIMARY KEY,
    idempotency_key TEXT NOT NULL,
    plan_id TEXT NOT NULL REFERENCES action_plans(id) ON DELETE RESTRICT,
    action_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    cluster TEXT NOT NULL,
    status TEXT NOT NULL,
    attempt INTEGER NOT NULL CHECK (attempt > 0),
    current_value BIGINT NOT NULL CHECK (current_value > 0),
    desired_value BIGINT NOT NULL CHECK (desired_value > 0),
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ NULL,
    error TEXT NOT NULL DEFAULT '',
    result TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL,
    UNIQUE(idempotency_key, attempt)
);

CREATE INDEX IF NOT EXISTS idx_execution_records_idempotency
    ON execution_records(idempotency_key, attempt DESC);
CREATE INDEX IF NOT EXISTS idx_execution_records_plan
    ON execution_records(plan_id, attempt DESC);
CREATE INDEX IF NOT EXISTS idx_execution_records_status
    ON execution_records(status);

CREATE TABLE IF NOT EXISTS audit_events (
    id TEXT PRIMARY KEY,
    plan_id TEXT NOT NULL,
    action_id TEXT NOT NULL,
    execution_id TEXT NULL,
    attempt INTEGER NOT NULL DEFAULT 0,
    event_type TEXT NOT NULL,
    actor TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    provider TEXT NOT NULL,
    cluster TEXT NOT NULL,
    previous_state TEXT NOT NULL DEFAULT '',
    new_state TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_audit_events_plan_time
    ON audit_events(plan_id, timestamp ASC);
CREATE INDEX IF NOT EXISTS idx_audit_events_execution_time
    ON audit_events(execution_id, timestamp ASC);

CREATE TABLE IF NOT EXISTS verification_results (
    execution_id TEXT PRIMARY KEY REFERENCES execution_records(id) ON DELETE CASCADE,
    id TEXT NOT NULL UNIQUE,
    plan_id TEXT NOT NULL,
    action_id TEXT NOT NULL,
    attempt INTEGER NOT NULL,
    provider TEXT NOT NULL,
    cluster TEXT NOT NULL,
    status TEXT NOT NULL,
    expected_value BIGINT NOT NULL,
    actual_value BIGINT NOT NULL,
    verified_at TIMESTAMPTZ NOT NULL,
    message TEXT NOT NULL,
    error TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_verification_results_plan
    ON verification_results(plan_id, verified_at ASC);
CREATE INDEX IF NOT EXISTS idx_verification_results_status
    ON verification_results(status);

CREATE TABLE IF NOT EXISTS recovery_actions (
    id TEXT PRIMARY KEY,
    plan_id TEXT NOT NULL,
    action_id TEXT NOT NULL,
    execution_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    cluster TEXT NOT NULL,
    resource TEXT NOT NULL,
    from_value BIGINT NOT NULL,
    to_value BIGINT NOT NULL,
    reason TEXT NOT NULL,
    status TEXT NOT NULL,
    requires_approval BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_recovery_actions_plan
    ON recovery_actions(plan_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_recovery_actions_execution
    ON recovery_actions(execution_id);
