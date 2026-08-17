package redis

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestNewFromEnv_ShouldReturnNotConfiguredWithoutAddress(t *testing.T) {
	previous := os.Getenv("REDIS_ADDR")
	_ = os.Unsetenv("REDIS_ADDR")
	defer os.Setenv("REDIS_ADDR", previous)

	client, err := NewFromEnv()
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
	if client != nil {
		t.Fatal("expected nil client when Redis is not configured")
	}
}

func TestClientMethods_ShouldFailClearlyWhenNotConfigured(t *testing.T) {
	client := &Client{}

	if err := client.Ping(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Ping, got %v", err)
	}
	if _, err := client.GetJSON(context.Background(), "key", &struct{}{}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from GetJSON, got %v", err)
	}
	if err := client.SetJSON(context.Background(), "key", struct{}{}, time.Minute); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from SetJSON, got %v", err)
	}
	if _, _, err := client.TryLock(context.Background(), "lock", time.Second); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from TryLock, got %v", err)
	}
	if err := client.Unlock(context.Background(), "lock", "token"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured from Unlock, got %v", err)
	}
}
