package challenges

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// IdempotencyStore stores short-lived successful write responses for replay-safe
// challenge mutations. PostgreSQL remains the durable source of truth.
type IdempotencyStore interface {
	Get(ctx context.Context, key string) (*IdempotencyRecord, error)
	Claim(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, key string) error
	Store(ctx context.Context, key string, record IdempotencyRecord, ttl time.Duration) error
}

type IdempotencyRecord struct {
	Fingerprint string
	Payload     []byte
}

type RedisIdempotencyStore struct {
	client *redis.Client
}

func NewRedisIdempotencyStore(client *redis.Client) *RedisIdempotencyStore {
	return &RedisIdempotencyStore{client: client}
}

func (s *RedisIdempotencyStore) Get(ctx context.Context, key string) (*IdempotencyRecord, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	values, err := s.client.HMGet(ctx, key, "fingerprint", "payload").Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get challenge idempotency record: %w", err)
	}
	if len(values) != 2 || values[0] == nil || values[1] == nil {
		return nil, nil
	}
	fingerprint, ok := values[0].(string)
	if !ok {
		return nil, fmt.Errorf("unexpected challenge idempotency fingerprint type %T", values[0])
	}
	payload, ok := values[1].(string)
	if !ok {
		return nil, fmt.Errorf("unexpected challenge idempotency payload type %T", values[1])
	}
	return &IdempotencyRecord{Fingerprint: fingerprint, Payload: []byte(payload)}, nil
}

func (s *RedisIdempotencyStore) Claim(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if s == nil || s.client == nil {
		return true, nil
	}
	ok, err := s.client.SetNX(ctx, key+":lock", "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("claim challenge idempotency key: %w", err)
	}
	return ok, nil
}

func (s *RedisIdempotencyStore) Release(ctx context.Context, key string) error {
	if s == nil || s.client == nil {
		return nil
	}
	if err := s.client.Del(ctx, key+":lock").Err(); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("release challenge idempotency key: %w", err)
	}
	return nil
}

func (s *RedisIdempotencyStore) Store(ctx context.Context, key string, record IdempotencyRecord, ttl time.Duration) error {
	if s == nil || s.client == nil {
		return nil
	}
	if err := s.client.HSet(ctx, key, map[string]any{
		"fingerprint": record.Fingerprint,
		"payload":     string(record.Payload),
	}).Err(); err != nil {
		return fmt.Errorf("store challenge idempotency record: %w", err)
	}
	if err := s.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("expire challenge idempotency record: %w", err)
	}
	return nil
}
