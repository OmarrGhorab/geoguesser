# Caching

## Concepts

Caching reduces latency and database load. Use cache-aside by default. Use write-through or write-behind only when consistency and durability trade-offs are understood.

## Architecture Decisions

- Cache DTOs, not GORM models.
- Use versioned keys.
- Use TTLs and invalidation.
- Keep cache behind interfaces.
- Treat cache failures as degraded mode when possible.

## Trade-offs

Cache-aside is simple but can serve stale data. Write-through improves consistency but slows writes. Write-behind improves write latency but risks data loss without durable queues.

## Anti-patterns

- Infinite TTL for mutable data.
- Caching per-user sensitive data without tenant/user keying.
- Cache invalidation scattered across handlers.
- Cache stampede on hot keys.
- Using Redis to hide bad SQL.

## Common Mistakes

- Not invalidating after write.
- Different code paths using different key formats.
- Missing negative cache for expensive misses.
- Caching errors.
- No metrics for hit/miss.

## Production Examples

Cache product list pages by filters, sort, page size, and cursor. Invalidate product keys after create/update/delete.

## Go Code Samples

```go
func cacheKey(parts ...string) string {
	return "api:v1:" + strings.Join(parts, ":")
}

func withJitter(base time.Duration) time.Duration {
	return base + time.Duration(rand.Int63n(int64(base/10)))
}
```

## Performance Considerations

Use pipelines, bounded values, TTL jitter, request coalescing, and metrics. Cache only after measuring hot paths.

## Security Considerations

Include tenant and user scope in keys. Avoid caching authorization decisions unless expiry and invalidation are safe.

## Scalability Considerations

Plan invalidation strategy before adding cache. Monitor Redis memory, evictions, hit rate, and backend fallback load.

