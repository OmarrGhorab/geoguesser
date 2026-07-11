# Implementation Plan: Matchmaking And Ranked Foundations

**Branch**: `main` | **Date**: 2026-07-11 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/008-matchmaking-ranked/spec.md`

## Summary

Implement registered-only ranked matchmaking with duplicate-safe join, leave, and status flows; atomic two-player queue claims; a durable ranked match and immediately recoverable game destination; and request-driven stale-state recovery. Backend work adds `internal/matchmaking`, a Redis lease/claim coordinator, durable `matches` and `match_players` migrations, transactional ranked-game formation, targeted multiplayer generalization in `internal/games`, per-user rate limits, metrics, configuration, and corrected OpenAPI contracts. Frontend work adds a localized matchmaking page, server-only API helpers, Server Actions for queue commands, a same-origin status proxy, and a narrow Client Component that polls only while searching and announces state changes accessibly.

## Technical Context

**Language/Version**: Go 1.25 backend module; Next.js 16.2.9 App Router, React 19.2.4, TypeScript frontend.

**Primary Dependencies**: Chi Router, GORM, PostgreSQL, Goose, `go-redis/v9`, Prometheus, `log/slog`, existing session/CSRF/rate-limit middleware, game/map services; Next App Router, native `fetch`, Server Actions, next-intl, Tailwind CSS v4, existing shadcn/Radix patterns, Vitest and Testing Library. No new runtime dependency is required.

**Storage**: Redis for ephemeral queue entries, player leases, atomic pair claims, and bounded recovery metadata. PostgreSQL for ranked games, game participants, matches, match participants, rounds, guesses, and terminal outcomes. PostgreSQL is authoritative after match formation commits.

**Testing**: Backend `go test ./...`, `go test -race ./...`, targeted matchmaking/game/handler/repository tests, Redis concurrency integration tests, PostgreSQL transaction/integrity tests, route security tests, migration up/down validation, and OpenAPI validation. Frontend `npx pnpm@10.24.0 --dir client test`, `lint`, `typecheck`, and `build`, plus two-session browser validation in English and Arabic.

**Target Platform**: Dockerized Go HTTP API with PostgreSQL and Redis plus a localized Next.js App Router web client behind a same-origin production proxy; local development uses existing backend and frontend environment configuration.

**Project Type**: Full-stack GeoGuess web application.

**Performance Goals**: Join, leave, and status complete within 1 second p95 under normal load; two available compatible players receive the same assignment within 3 seconds p95; Redis operations are atomic and bounded to at most 20 candidate inspections per formation attempt; authoritative status recovery uses one bounded active-match query; the matchmaking page renders meaningful initial state within 2 seconds p95; polling occurs no faster than once every 2 seconds, never overlaps, and stops outside `searching`; added initial client JavaScript stays below 25 KB gzip.

**Constraints**: Registered active accounts only; one active queue entry and one active ranked assignment per player; initial mode is two-player `ranked_standard`; no rating calculation, seasons, divisions, rewards, parties, tournaments, or duel combat; Redis failure must not erase a durable assignment; match formation must tolerate retries and crashes; hidden coordinates, guesses, tokens, raw Redis keys, and opponent-private data must not enter queue/status payloads or telemetry; unsafe cookie-authenticated commands require CSRF; visible copy supports `en` and `ar` including RTL.

**Scale/Scope**: Initial design targets up to 10,000 simultaneously searching players across bounded supported modes, two players per match, five standard rounds, and request-driven formation/reconciliation. A background matcher/sweeper can replace the request-driven trigger in Phase 12 without changing public contracts or durable entities.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Architecture boundaries**: PASS. Backend ownership stays in `backend/internal/matchmaking`, targeted multiplayer reuse stays in `backend/internal/games`, Redis implementation stays behind a matchmaking coordinator abstraction, route/config wiring stays in existing app packages, and schema changes use Goose. Frontend work stays under localized app routes, `features/matchmaking`, server-only API helpers, and message catalogs. No AutoMigrate.
- **Framework guidance**: PASS. Relevant installed Next.js 16 guidance was read: Server/Client Components, data fetching, data security, forms/Server Actions, and internationalization. Initial authenticated data remains server-side; only polling, pending controls, focus, and live announcements require a narrow Client Component. A same-origin Route Handler avoids exposing backend secrets or introducing a browser-only API base.
- **Testing gates**: PASS. Plan requires pure state/compatibility tests, handler and route tests, real Redis atomicity/concurrency tests, PostgreSQL formation/lifecycle tests, OpenAPI validation, frontend polling/action/accessibility tests, full backend race tests, frontend test/lint/typecheck/build, and two-session browser evidence.
- **UX consistency**: PASS. Design covers loading, not queued, joining, searching, delayed, leaving, matched, unavailable, ineligible, unauthorized, rate-limited, retry, and success states. Controls use accessible names, disabled/pending states, visible focus, and non-color-only feedback; state changes use polite live regions and errors use alerts.
- **Localization and RTL**: PASS. All matchmaking and ranked-foundation copy is planned in `client/messages/en.json` and `client/messages/ar.json`; the localized route inherits the existing direction behavior and must be manually verified in Arabic.
- **Performance budgets**: PASS. API, formation, Redis candidate bound, database query bound, polling cadence, initial render, and bundle budgets are explicit. Metrics, integration timing assertions, frontend build output, and browser notes provide measurement evidence.
- **Contracts and data**: PASS. The stale guest/quick-play matchmaking placeholders in `backend/openapi/openapi.yaml` will be replaced by registered-only ranked contracts. A new Goose migration adds durable match entities and integrity indexes.
- **Operational readiness**: PASS. Plan adds validated queue lease, claim TTL, map, round, timer, and start-delay configuration to `.env.example`; PostgreSQL and Redis are already readiness dependencies. Logging and metrics use bounded labels and redacted identifiers. Request-driven recovery handles crash windows until Phase 12 workers add proactive sweeping.

## Project Structure

### Documentation (this feature)

```text
specs/008-matchmaking-ranked/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── matchmaking-openapi.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── cmd/api/
├── internal/
│   ├── app/
│   ├── config/
│   ├── games/
│   ├── matchmaking/
│   ├── middleware/
│   └── platform/
│       └── redis/
├── migrations/
└── openapi/

client/
├── app/
│   ├── api/
│   │   └── matchmaking/status/
│   └── [locale]/
│       ├── games/[gameId]/
│       └── matchmaking/
├── features/
│   ├── game/
│   └── matchmaking/
├── lib/
│   └── api/
├── messages/
└── stores/

specs/008-matchmaking-ranked/
└── [planning and validation artifacts]
```

**Structure Decision**: Add a dedicated `matchmaking` backend feature because queue state, pair claims, durable match formation, and recovery form one business boundary. Keep Redis details behind a coordinator interface and reuse `games` through narrow ranked multiplayer formation/lifecycle seams so `games` never imports `matchmaking`. Use direct ranked game destinations rather than fabricating private rooms, codes, or hosts. Add a localized ranked-game destination that reuses the existing game feature and API contract rather than embedding gameplay in the queue screen. Frontend canonical state remains backend-owned; no Zustand server-state store is introduced.

## Complexity Tracking

No constitution violations.

## Residual Risk (post-implementation 2026-07-11)

- **Interactive browser dual-session QA** (quickstart happy path + AR/RTL keyboard pass) remains an operator residual; automated unit/integration coverage stands in for CI.
- **Match lifecycle is transaction-scoped** for formation (active match + active game), completion, and cancel/abandon via `MultiplayerTxHooks`. A bounded internal failure code records launch/setup failures as `failed_to_start`, including failures discovered after assignment activation but before meaningful gameplay. HTTP surfaces for abandon/cancel remain internal service methods until a product API is specified.
- **`go test -race` on Windows** requires CGO; CI Linux job is the authoritative race gate.
- Ranked game UI uses lat/lng inputs rather than full Street View/map chrome; media URL is shown when present.
- **CI integration isolation** is handled by PostgreSQL/Redis service containers and sequential matchmaking/game/app integration tests; broader database-mutating packages remain unit-only in the default parallel suite.

## Phase 0 Research

See [research.md](./research.md).

## Phase 1 Design

See [data-model.md](./data-model.md), [contracts/matchmaking-openapi.md](./contracts/matchmaking-openapi.md), and [quickstart.md](./quickstart.md).

## Post-Design Constitution Check

- **Architecture boundaries**: PASS. Design artifacts isolate queue coordination in `matchmaking`, keep durable operations transactional, limit `games` changes to reusable multiplayer/lifecycle seams, and place frontend work in localized matchmaking feature paths.
- **Framework guidance**: PASS. The design uses a Server Component for initial status, server-only native fetch helpers, Server Actions for mutations, and one narrow Client Component plus same-origin status Route Handler for active polling.
- **Testing gates**: PASS. Contracts and quickstart map join idempotency, cross-mode uniqueness, leave/claim races, crash recovery, disabled-user revalidation, exact two-player formation, hidden-data safety, polling behavior, accessibility, localization, and all mandatory commands to verification.
- **UX consistency**: PASS. The contract defines every user-facing state and recoverable action; polling stops or backs off appropriately and focus/live-region behavior is explicit.
- **Localization and RTL**: PASS. Both supported catalogs and Arabic RTL browser validation are required.
- **Performance budgets**: PASS. Queue scripts, candidate scans, durable lookups, polling, and frontend bundle work are bounded and measurable.
- **Contracts and data**: PASS. The contract replaces the stale OpenAPI placeholder and the data model defines migration constraints, lifecycle transitions, Redis keys, and recovery precedence.
- **Operational readiness**: PASS. Configuration, readiness dependencies, fail-closed queue behavior, durable status fallback, rate limits, metrics, logs, redaction, and deferred Phase 12 sweeping are documented.
