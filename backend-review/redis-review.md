# Redis Review

## Verify

- Caching only where appropriate.
- TTL.
- Cache invalidation.
- Session storage.
- Rate limiting.
- Distributed locks if required.
- Cache stampede protection.
- Cache penetration protection.
- Cache warming.
- Memory usage.
- Eviction policy.
- Serialization.

## Reject

- Caching everything.
- Missing invalidation.
- Infinite TTL.
- Duplicating PostgreSQL as source of truth.
- Locks without expiry.
- Ignored Redis errors on critical paths.

## Common Findings

Medium: cache key has no TTL. Impact: stale data can persist forever and memory can grow without bound. Recommendation: set bounded TTL with jitter and invalidate on writes.

