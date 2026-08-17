package redis

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestRedisClient_Integration_ShouldCacheAndLock(t *testing.T) {
	if os.Getenv("REDIS_ADDR") == "" {
		t.Skip("REDIS_ADDR is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	type payload struct {
		PlanID string `json:"planId"`
	}

	key := "test:cache:plan"
	if err := client.SetJSON(ctx, key, payload{PlanID: "plan-redis"}, time.Minute); err != nil {
		t.Fatal(err)
	}
	var cached payload
	found, err := client.GetJSON(ctx, key, &cached)
	if err != nil {
		t.Fatal(err)
	}
	if !found || cached.PlanID != "plan-redis" {
		t.Fatalf("unexpected cache payload: found=%v payload=%+v", found, cached)
	}

	lockKey := "test:lock:execution"
	token, locked, err := client.TryLock(ctx, lockKey, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !locked || token == "" {
		t.Fatalf("expected lock acquisition, token=%q locked=%v", token, locked)
	}

	_, secondLocked, err := client.TryLock(ctx, lockKey, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if secondLocked {
		t.Fatal("expected second lock acquisition to be rejected")
	}

	if err := client.Unlock(ctx, lockKey, token); err != nil {
		t.Fatal(err)
	}
}
