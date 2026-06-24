# Checklists

## Universal Gate

- Architecture is modular monolith.
- Handlers are thin.
- Services own business rules.
- Repositories own database access.
- No globals or hidden dependencies.
- Context propagates.
- Errors are handled and wrapped.
- Tests cover business logic.
- Observability exists.
- Security controls exist.

## Operations Gate

- Prometheus metrics.
- Grafana dashboard.
- OpenTelemetry traces.
- Sentry initialized and redacted.
- slog JSON logs.
- Request IDs and correlation IDs.
- `/health`, `/ready`, `/live`.
- Alerts for critical paths.

## Delivery Gate

- Docker multi-stage non-root build.
- `.dockerignore`.
- Image scanning.
- GitHub Actions lint/test/build.
- CodeQL, Trivy, Dependabot.
- Semantic release workflow.

