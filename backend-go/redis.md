# Redis

## Concepts

Use Redis for cache-aside, write-through, write-behind where justified, TTLs, cache invalidation, pub/sub, distributed locks, sessions, rate limiting, and idempotency keys.

## Architecture Decisions

- Wrap Redis behind explicit interfaces.
- Set TTLs for every cache entry unless persistence is intended.
- Use cache keys with versioned prefixes.
- Use idempotency keys for payment-like or retryable operations.
- Use distributed locks sparingly and with expiry.

## Trade-offs

Redis improves latency and coordination but introduces consistency complexity. Cache invalidation and failure behavior must be explicit.

## Anti-patterns

- Redis as the source of truth for durable data.
- No TTL.
- Global key names without prefixes.
- Locks without expiry.
- Caching authorization decisions carelessly.

## Common Mistakes

- Cache stampede.
- Stale data after writes.
- Missing serialization version.
- High-cardinality pub/sub channels.
- Ignoring Redis errors silently.

## Production Examples

Use cache-aside for product reads: check Redis, load PostgreSQL on miss, marshal DTO, set TTL, invalidate after write.

## Go Code Samples

```go
func (c *Cache) GetUser(ctx context.Context, id uuid.UUID) (*UserDTO, error) {
	key := "v1:user:" + id.String()
	val, err := c.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, fmt.Errorf("redis get user: %w", err)
	}
	var user UserDTO
	if err := json.Unmarshal([]byte(val), &user); err != nil {
		return nil, fmt.Errorf("decode cached user: %w", err)
	}
	return &user, nil
}
```

## Performance Considerations

Batch with pipelines where useful. Avoid huge values. Use TTL jitter to reduce synchronized expiration.

## Security Considerations

Use TLS/auth where needed. Do not store raw secrets in Redis. Encrypt sensitive session payloads if storing more than opaque IDs.

## Scalability Considerations

Design keys for sharding and predictable invalidation. Monitor hit rate, memory, evictions, and latency.

