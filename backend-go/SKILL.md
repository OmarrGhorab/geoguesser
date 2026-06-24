---
name: backend-go
description: Production Go backend engineering skill for Go 1.24+ services using Chi Router, PostgreSQL, GORM, Redis, Goose migrations, OpenAPI 3.1, JWT authentication, HTTP-only cookies, refresh tokens, log/slog, Docker, Docker Compose, GitHub Actions, Prometheus, Grafana, OpenTelemetry, Sentry, Testcontainers, Air, golangci-lint, and Makefile. Use for building, reviewing, or refactoring modern cloud-native Go APIs with Clean Architecture, DDD where useful, dependency injection, repository and service layers, observability, security, performance, deployment, and production coding standards. Do not use for Gin, Fiber, Echo, Beego, Revel, MySQL, or MongoDB unless explicitly requested.
---

# Backend Go

## Mission

Build production-grade Go 1.24+ backend services. This is not a generic Go skill. Apply Go idioms, Effective Go, Go Proverbs, Clean Architecture, pragmatic Domain-Driven Design, and cloud-native operations.

## Stack Contract

Assume every project uses:

- Go 1.24+
- Chi Router
- PostgreSQL
- GORM
- Redis
- Docker and Docker Compose
- GitHub Actions
- OpenAPI 3.1 and Swagger UI
- JWT access tokens, refresh tokens, HTTP-only secure cookies
- log/slog
- Goose migrations
- Prometheus, Grafana, OpenTelemetry, Sentry
- Testcontainers
- Air
- golangci-lint
- Makefile

Never recommend Gin, Fiber, Echo, Beego, Revel, MySQL, or MongoDB unless explicitly requested.

## Core Philosophy

- Keep handlers thin.
- Put business logic in services.
- Put database access in repositories.
- Never access the database directly from handlers.
- Use dependency injection everywhere.
- Propagate `context.Context` through every request path.
- Implement graceful shutdown.
- Use structured JSON logging with request IDs and correlation IDs.
- Prefer explicit error handling and wrapped errors.
- Use repository and service layers.
- Keep interfaces close to consumers.
- Prefer concrete types until an interface is needed.
- Avoid global state and service locators.
- Compose behavior; do not simulate inheritance.
- Apply SOLID where it improves clarity.
- Prefer simplicity over cleverness.
- Keep packages and functions small.
- Never panic in application code.
- Never ignore returned errors.

## How To Use This Skill

Read only the files relevant to the task:

- `architecture.md`, `clean-architecture.md`, `project-structure.md`: system shape.
- `api-design.md`, `openapi.md`, `pagination.md`: REST conventions, OpenAPI 3.1, versioning, docs, lists.
- `chi.md`, `handlers.md`, `middleware.md`: HTTP layer.
- `services.md`, `repositories.md`, `dependency-injection.md`: application layering and explicit wiring.
- `gorm.md`, `postgresql.md`, `migrations.md`: persistence.
- `redis.md`, `caching.md`: Redis and cache strategy.
- `authentication.md`, `authorization.md`, `security.md`: auth and security.
- `validation.md`, `error-handling.md`, `logging.md`, `configuration.md`: request and runtime foundations.
- `background-jobs.md`, `file-storage.md`, `email.md`: common backend capabilities.
- `docker.md`, `docker-compose.md`, `github-actions.md`, `deployment.md`: delivery.
- `testing.md`, `observability.md`, `health-checks.md`, `performance.md`: production readiness and operations.
- `coding-standards.md`, `checklists.md`: final gates.
- `examples/`: example areas for auth, users, products, uploads, Redis cache, pagination, transactions, middleware, JWT, testing, Docker, and GitHub Actions.

## Default Architecture

```text
cmd/api/main.go
internal/config
internal/http
internal/middleware
internal/auth
internal/users
  handler.go
  service.go
  repository.go
  model.go
  dto.go
  errors.go
internal/platform/postgres
internal/platform/redis
internal/platform/observability
migrations
openapi
```

Handlers decode and encode HTTP. Services enforce use cases and transactions. Repositories hide persistence. Platform packages own infrastructure clients. Dependencies are wired in `cmd/api/main.go`.

## Baseline Go Sample

```go
type UserService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}

type UserHandler struct {
	users UserService
	log   *slog.Logger
}

func NewUserHandler(users UserService, log *slog.Logger) *UserHandler {
	return &UserHandler{users: users, log: log}
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_user_id")
		return
	}

	user, err := h.users.GetByID(ctx, id)
	if err != nil {
		writeError(w, statusFromError(err), codeFromError(err))
		return
	}

	writeJSON(w, http.StatusOK, user)
}
```

## Review Gates

Before finishing backend work, verify:

- `gofmt` and `goimports` pass.
- `golangci-lint run` passes.
- `go test ./...` passes.
- Race detector is used for concurrency-sensitive changes.
- Integration tests use Testcontainers for PostgreSQL and Redis.
- Migrations are reversible or explicitly irreversible with rationale.
- OpenAPI is updated for request/response changes.
- Logs, metrics, traces, Sentry errors, health checks, and readiness behavior are production-aware.
- Docker images are multi-stage, non-root, scanned, and small.
