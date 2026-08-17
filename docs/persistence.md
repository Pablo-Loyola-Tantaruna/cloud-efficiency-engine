# Persistence

`feature/finops-persistence` adds PostgreSQL as the durable store for FinOps action workflows.

## Configuration

Set:

```bash
DATABASE_URL=postgres://postgres:postgres@localhost:5432/finops?sslmode=disable
DB_MAX_CONNS=10
DB_MIN_CONNS=2
```

When `DATABASE_URL` is configured, the API creates a PostgreSQL pool, waits for connectivity, and applies embedded migrations before starting the HTTP server. Without `DATABASE_URL`, the existing in-memory application behavior remains available for local development.

## Local development

```bash
docker compose up -d postgres prometheus
```

The default local database is:

```text
postgres://postgres:postgres@localhost:5432/finops?sslmode=disable
```

## Schema

The initial migration persists:

- action plans and approvals;
- execution records and attempts;
- audit events;
- verification results;
- recovery actions.

JSONB snapshots preserve the complete domain representation while indexed columns support common lookup paths such as plan, execution, idempotency key, provider, status, and time.

## Concurrency guarantees

Execution creation uses a database uniqueness constraint on `(idempotency_key, attempt)` plus `INSERT ... ON CONFLICT DO NOTHING`. Lifecycle updates use `SELECT ... FOR UPDATE` and the same domain transition rules used by the in-memory implementation.
