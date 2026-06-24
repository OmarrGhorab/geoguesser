# Health Checks

## Concepts

Health checks include `/health`, `/ready`, and `/live`. Liveness means the process should stay running. Readiness means the instance can receive traffic. Health may summarize service status.

## Architecture Decisions

- `/live` should be cheap and local.
- `/ready` should check critical dependencies.
- `/health` may include version and dependency summary for internal use.
- Do not require auth for orchestrator probes unless platform supports it cleanly.

## Trade-offs

Deep dependency checks catch failures but can overload dependencies if called too often. Keep readiness checks bounded and cached briefly if needed.

## Anti-patterns

- Liveness depends on PostgreSQL.
- Readiness always returns 200.
- Health exposes secrets.
- Slow health checks.
- No shutdown readiness drain.

## Common Mistakes

- Same endpoint for live and ready.
- No timeout on dependency checks.
- Returning verbose internal errors publicly.
- No version/build info for operations.
- Probes hitting expensive queries.

## Production Examples

```text
GET /live   -> 200 if process event loop is alive
GET /ready  -> 200 if PostgreSQL and Redis are reachable
GET /health -> internal summary with build/version
```

## Go Code Samples

```go
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
	defer cancel()
	if err := h.db.PingContext(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
```

## Performance Considerations

Keep checks fast, bounded, and low allocation. Avoid expensive SQL.

## Security Considerations

Do not expose dependency URLs, credentials, or stack traces.

## Scalability Considerations

Readiness should turn false during shutdown so load balancers drain traffic safely.

