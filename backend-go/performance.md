# Performance

## Concepts

Optimize with connection pooling, Redis caching, worker pools, goroutines, channels, context cancellation, timeouts, efficient JSON, memory allocation control, pprof, benchmarking, mutexes, sync.Pool, database indexing, query optimization, memory profiling, CPU profiling, and goroutine leak detection.

## Architecture Decisions

- Set timeouts everywhere.
- Tune database pools.
- Use caching only for measured hot paths.
- Use worker pools for bounded concurrency.
- Profile before optimizing.
- Add pprof only behind secure access.

## Trade-offs

Goroutines are cheap, not free. Caches reduce latency but add consistency risk. sync.Pool can reduce allocations but complicates code and may not help.

## Anti-patterns

- Unbounded goroutines.
- Channels where a mutex is simpler.
- Premature micro-optimization.
- No context cancellation.
- Ignoring query plans.

## Common Mistakes

- Missing HTTP timeouts.
- Connection pool too small or too large.
- Goroutine leaks after client disconnect.
- N+1 queries.
- JSON encoding huge unpaginated results.

## Production Examples

Use `pprof` in private admin/debug deployments, run `go test -bench=. -benchmem`, and compare CPU/memory profiles before and after changes.

## Go Code Samples

```go
ctx, cancel := context.WithTimeout(parent, 2*time.Second)
defer cancel()

if err := repo.List(ctx, filter); err != nil {
	return fmt.Errorf("list items: %w", err)
}
```

## Performance Considerations

Use profiles, benchmarks, query plans, and load tests. Avoid optimizing unmeasured code.

## Security Considerations

Do not expose pprof publicly. Avoid logging profiles with secrets or PII.

## Scalability Considerations

Bound concurrency, backpressure queues, paginate, cache hot reads, and monitor saturation signals.

