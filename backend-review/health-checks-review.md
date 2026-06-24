# Health Checks Review

## Verify

- `/health`.
- `/ready`.
- `/live`.
- Dependency checks.
- Probe timeouts.
- Shutdown readiness drain.
- No sensitive details in public response.

## Reject

- No health endpoints.
- Liveness checks depending on PostgreSQL.
- Readiness endpoint always returning 200.
- Slow dependency checks.

## Common Findings

High: readiness does not check PostgreSQL or Redis. Impact: instance can receive traffic while dependencies are unavailable. Recommendation: add bounded dependency checks with timeout and return 503 on failure.

