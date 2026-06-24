# Phase 5 Backend Architecture

## Purpose

This document defines the Go backend project structure before implementation. It maps the Phase 2 system design, Phase 3 database model, and Phase 4 OpenAPI contract into a maintainable Go modular monolith.

The backend should live under:

```text
backend/
```

## Architecture Style

Use a modular monolith with package-by-feature organization.

Why:

- The domain is still evolving.
- A single deployable backend is simpler to build, run, test, observe, and operate.
- Feature packages make ownership clear without introducing microservice complexity.
- Clean boundaries allow future extraction if a feature truly outgrows the monolith.

Do not create top-level `controllers/`, `models/`, `services/`, or `repositories/` folders. That package-by-layer style spreads one feature across the repo and creates cross-feature coupling. In Go, keep feature behavior together under `internal/<feature>`.

## Proposed Backend Tree

```text
backend/
  cmd/
    api/
      main.go
    worker/
      main.go

  internal/
    app/
      app.go
      server.go
      routes.go
      shutdown.go

    config/
      config.go
      env.go
      validation.go

    http/
      request.go
      response.go
      errors.go
      pagination.go
      validation.go

    middleware/
      auth.go
      csrf.go
      cors.go
      logging.go
      recover.go
      request_id.go
      rate_limit.go
      security_headers.go
      timeout.go

    auth/
      handler.go
      service.go
      repository.go
      model.go
      dto.go
      tokens.go
      passwords.go
      cookies.go
      errors.go

    users/
      handler.go
      service.go
      repository.go
      model.go
      dto.go
      errors.go

    profiles/
      handler.go
      service.go
      repository.go
      dto.go
      errors.go

    maps/
      handler.go
      service.go
      repository.go
      model.go
      dto.go
      selection.go
      errors.go

    locations/
      handler.go
      service.go
      repository.go
      model.go
      dto.go
      media.go
      errors.go

    games/
      handler.go
      service.go
      repository.go
      model.go
      dto.go
      scoring.go
      state.go
      errors.go

    rooms/
      handler.go
      service.go
      repository.go
      model.go
      dto.go
      codes.go
      presence.go
      errors.go

    matchmaking/
      handler.go
      service.go
      queue.go
      worker.go
      dto.go
      errors.go

    realtime/
      handler.go
      hub.go
      events.go
      publisher.go
      subscriber.go
      errors.go

    leaderboards/
      handler.go
      service.go
      repository.go
      model.go
      dto.go
      rankings.go
      errors.go

    friends/
      handler.go
      service.go
      repository.go
      model.go
      dto.go
      errors.go

    achievements/
      service.go
      repository.go
      model.go
      rules.go
      dto.go
      errors.go

    billing/
      handler.go
      service.go
      repository.go
      provider.go
      model.go
      dto.go
      entitlements.go
      errors.go

    ads/
      service.go
      dto.go
      placement.go
      errors.go

    health/
      handler.go
      service.go
      dto.go

    jobs/
      scheduler.go
      cleanup_rooms.go
      rebuild_leaderboards.go
      expire_sessions.go
      reconcile_payments.go

    platform/
      postgres/
        postgres.go
        transaction.go
      redis/
        redis.go
        cache.go
        rate_limit.go
        locks.go
      observability/
        logging.go
        metrics.go
        tracing.go
        sentry.go
      email/
        client.go
        templates.go
      storage/
        storage.go
        local.go
        s3.go
      payments/
        provider.go
        stripe.go
      clock/
        clock.go
      id/
        id.go

  migrations/
    .gitkeep

  openapi/
    openapi.yaml

  deployments/
    docker/
      Dockerfile
      docker-compose.yml
    nginx/
      nginx.conf

  scripts/
    dev.ps1
    lint.ps1
    test.ps1

  go.mod
  go.sum
  Makefile
  .air.toml
  .golangci.yml
  README.md
```

## Top-Level Directory Responsibilities

### `cmd/`

Executable entrypoints.

- `cmd/api/main.go`: starts the HTTP API.
- `cmd/worker/main.go`: starts background jobs later.

Keep these files thin. They should load config, create dependencies, wire the app, and run.

### `internal/`

All application code lives here. Go prevents external projects from importing `internal`, which protects the backend from accidental public API leakage.

### `internal/app/`

Owns runtime assembly:

- Chi router creation.
- Route registration.
- HTTP server configuration.
- Graceful shutdown.
- Dependency wiring helpers.

Do not put business logic here.

### `internal/config/`

Owns configuration loading and validation:

- Environment variables.
- Defaults.
- Secret references.
- Development and production validation.
- Configuration precedence.

Use Koanf or Viper when implementation starts. Keep config values typed and validated before the server boots.

### `internal/http/`

Shared HTTP helpers:

- JSON encode/decode.
- Error envelope writing.
- Request body size handling.
- Pagination parsing.
- Request validation helpers.

This package must not know domain rules.

### `internal/middleware/`

Cross-cutting HTTP middleware:

- Request ID and correlation ID.
- Structured request logging.
- Panic recovery.
- Auth session loading.
- CSRF checks.
- CORS.
- Rate limits.
- Security headers.
- Request timeouts.

Middleware may authenticate and attach safe request context values, but it must not own domain authorization decisions.

### Feature Packages

Feature packages own their domain:

- `auth`
- `users`
- `profiles`
- `maps`
- `locations`
- `games`
- `rooms`
- `matchmaking`
- `realtime`
- `leaderboards`
- `friends`
- `achievements`
- `billing`
- `ads`
- `health`

Each feature should contain its handler, service, repository, DTOs, models, errors, and focused helper files.

### `internal/platform/`

Infrastructure adapters:

- PostgreSQL/GORM.
- Redis.
- OpenTelemetry, Prometheus, Sentry, slog.
- Email.
- File/object storage.
- Payment provider clients.
- Clock and ID generation.

Platform packages should expose small clients and helpers. They must not contain gameplay business rules.

### `migrations/`

Goose SQL migrations.

Rules:

- No migrations until the ERD is stable.
- No GORM AutoMigrate in production.
- Every schema change needs up/down SQL or an explicit irreversible note.
- Large changes use expand-contract migrations.

### `openapi/`

OpenAPI 3.1 API contract.

The current contract is:

```text
backend/openapi/openapi.yaml
```

Keep it updated whenever endpoint requests or responses change.

### `deployments/`

Deployment-specific files:

- Dockerfile.
- Docker Compose.
- Nginx config.
- Future observability config if needed.

### `scripts/`

Local scripts only. Keep CI logic in GitHub Actions later, but scripts can make local developer commands easier.

## Feature Package Anatomy

Most feature packages should use this shape:

```text
internal/games/
  handler.go
  service.go
  repository.go
  model.go
  dto.go
  errors.go
  service_test.go
  repository_test.go
```

### `handler.go`

Responsibilities:

- Parse path params, query params, headers, cookies, and JSON bodies.
- Validate transport-level request shape.
- Call service methods.
- Map service errors to HTTP errors.
- Write response DTOs.

Handlers must not:

- Access GORM directly.
- Access Redis directly unless the feature is explicitly a Redis transport feature.
- Calculate scores.
- Authorize complex domain actions.
- Start database transactions.

### `service.go`

Responsibilities:

- Enforce business rules.
- Authorize use cases.
- Coordinate repositories.
- Own transaction boundaries.
- Calculate scores and state transitions.
- Call cache/realtime/payment abstractions when needed.

Services must not:

- Import `net/http`.
- Return raw GORM models as API DTOs.
- Hide dependencies through package globals.

### `repository.go`

Responsibilities:

- Own PostgreSQL/GORM queries.
- Accept `context.Context`.
- Return domain models or persistence models owned by the package.
- Keep query methods explicit.

Repositories must not:

- Return `*gorm.DB`.
- Perform business authorization.
- Know HTTP status codes.
- Use unbounded list queries.

### `model.go`

Persistence/domain structs for the package.

Rules:

- Keep GORM tags here when using GORM models.
- Do not expose models directly to handlers as responses.
- Use UUIDs and explicit relationships.

### `dto.go`

Request and response DTOs.

Rules:

- Match OpenAPI schema names where practical.
- Do not include hidden internal fields.
- Do not include exact location coordinates before reveal.

### `errors.go`

Domain error definitions.

Use stable errors that handlers can map:

```text
ErrGameNotFound
ErrRoundClosed
ErrAlreadyGuessed
ErrForbidden
ErrInvalidTransition
```

## Dependency Direction

Allowed dependency direction:

```text
cmd/api
  -> internal/app
  -> internal/<feature>
  -> internal/platform
```

Feature packages may depend on platform interfaces or clients passed through constructors. They should not import each other freely.

When one feature needs another feature's behavior, prefer one of:

- A small interface defined by the consuming package.
- A domain service composed in `internal/app`.
- A shared primitive moved to a narrow package only after duplication proves it is real.

Avoid cycles such as:

```text
games -> rooms -> games
```

Instead, `rooms.Service` can call a small `GameStarter` interface implemented by `games.Service`, wired in `internal/app`.

## Dependency Injection

Use constructor injection and manual wiring.

Example shape:

```go
type Service struct {
	repo   Repository
	cache  RoomCache
	log    *slog.Logger
	clock  clock.Clock
}

func NewService(repo Repository, cache RoomCache, log *slog.Logger, clock clock.Clock) *Service {
	return &Service{repo: repo, cache: cache, log: log, clock: clock}
}
```

Rules:

- Wire dependencies in `cmd/api/main.go` or `internal/app`.
- Avoid DI containers.
- Avoid service locators.
- Avoid package-level mutable globals.
- Keep interfaces close to the consumer.
- Prefer concrete types until tests or alternate implementations require an interface.

## API Routing Plan

`internal/app/routes.go` should mount routes by feature.

```text
/api/v1/auth/*
/api/v1/profile
/api/v1/users/{userId}/stats
/api/v1/maps/*
/api/v1/locations/{locationId}/media
/api/v1/games/*
/api/v1/rooms/*
/api/v1/matchmaking/*
/api/v1/leaderboards/*
/api/v1/billing/*
/api/v1/webhooks/payments
/api/v1/health
/api/v1/ready
/api/v1/metrics
/realtime/*
```

OpenAPI remains the source of truth for request/response contracts.

## Package Ownership By Domain

| Package | Owns | Depends On |
| --- | --- | --- |
| `auth` | Login, register, refresh, logout, cookies, passwords, tokens, sessions. | PostgreSQL, Redis rate limit, clock, ID, logger. |
| `users` | Registered account records and public stats lookup. | PostgreSQL, logger. |
| `profiles` | Profile reads/updates. | Users repository, PostgreSQL, logger. |
| `maps` | Public map listing, access tier decisions, map membership. | PostgreSQL, Redis cache, entitlements interface. |
| `locations` | Location metadata, media references, admin import later. | PostgreSQL, storage/media provider, logger. |
| `games` | Game creation, rounds, guesses, scoring, results, state transitions. | PostgreSQL, Redis cache, maps service interface, realtime publisher, clock. |
| `rooms` | Private/public rooms, room codes, host commands, membership. | PostgreSQL, Redis presence/cache, games starter interface, realtime publisher. |
| `matchmaking` | Queue entry/exit, match formation, public room creation. | Redis queue, rooms service interface, worker scheduler. |
| `realtime` | WebSocket/SSE connections and room events. | Redis pub/sub, auth session loader, logger. |
| `leaderboards` | Ranking reads, leaderboard materialization. | PostgreSQL, Redis sorted sets, games results. |
| `friends` | Friend requests and friend graph. | PostgreSQL, users repository. |
| `achievements` | Achievement rules and awards. | PostgreSQL, games completion events. |
| `billing` | Entitlements, checkout, portal, payment webhook handling. | PostgreSQL, payment provider, Redis idempotency, logger. |
| `ads` | Ad placement decisions. | Billing entitlement interface, config. |
| `health` | Health, readiness, metrics route adapters. | PostgreSQL ping, Redis ping, observability. |

## MVP Implementation Order

1. Backend foundation:
   - `cmd/api`
   - `internal/app`
   - `internal/config`
   - `internal/http`
   - `internal/middleware`
   - `internal/platform/postgres`
   - `internal/platform/redis`
   - `internal/platform/observability`
   - `health`

2. Auth and profile:
   - `auth`
   - `users`
   - `profiles`

3. Gameplay read foundations:
   - `maps`
   - `locations`

4. Solo game loop:
   - `games`
   - scoring
   - guess submission
   - result DTOs

5. Private rooms:
   - `rooms`
   - `realtime`
   - Redis presence

6. Expansion:
   - `matchmaking`
   - `leaderboards`
   - `friends`
   - `achievements`
   - `billing`
   - `ads`

## Transaction Boundaries

Services own transactions.

Use transactions for:

- Register user plus profile plus session.
- Create game plus rounds plus initial player.
- Submit guess plus update player total plus maybe complete round.
- Start room plus create game plus round records.
- Complete game plus leaderboard candidate writes.
- Process payment webhook plus subscription/entitlement updates.

Do not pass transactions through `context.Context`. Use an explicit transaction helper in `internal/platform/postgres`.

## Redis Boundaries

Redis usage should be behind feature-specific abstractions:

- `rooms.PresenceStore`
- `matchmaking.Queue`
- `leaderboards.Cache`
- `auth.RateLimiter`
- `billing.IdempotencyStore`

Do not scatter raw Redis keys across handlers and services. Keep key naming close to the owning feature.

## Background Jobs

Use `cmd/worker` when background work becomes real.

Initial jobs:

- Expire abandoned rooms.
- Remove stale matchmaking entries.
- Rebuild leaderboard snapshots.
- Expire old sessions.
- Reconcile payment events.

Jobs must:

- Accept `context.Context`.
- Respect graceful shutdown.
- Use structured logs.
- Use retries with exponential backoff for external dependencies.
- Avoid duplicate work through locks or idempotency.

## Observability Architecture

Every request path should include:

- Request ID.
- Correlation ID.
- JSON structured logs.
- OpenTelemetry spans.
- Prometheus metrics.
- Sentry capture for unexpected errors.

Platform ownership:

```text
internal/platform/observability/
  logging.go
  metrics.go
  tracing.go
  sentry.go
```

Feature packages should log domain-relevant events, but middleware should own request logs.

## Security Architecture

Security-sensitive code should be centralized:

- Password hashing in `auth/passwords.go`.
- JWT signing/verification in `auth/tokens.go`.
- Cookie writing in `auth/cookies.go`.
- CSRF middleware in `middleware/csrf.go`.
- Rate limits in `middleware/rate_limit.go` and feature-specific Redis stores.
- Security headers in `middleware/security_headers.go`.
- Payment signature verification in `billing/provider.go` or `internal/platform/payments`.

Rules:

- Never store auth tokens in localStorage.
- Never log tokens, passwords, CSRF values, or payment secrets.
- Never expose exact coordinates before guess lock or round reveal.
- Do not trust guest identities for privileged actions.
- Authorize inside services, not only middleware.

## Testing Architecture

Test close to the code.

Recommended tests:

- Unit tests for scoring, room code generation, auth token behavior, and domain transitions.
- Handler tests for request parsing, error mapping, and response shape.
- Repository tests using Testcontainers for PostgreSQL.
- Redis integration tests using Testcontainers for presence, queues, rate limits, and cache behavior.
- End-to-end API contract tests against OpenAPI after implementation.

Folder pattern:

```text
internal/games/service_test.go
internal/games/repository_test.go
internal/games/handler_test.go
```

## Naming Standards

- Package names are short and lowercase: `games`, `rooms`, `auth`.
- Avoid package names like `utils`, `common`, `helpers`, `managers`.
- File names describe purpose: `service.go`, `repository.go`, `scoring.go`.
- Interfaces are named by behavior when useful: `GameStarter`, `PresenceStore`, `TokenSigner`.
- Keep receiver names short and consistent: `s *Service`, `h *Handler`, `r *Repository`.

## Anti-Patterns To Reject

- Database calls in handlers.
- GORM models returned as API responses.
- Package-level global DB, Redis, logger, or config.
- A generic repository shared by all entities.
- A top-level `pkg/` folder for internal app code.
- A top-level `services/` folder that all features import.
- Hidden dependencies in context.
- Circular imports between feature packages.
- Panics in request paths.
- Ignored errors.
- Long functions that mix HTTP, business logic, persistence, and logging.

## Open Questions

- Should guests be allowed to create private rooms, or only join them?
- Should realtime start with WebSockets immediately, or should MVP use polling/SSE first?
- Which payment provider will determine `internal/platform/payments` shape?
- Which imagery provider will determine `locations.MediaProvider` shape?
- Should generated OpenAPI types be introduced immediately, or should hand-written DTOs be used first?

## Phase 5 Exit Criteria

Phase 5 is ready for scaffolding when:

- Backend folder tree is accepted.
- MVP implementation order is accepted.
- Feature package boundaries are accepted.
- Dependency direction rules are accepted.
- Transaction and Redis ownership rules are accepted.
- Open questions are either answered or explicitly deferred.
