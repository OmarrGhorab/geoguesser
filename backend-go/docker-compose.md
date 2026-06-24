# Docker Compose

## Concepts

Use Docker Compose for local development with API, PostgreSQL, Redis, Prometheus, Grafana, and supporting services.

## Architecture Decisions

- Use Compose for local dependencies, not production orchestration.
- Define health checks for PostgreSQL and Redis.
- Mount source only for development with Air.
- Use named volumes for local persistence.

## Trade-offs

Compose improves onboarding but can differ from production. Keep production behavior in Dockerfiles and config, not Compose-only scripts.

## Anti-patterns

- Production secrets in compose files.
- No health checks.
- App starts before dependencies are ready.
- Using Compose as the only test environment.
- Exposing unnecessary ports.

## Common Mistakes

- Different env var names than production.
- Missing migration service.
- Volumes hiding fresh schema changes.
- No Redis persistence choice documented.
- No observability stack locally.

## Production Examples

```yaml
services:
  api:
    build: .
    env_file: .env
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
  postgres:
    image: postgres:17
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app"]
  redis:
    image: redis:7
```

## Go Code Samples

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
```

## Performance Considerations

Avoid bind mounts for database data. Use resource limits when local stacks become heavy.

## Security Considerations

Never commit real secrets. Bind local services to localhost where possible.

## Scalability Considerations

Compose files should mirror service dependencies enough for integration tests and local debugging.

