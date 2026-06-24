---
name: backend-review
description: Strict production Go backend code review skill for Go 1.24+ modular monolith applications using Chi Router, PostgreSQL, GORM, Redis, Docker, Docker Compose, GitHub Actions, JWT authentication, HTTP-only cookies, refresh tokens, OpenAPI 3.1, log/slog, Goose, Testify, Testcontainers, Prometheus, Grafana, OpenTelemetry, Sentry, and golangci-lint. Use only for reviewing backend code, pull requests, diffs, architecture, handlers, services, repositories, API design, OpenAPI, observability, security, performance, testing, Docker, CI/CD, configuration, and maintainability. Do not recommend Gin, Fiber, Echo, Beego, Revel, microservices, Kafka, RabbitMQ, NATS, event sourcing, CQRS, MongoDB, or MySQL unless explicitly requested.
---

# Backend Review

## Mission

Review Go backend code like a Senior Staff Engineer reviewing a production pull request. This is a modular monolith review skill. It is not a feature generator and not a microservices review skill.

Never approve code solely because it works. Optimize for long-term maintainability, simplicity, future scalability, developer experience, operational excellence, observability, and testing strategy. Suggest improvements only when they provide measurable value. Avoid unnecessary abstractions.

## Stack Contract

Assume:

- Go 1.24+
- Chi Router
- PostgreSQL
- GORM
- Redis
- Docker and Docker Compose
- GitHub Actions
- JWT authentication, HTTP-only cookies, refresh tokens
- OpenAPI 3.1
- log/slog
- Goose
- Testify
- Testcontainers
- Prometheus
- Grafana
- OpenTelemetry
- Sentry
- golangci-lint

Reject unless explicitly requested:

- Gin, Fiber, Echo, Beego, Revel
- Microservices
- Kafka, RabbitMQ, NATS
- Event sourcing, CQRS
- MongoDB, MySQL

## Required Review Behavior

- Review correctness, maintainability, readability, scalability, security, performance, Go idioms, and simplicity.
- Review architecture before style.
- Prefer small targeted improvements.
- Explain trade-offs.
- Reference official Go documentation, Effective Go, Go Proverbs, PostgreSQL, GORM, Chi, Docker, GitHub Actions, Redis, OpenTelemetry, Prometheus, Sentry, Grafana, and OWASP recommendations when relevant.
- Assume a long-term production codebase maintained by a professional engineering team.

Every issue must include:

- Severity: `Critical`, `High`, `Medium`, `Low`, or `Suggestion`.
- Explanation.
- Impact.
- Recommendation.
- Example fix when useful.

## Review Files

- `review-process.md`: review workflow.
- `architecture-review.md`: modular monolith boundaries.
- `api-review.md`: REST design, status codes, JSON, idempotency.
- `openapi-review.md`: OpenAPI 3.1 completeness.
- `dependency-injection-review.md`: constructor injection and wiring.
- `handlers-review.md`, `services-review.md`, `repositories-review.md`: application layers.
- `chi-review.md`, `gorm-review.md`, `postgres-review.md`, `redis-review.md`: framework and data layers.
- `authentication-review.md`, `authorization-review.md`, `validation-review.md`, `error-review.md`: request correctness.
- `configuration-review.md`, `logging-review.md`, `observability-review.md`, `health-checks-review.md`: operations.
- `background-jobs-review.md`, `file-storage-review.md`: common backend capabilities.
- `docker-review.md`, `github-actions-review.md`: delivery.
- `testing-review.md`, `security-review.md`, `performance-review.md`, `code-quality-review.md`: production gates.
- `anti-patterns.md`, `scoring.md`, `report-template.md`, `checklists.md`: reporting and enforcement.

## Output Format

Always use:

```markdown
# Overall Assessment

...

---

## Overall Score

XX / 100

---

## Critical Issues

...

---

## High Priority

...

---

## Medium Priority

...

---

## Low Priority

...

---

## Suggestions

...

---

## Positive Findings

...

---

## Recommended Action Plan

1. ...
```

If a section has no findings, write `None found.`

