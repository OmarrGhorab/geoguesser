# Deployment

## Concepts

Deployment turns code into a reliable service. Include images, migrations, config, secrets, health checks, observability, rollback, and release strategy.

## Architecture Decisions

- Build immutable Docker images.
- Run migrations as controlled jobs.
- Use environment-specific config.
- Require health and readiness probes.
- Emit telemetry before taking traffic.
- Use semantic versions for releases.

## Trade-offs

Automated deployment increases speed but requires strong CI gates, rollback strategy, and environment protection.

## Anti-patterns

- Manual SSH deploys.
- Running migrations from every app instance.
- No rollback plan.
- Mutable production containers.
- Missing readiness checks.

## Common Mistakes

- App receives traffic before dependencies are ready.
- Migrations incompatible with old code.
- Secrets missing in production.
- No Sentry release tag.
- No dashboard for new endpoint.

## Production Examples

Use blue/green or rolling deploys with readiness probes. Apply expand-contract migrations for schema changes.

## Go Code Samples

```go
srv := &http.Server{Addr: cfg.HTTPAddr, Handler: router}
go func() {
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server failed", slog.Any("error", err))
	}
}()
```

## Performance Considerations

Warm caches carefully. Keep images small. Tune readiness so rollouts do not overload remaining pods.

## Security Considerations

Use least-privilege runtime identity, secret managers, image scanning, and signed artifacts where available.

## Scalability Considerations

Deployments should support multiple instances, stateless API processes, externalized sessions, and horizontal scaling.

