# Implementation Plan: Daily And Shared Challenges

**Branch**: `main` | **Date**: 2026-06-27 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/005-daily-shared-challenges/spec.md`

## Summary

Implement fixed-seed daily and shared challenges across the existing GeoGuess backend and Next.js frontend. The backend will own challenge identity, locked rule snapshots, deterministic location selection, attempts, leaderboards, streaks, missions, and spoiler-safe result visibility. The frontend will add localized daily/shared challenge pages, countdowns, leaderboard and comparison views, mission and streak surfaces, and reload-safe UI states that consume safe server DTOs.

## Technical Context

**Language/Version**: Go 1.25 backend module; Next.js 16.2.9 App Router, React 19.2.4, TypeScript frontend.

**Primary Dependencies**: Chi router, GORM, PostgreSQL, Redis, Goose, Prometheus, `log/slog`; Next App Router, native `fetch`, next-intl, Tailwind CSS v4, shadcn/ui/Radix patterns, Zustand only for local UI preferences.

**Storage**: PostgreSQL for durable challenges, challenge attempts, fixed selected locations, leaderboard entries, streaks, missions, and mission progress events. Redis only for short-lived rate-limit, idempotency, and optional countdown/read cache keys.

**Testing**: Backend `go test ./...`, targeted challenge/games tests, repository integration tests with `DATABASE_URL`, OpenAPI lint. Frontend `pnpm --dir client lint`, `pnpm --dir client typecheck`, `pnpm --dir client build`, component/flow tests or recorded browser manual validation for challenge UI.

**Target Platform**: Dockerized Go HTTP API plus localized Next.js App Router web client.

**Project Type**: Full-stack GeoGuess web application.

**Performance Goals**: Daily/shared challenge metadata loads in under 2 seconds for 95% of normal sessions; backend challenge metadata and attempt start paths return within 1 second p95; leaderboard pages use bounded pagination and avoid N+1 result reads; mission/streak updates are visible within 5 seconds after qualifying actions.

**Constraints**: Challenge rules and selected locations are immutable after creation; backend owns seed, scoring eligibility, leaderboard visibility, streak mutation, and mission progress; unfinished players must not receive answer spoilers; all visible UI copy must be localized for English and Arabic with RTL support; frontend route/data work must follow local Next docs under `client/node_modules/next/dist/docs/`.

**Scale/Scope**: Daily challenge plus shared challenge links for guests and accounts; 1 to 10 rounds per challenge; account-backed daily leaderboard and session-safe guest result handling; challenge-focused missions and daily streaks; no full economy, shop, or multiplayer real-time rooms in this phase.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Architecture boundaries**: PASS. Backend work stays in `backend/internal/challenges` with route wiring in `backend/internal/app`, reuse of `backend/internal/games` where appropriate, infrastructure adapters in `backend/internal/platform`, and database changes in Goose migrations. Frontend work stays in `client/app`, `client/features`, `client/lib`, `client/messages`, and existing component conventions. No GORM AutoMigrate.
- **Framework guidance**: PASS. Relevant installed Next.js docs were read for this plan: `01-app/01-getting-started/05-server-and-client-components.md`, `06-fetching-data.md`, `08-caching.md`, `15-route-handlers.md`, `02-guides/data-security.md`, and `02-guides/internationalization.md`. Plan favors Server Components and server-only data modules, with Client Components only for timers, interactive controls, and browser/session scoped state.
- **Testing gates**: PASS. Plan requires backend unit/service/repository/handler tests, OpenAPI contract updates, deterministic seed tests, date-bound streak tests, mission progress tests, frontend lint/typecheck/build, and browser/manual validation for daily/shared flows in English and Arabic.
- **UX consistency**: PASS. Plan covers loading, empty, unavailable, invalid-link, already-completed, not-completed, reset, disabled, success, leaderboard, mission, and streak states. Controls must use accessible names, keyboard focus, and existing shadcn/Radix/Tailwind patterns.
- **Localization and RTL**: PASS. All visible copy for challenges, missions, streaks, countdowns, leaderboard, result summaries, and error states must be added to `client/messages/en.json` and `client/messages/ar.json`; Arabic layouts use existing locale direction handling.
- **Performance budgets**: PASS. Budgets are stated for metadata load, attempt start, leaderboard pagination, and mission/streak update visibility. Measurement comes from backend tests/logs/metrics plus frontend build/browser validation.
- **Contracts and data**: PASS. Backend API changes update `backend/openapi/openapi.yaml` and this feature's contract artifact. Data changes are Goose migrations for challenge, attempt, leaderboard, streak, mission, and progress tables/indexes.
- **Operational readiness**: PASS. Plan covers `.env.example` updates if reset boundaries or feature toggles are introduced, structured logs without hidden coordinates, spoiler-safe DTOs, idempotent attempt creation, rate limits, metrics, and readiness impact.

## Project Structure

### Documentation (this feature)

```text
specs/005-daily-shared-challenges/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── challenges-openapi.md
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
│   │   ├── routes.go
│   │   └── server.go
│   ├── challenges/
│   │   ├── dto.go
│   │   ├── errors.go
│   │   ├── handler.go
│   │   ├── model.go
│   │   ├── repository.go
│   │   ├── seed.go
│   │   ├── service.go
│   │   ├── streaks.go
│   │   └── missions.go
│   ├── games/
│   ├── maps/
│   ├── middleware/
│   └── platform/
├── migrations/
└── openapi/
    └── openapi.yaml

client/
├── app/
│   └── [locale]/
│       └── challenges/
│           ├── daily/
│           └── [challengeId]/
├── features/
│   └── challenges/
├── lib/
│   ├── api/
│   └── i18n/
├── messages/
│   ├── en.json
│   └── ar.json
└── stores/
```

**Structure Decision**: Use a new backend `internal/challenges` package because daily/shared challenge identity, deterministic seed selection, leaderboards, streaks, and missions are distinct from the existing solo game loop. The challenge service may reuse games persistence or orchestration where it preserves locked rules, but challenge-specific invariants stay in the challenge boundary. Frontend routes live under localized `client/app/[locale]/challenges` with server-only API helpers and small Client Components for countdowns and interactive presentation.

## Complexity Tracking

No constitution violations.

## Phase 0 Research

See [research.md](./research.md).

## Phase 1 Design

See [data-model.md](./data-model.md), [contracts/challenges-openapi.md](./contracts/challenges-openapi.md), and [quickstart.md](./quickstart.md).

## Post-Design Constitution Check

- **Architecture boundaries**: PASS. Design artifacts keep durable data in PostgreSQL migrations, backend behavior in `internal/challenges`, frontend surfaces in localized app routes/features, and contracts in OpenAPI.
- **Framework guidance**: PASS. Design references the installed Next App Router Server/Client Component, data fetching, caching, route handler, data security, and internationalization guidance.
- **Testing gates**: PASS. Design maps deterministic seed, immutable rules, spoiler protection, leaderboard, mission, streak, and UI state behavior to automated or manual validation.
- **UX consistency**: PASS. UI contracts identify state coverage and localization/RTL requirements.
- **Localization and RTL**: PASS. Data and contract design keeps user-facing text frontend-owned through message catalogs.
- **Performance budgets**: PASS. Data model includes indexes and bounded reads for challenge lookup, attempts, leaderboards, streaks, and mission progress.
- **Contracts and data**: PASS. Contract artifact lists new endpoints and required OpenAPI updates; data model identifies Goose migration scope.
- **Operational readiness**: PASS. Research and quickstart include reset boundary, logging/redaction, metrics, idempotency, and readiness notes.
