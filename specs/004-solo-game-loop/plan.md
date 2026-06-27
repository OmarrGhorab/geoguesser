# Implementation Plan: Solo Game Loop

**Branch**: `main` | **Date**: 2026-06-27 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/004-solo-game-loop/spec.md`

## Summary

Implement the backend-authoritative solo game loop in a new `backend/internal/games` feature package. The service will create solo games for guest or registered players, select non-repeating map locations, own round start/end timestamps, accept one idempotent guess per round, compute haversine distance and versioned score server-side, reveal coordinates only after round completion, and return durable final results from PostgreSQL.

## Technical Context

**Language/Version**: Go 1.24+ backend module.

**Primary Dependencies**: Chi router, GORM, PostgreSQL, Redis, Goose, Testify, Testcontainers, `log/slog`, existing `internal/platform/clock`, existing auth/session middleware.

**Storage**: PostgreSQL for durable `games`, `rounds`, `game_players`, `guesses`, `maps`, `locations`, and `map_locations`; Redis for short-lived idempotency, current-game/current-round cache, and rate-limit keys only.

**Testing**: `go test ./...`; targeted tests for `backend/internal/games`; OpenAPI lint through the existing pnpm contract gate.

**Target Platform**: Go HTTP API running in Docker Compose and CI.

**Project Type**: Modular monolith web API feature.

**Performance Goals**: Current round reads, accepted guess submissions, and final results return within 1 second for normal solo play; location selection avoids full-table random scans; result reads avoid N+1 queries.

**Constraints**: Backend owns timing and scoring; exact coordinates and provider metadata that reveal location remain hidden before reveal; one guess per player per round; no repeated locations within a game; guest and registered owners are both supported; frontend implementation is out of scope for this backend phase.

**Scale/Scope**: MVP solo games with 1 to 10 rounds, one active player per solo game, default 5 rounds, optional 10 to 600 second timers, score range 0 to 5000 per round.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Architecture boundaries**: PASS. Work stays in `backend/internal/games`, route wiring in `backend/internal/app`, reusable infrastructure in `backend/internal/platform`, contracts in `backend/openapi/openapi.yaml`, and planning in `specs/004-solo-game-loop`. Existing durable tables are already in Goose migration `backend/migrations/00001_initial_schema.sql`; any schema adjustment must be a new Goose migration.
- **Framework guidance**: PASS. No frontend code is planned for this backend phase. If later UI work is added, the relevant Next.js 16 docs under `client/node_modules/next/dist/docs/` must be read before changes.
- **Testing gates**: PASS. Required gates are `go test ./...`, targeted `go test ./internal/games/...` once the package exists, route/handler tests, service unit tests for state and scoring, repository tests with PostgreSQL/Testcontainers, and OpenAPI lint. Frontend `pnpm lint`, `pnpm typecheck`, and `pnpm build` remain release gates but should have no feature delta unless frontend work is added.
- **UX consistency**: PASS. Backend will return stable states and error codes for client loading, disabled, error, success, reveal, and final-result states. No visual components are planned.
- **Localization and RTL**: PASS. Backend responses use stable codes and safe messages; user-visible copy belongs in `en` and `ar` message catalogs when frontend consumes these states.
- **Performance budgets**: PASS. Budget is less than 1 second for current round, accepted guess, and final results during normal solo play. Data access must use indexed round/player/game queries and batched result reads. Measurement comes from automated integration tests plus request timing logs/metrics after implementation.
- **Contracts and data**: PASS. Existing OpenAPI game paths and schemas must be reviewed and tightened alongside implementation. Data changes use Goose migrations only; no GORM AutoMigrate.
- **Operational readiness**: PASS. No new required secrets are expected. Plan covers structured logs without hidden coordinates, rate limits, idempotency, readiness impact, and security checks for owner-only access.

## Project Structure

### Documentation (this feature)

```text
specs/004-solo-game-loop/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── games-openapi.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── cmd/api/
│   └── main.go
├── internal/
│   ├── app/
│   │   └── routes.go
│   ├── games/
│   │   ├── dto.go
│   │   ├── errors.go
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   ├── model.go
│   │   ├── repository.go
│   │   ├── repository_test.go
│   │   ├── scoring.go
│   │   ├── scoring_test.go
│   │   ├── service.go
│   │   ├── service_test.go
│   │   └── state.go
│   ├── maps/
│   │   ├── repository.go
│   │   └── service.go
│   ├── locations/
│   │   ├── media.go
│   │   └── model.go
│   ├── middleware/
│   │   └── rate_limit.go
│   ├── platform/
│   │   ├── clock/
│   │   ├── postgres/
│   │   └── redis/
│   └── session/
├── migrations/
└── openapi/
    └── openapi.yaml
```

**Structure Decision**: Use the established backend modular-monolith layout. `internal/games` owns handlers, service rules, repositories, DTOs, state transitions, and scoring. It consumes `maps.SelectLocations` through a narrow interface and uses platform clock/transaction helpers by constructor injection.

## Complexity Tracking

No constitution violations.

## Phase 0 Research

See [research.md](./research.md).

## Phase 1 Design

See [data-model.md](./data-model.md), [contracts/games-openapi.md](./contracts/games-openapi.md), and [quickstart.md](./quickstart.md).

## Post-Design Constitution Check

- **Architecture boundaries**: PASS. The design uses one new feature package and does not introduce layer-oriented top-level folders or service locators.
- **Framework guidance**: PASS. No frontend changes are part of this plan.
- **Testing gates**: PASS. The design maps user stories to unit, handler, repository, integration, OpenAPI, and release-gate checks.
- **UX consistency**: PASS. Contracts expose enough state for disabled submit, reveal, completed game, not-started game, and error views.
- **Localization and RTL**: PASS. Client copy remains catalog-owned; backend returns stable machine-readable errors.
- **Performance budgets**: PASS. Selection, current-round, guess, and results access patterns use indexed durable tables and Redis only for ephemeral acceleration.
- **Contracts and data**: PASS. Existing game contracts are documented for implementation review; schema gaps are identified for migration/task decisions.
- **Operational readiness**: PASS. Logging, rate limiting, idempotency, hidden-coordinate redaction, metrics, and readiness impact are accounted for.
