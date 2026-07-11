# Implementation Plan: Friends And Social Graph Backend

**Branch**: `009-friends-social-graph` | **Date**: 2026-07-11 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/009-friends-social-graph/spec.md`

## Summary

Implement a backend-only social graph in `backend/internal/friends` with sorted-pair `friendships` storage, friend request lifecycle, accepted-friend lists and removal, block/unblock privacy rules, and an authenticated friends leaderboard cohort in `backend/internal/leaderboards`. Delivery uses Goose migration `00015`, Chi routes with registered auth, CSRF, hashed per-user rate limits, OpenAPI contract updates, Prometheus metrics, and PostgreSQL-backed tests.

## Technical Context

**Language/Version**: Go 1.24+ backend module (`backend/`).

**Primary Dependencies**: Chi Router, GORM, PostgreSQL, Goose, Prometheus, `log/slog`, existing session/CSRF/rate-limit middleware, profiles public projection fields, leaderboard read models.

**Storage**: PostgreSQL is authoritative for all friendship edges and friends leaderboard cohort reads. No Redis friendship cache.

**Testing**: Unit tests for service rules and cursors; handler/route tests for auth, CSRF, status codes, and privacy; PostgreSQL integration tests for constraints, locking, concurrency, pagination, and leaderboard cohort filtering; OpenAPI validation; full package gates in polish phase.

**Target Platform**: Dockerized Go HTTP API with PostgreSQL and Redis (Redis used only for existing rate limiting/sessions, not friendship state).

**Project Type**: Backend feature slice of the full-stack GeoGuess application.

**Performance Goals**: Friend list and request list pages p95 under 200 ms on local fixtures for limit ≤ 100; friends leaderboard page p95 under 300 ms without Redis page cache; at most one primary list query plus bounded profile join per page; no N+1 profile loads.

**Constraints**: Registered active users only; sorted UUID pairs; declines delete rows; block ownership private; privacy-safe not-found for blocked/missing/inactive targets on discovery paths; friends rank relative to self + accepted friends; no profile hiding, room, or matchmaking coupling in this phase.

**Scale/Scope**: Designed for typical social fan-out (hundreds of friends per user) with cursor pagination (default 20, max 100) and indexed pair/status lookups.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Architecture boundaries**: PASS. Ownership stays in `internal/friends`; leaderboards consume friendship facts through a narrow repository contract without importing friends internals; Goose migration only; no AutoMigrate.
- **Testing gates**: PASS. Plan requires service, repository, handler, route, metrics, privacy, concurrency, and migration validation tests plus polish-phase full gates.
- **UX consistency**: N/A for backend-only delivery; API errors remain stable envelope codes for future localized clients.
- **Localization and RTL**: N/A backend-only; messages stay English server strings for frontend localization.
- **Performance budgets**: PASS. Explicit list/leaderboard budgets and index strategy recorded.
- **Contracts and data**: PASS. OpenAPI and migration tasks are mandatory.
- **Operational readiness**: PASS. Metrics, structured logs, rate limits, and readiness via existing PostgreSQL dependency.

## Project Structure

### Documentation (this feature)

```text
specs/009-friends-social-graph/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── friends-openapi.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code

```text
backend/
├── cmd/api/main.go
├── internal/
│   ├── app/{server.go,routes.go,routes_test.go}
│   ├── friends/{model,dto,errors,cursor,metrics,repository,service,handler,*_test}.go
│   └── leaderboards/{repository,service,handler,*_test}.go
├── migrations/00015_friends_social_graph.sql
└── openapi/openapi.yaml
```

**Structure Decision**: Dedicated `friends` package owns relationship lifecycle. Leaderboards extend with a friends cohort query and authenticated route without depending on friends package types beyond SQL-level friendship facts.

## Complexity Tracking

No constitution violations. Blocking is intentionally limited to friend APIs and friends leaderboards to avoid premature coupling to profiles, rooms, and matchmaking.
