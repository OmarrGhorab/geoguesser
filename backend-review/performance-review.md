# Performance Review

## Verify

- Database performance.
- Redis usage.
- Memory allocations.
- Connection pools.
- Context cancellation.
- Timeouts.
- Profiling opportunities.
- pprof.
- Concurrency safety.
- Goroutines.
- Mutex usage.
- Mutex contention.
- Channels.
- sync.Pool use.
- GC pressure.
- Memory leaks.
- Database indexes.
- Query plans.
- Cache hit rate.
- Connection exhaustion.

## Reject

- Goroutine leaks.
- Blocking operations without timeout.
- Race conditions.
- N+1 queries.
- Unbounded concurrency.
- Premature sync.Pool complexity without benchmark evidence.

## Common Findings

High: repository starts one goroutine per record without limit. Impact: large inputs can exhaust memory, DB connections, and scheduler capacity. Recommendation: use bounded worker pool with context cancellation.

