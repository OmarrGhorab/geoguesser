# Background Jobs Review

## Verify

- Retries.
- Exponential backoff.
- Graceful shutdown.
- Context cancellation.
- Worker leaks.
- Idempotent job handlers.
- Dead-letter or terminal failure behavior.
- Queue depth metrics.

## Reject

- Fire-and-forget goroutines from handlers.
- Unbounded workers.
- No retry limit.
- Ignoring context cancellation.
- Non-idempotent retries.

## Common Findings

High: handler starts goroutine to send email after response. Impact: work is lost on shutdown and errors are invisible. Recommendation: enqueue a background job with retry, idempotency, metrics, and graceful shutdown.

