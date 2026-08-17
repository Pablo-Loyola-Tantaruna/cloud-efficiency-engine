package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var ErrNotConfigured = errors.New("redis is not configured")

// Client is the infrastructure boundary used by API/application services.
// Redis is deliberately not the source of truth for FinOps state; PostgreSQL is.
type Client struct {
	client *goredis.Client
}

func NewFromEnv() (*Client, error) {
	addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if addr == "" {
		return nil, ErrNotConfigured
	}

	db := 0
	if value := strings.TrimSpace(os.Getenv("REDIS_DB")); value != "" {
		if _, err := fmt.Sscanf(value, "%d", &db); err != nil || db < 0 {
			return nil, fmt.Errorf("invalid REDIS_DB %q", value)
		}
	}

	client := goredis.NewClient(&goredis.Options{
		Addr:         addr,
		Password:     os.Getenv("REDIS_PASSWORD"),
		DB:           db,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	return &Client{client: client}, nil
}

func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.client == nil {
		return ErrNotConfigured
	}
	return c.client.Ping(ctx).Err()
}

func (c *Client) GetJSON(ctx context.Context, key string, target any) (bool, error) {
	if c == nil || c.client == nil {
		return false, ErrNotConfigured
	}
	value, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, goredis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal([]byte(value), target); err != nil {
		return false, fmt.Errorf("decode redis value: %w", err)
	}
	return true, nil
}

func (c *Client) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return ErrNotConfigured
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode redis value: %w", err)
	}
	return c.client.Set(ctx, key, payload, ttl).Err()
}

func (c *Client) Delete(ctx context.Context, key string) error {
	if c == nil || c.client == nil {
		return ErrNotConfigured
	}
	if strings.TrimSpace(key) == "" {
		return errors.New("redis key must not be empty")
	}
	return c.client.Del(ctx, key).Err()
}

// TryLock acquires a short-lived distributed lock. The caller must retain the
// returned token and pass it to Unlock. Locks are coordination only; durable
// idempotency remains enforced by PostgreSQL.
func (c *Client) TryLock(ctx context.Context, key string, ttl time.Duration) (string, bool, error) {
	if c == nil || c.client == nil {
		return "", false, ErrNotConfigured
	}
	if strings.TrimSpace(key) == "" {
		return "", false, errors.New("redis lock key must not be empty")
	}
	if ttl <= 0 {
		return "", false, errors.New("redis lock ttl must be greater than zero")
	}
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", false, fmt.Errorf("generate redis lock token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)
	locked, err := c.client.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return "", false, err
	}
	return token, locked, nil
}

func (c *Client) Unlock(ctx context.Context, key, token string) error {
	if c == nil || c.client == nil {
		return ErrNotConfigured
	}
	const script = `
if redis.call("get", KEYS[1]) == ARGV[1] then
  return redis.call("del", KEYS[1])
end
return 0`
	return c.client.Eval(ctx, script, []string{key}, token).Err()
}
