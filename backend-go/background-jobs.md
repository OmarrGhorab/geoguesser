# Background Jobs

## Concepts

Background work covers worker pools, cron jobs, async jobs, graceful shutdown, retries, and exponential backoff. Use it for email, webhooks, cleanup, backfills, and slow side effects.

## Architecture Decisions

- Keep job handlers service-oriented.
- Use bounded worker pools.
- Propagate context and shutdown signals.
- Use retries with exponential backoff and max attempts.
- Make jobs idempotent.

## Trade-offs

Async work improves request latency but introduces eventual consistency and retry complexity.

## Anti-patterns

- Fire-and-forget goroutines from handlers.
- Unbounded queues.
- No retry limit.
- Non-idempotent job handlers.
- Ignoring shutdown.

## Common Mistakes

- Losing request/correlation IDs.
- Retrying permanent failures.
- No dead-letter behavior.
- No visibility into queue depth.
- No backpressure.

## Production Examples

Send welcome email by enqueueing a job after user creation, not by blocking the signup response.

## Go Code Samples

```go
func (w *Worker) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case job := <-w.jobs:
			if err := w.handle(ctx, job); err != nil {
				w.log.ErrorContext(ctx, "job failed", slog.Any("error", err))
			}
		}
	}
}
```

## Performance Considerations

Bound concurrency and queue sizes. Use backoff with jitter. Monitor job duration and backlog.

## Security Considerations

Authorize job creation. Avoid putting secrets in job payloads. Validate payloads before execution.

## Scalability Considerations

Design jobs to be idempotent and safe across multiple workers and process restarts.

