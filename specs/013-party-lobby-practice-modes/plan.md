# Implementation Plan: Party Lobby And Practice Modes

**Branch**: `codex/party-lobby-practice-modes` | **Date**: 2026-07-18 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/013-party-lobby-practice-modes/spec.md`

## Summary

Deliver two backend-only modes by extending established game boundaries. `party_lobby` becomes the canonical name for new hosted free-for-all rooms while legacy `private_room` records remain readable. Existing room codes, host controls, Redis presence/realtime state, shared timed rounds, and capacity enforcement are reused; defaults become five rounds and 180 seconds, and terminal results add deterministic individual placements. `practice` extends the solo game service with an owner-only open-ended lifecycle: create one untimed round, score and reveal immediately, append one next round transactionally after completion, page history, and explicitly end the session. Both modes remain progression-neutral.

## Technical Context

**Language/Version**: Go 1.24+ backend module.

**Primary Dependencies**: Chi Router, GORM, PostgreSQL, Goose, Redis for Party Lobby room/realtime state only, `log/slog`, Prometheus, existing sessions/CSRF/rate-limit middleware, OpenAPI 3.1.

**Storage**: PostgreSQL is authoritative for games, rounds, guesses, players, rooms, memberships, totals, and results. Redis remains ephemeral for Party Lobby readiness, presence, snapshots, idempotent room commands, and realtime fanout. Practice has no Redis dependency.

**Testing**: Pure mode/default/ranking tests; service tests for authorization and transitions; PostgreSQL tests for locking, unique next-round creation, pagination, migration constraints, 50-player standings, and concurrent requests; handler/route/OpenAPI tests; targeted security/redaction tests; full `go test ./...`, `go vet ./...`, and `golangci-lint run ./...`.

**Target Platform**: Dockerized Go HTTP API on Linux with PostgreSQL and Redis; local development on Windows remains supported.

**Project Type**: Backend feature slice of the GeoGuess web application.

**Performance Goals**: Party Lobby room snapshots and standings p95 below 300 ms for 50 players; Practice create/read/guess/advance/end p95 below 300 ms on local fixtures excluding media-provider latency; paged Practice history uses one bounded page query plus bounded answer/guess joins with default 20 and maximum 100 rows; no unbounded round loading.

**Constraints**: Backend only; Party Lobby supports 2–50 players and 1–10 rounds; default timer 180 seconds with existing 10–600 range; Practice has no fixed round endpoint, timer, matchmaking, realtime, progression, or background deadline work; legacy `private_room` data and contracts remain compatible during rollout.

**Scale/Scope**: One room contains at most 50 participants and 10 rounds. One Practice session may exceed 1,000 rounds, so all history reads and queries must be paginated and indexed by game/round number.

## Constitution Check

*GATE: Passed before research and re-checked after design.*

- **Architecture boundaries**: PASS. Room ownership remains in `internal/rooms`, game/round behavior in `internal/games`, wiring in `cmd/api`, infrastructure in `internal/platform`, and schema changes in one additive Goose migration. No AutoMigrate.
- **Framework guidance**: N/A. No frontend files are in scope.
- **Testing gates**: PASS. Unit, handler, PostgreSQL, concurrency, migration, contract, security, metrics, and full backend gates are planned. Frontend gates are N/A because no frontend files change.
- **UX consistency**: N/A for implementation. Contracts explicitly distinguish modes and define loading/error/recovery facts for later UI work.
- **Localization and RTL**: N/A for backend implementation. Stable codes and fields are provided for future `en`/`ar` clients without server-localized visible copy.
- **Performance budgets**: PASS. Bounded Party Lobby size, paginated Practice history, indexed sequential reads, explicit p95 budgets, and benchmark/integration measurements are required.
- **Contracts and data**: PASS. OpenAPI, saved examples, migration `00020`, and compatibility behavior are mandatory.
- **Operational readiness**: PASS. Metrics use bounded mode/outcome labels; logs redact codes, identities, guesses, and hidden coordinates; Practice readiness depends only on existing core persistence/media dependencies; Party Lobby retains existing realtime degradation behavior.

## Project Structure

### Documentation (this feature)

```text
specs/013-party-lobby-practice-modes/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── party-lobby-practice-openapi.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code

```text
backend/
├── cmd/api/main.go
├── internal/
│   ├── app/{routes.go,routes_test.go}
│   ├── games/{state.go,model.go,dto.go,errors.go,repository.go,service.go,handler.go,metrics.go,*_test.go}
│   └── rooms/{model.go,dto.go,repository.go,service.go,handler.go,metrics.go,*_test.go}
├── migrations/00020_party_lobby_practice_modes.sql
├── openapi/openapi.yaml
└── postman/
```

**Structure Decision**: Reuse `rooms` for hosted-lobby authority and `games` for gameplay. Do not introduce a second party/lobby package or a new Practice service because both would duplicate established authorization, scoring, round selection, persistence, and HTTP behavior.

## Complexity Tracking

No constitution violations. Legacy `private_room` remains as a read-compatible alias to avoid a destructive rename and allow old room records or clients to complete normally.
