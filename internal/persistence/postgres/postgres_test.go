package postgres

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestConfigFromEnv_ShouldRequireDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("expected DATABASE_URL validation error")
	}
}

func TestConfigFromEnv_ShouldParsePoolLimits(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/finops_test?sslmode=disable")
	t.Setenv("DB_MAX_CONNS", "12")
	t.Setenv("DB_MIN_CONNS", "3")

	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.MaxConns != 12 || cfg.MinConns != 3 {
		t.Fatalf("unexpected pool limits: max=%d min=%d", cfg.MaxConns, cfg.MinConns)
	}
}

func TestApplyMigrations_ShouldSkipAlreadyAppliedVersions(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	pool, err := NewPool(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	if err := ApplyMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := ApplyMigrations(ctx, pool); err != nil {
		t.Fatalf("second migration run should be idempotent: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version='001_finops.sql'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one recorded migration, got %d", count)
	}
}
