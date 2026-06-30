# Implementation Plan: Room Realtime Reconnection

**Branch**: `codex/phase-06-room-realtime-reconnection` | **Date**: 2026-06-30 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/006-room-realtime-reconnection/spec.md`

## Summary

Implement private room multiplayer with authoritative HTTP room/game commands and a WebSocket room channel for versioned lobby, presence, round, guess-progress, result, and reconnect events. Backend work adds `internal/rooms` and `internal/realtime`, reuses existing durable `rooms`, `room_players`, `games`, `game_players`, `rounds`, and `guesses` tables, uses Redis for snapshots, presence, locks, reconnect windows, idempotency, and pub/sub fanout, and extends the game loop for multi-player round completion. Frontend work adds localized room create/join/lobby/game surfaces, server-only room API helpers, and narrow Client Components for realtime connection, countdown, presence, and recovery states.

## Technical Context

**Language/Version**: Go 1.25 backend module; Next.js 16.2.9 App Router, React 19.2.4, TypeScript frontend.

**Primary Dependencies**: Chi router, GORM, PostgreSQL, Redis, Goose, Prometheus, `log/slog`, `github.com/coder/websocket` for backend WebSocket transport; Next App Router, native `fetch`, next-intl, Tailwind CSS v4, shadcn/Radix patterns, Zustand only for local UI preferences.

**Storage**: PostgreSQL for durable room, membership, game, round, guess, and result facts. Redis for active room snapshots, presence heartbeats, reconnect windows, ready state, room locks, event versions, pub/sub fanout, rate limits, and short-lived idempotency claims.

**Testing**: Backend `go test ./...`, targeted `rooms`, `realtime`, `games`, repository, and handler tests, Redis integration tests, OpenAPI lint. Frontend `npx pnpm@10.24.0 --dir client lint`, `typecheck`, `build`, room component tests, and two-session browser validation or Playwright coverage for room create/join/start/reconnect.

**Target Platform**: Dockerized Go HTTP API plus localized Next.js App Router web client behind same-origin production proxy; local development uses direct API and realtime URLs.

**Project Type**: Full-stack GeoGuess web application.

**Performance Goals**: Room create/join/settings/start HTTP commands return within 1 second p95; room event delivery reaches connected clients within 2 seconds for 95% of normal sessions; initial room page state loads within 2 seconds p95; active rooms avoid per-second server polling; reconnect restoration completes within 2 seconds after transport reconnect for 95% of normal sessions.

**Constraints**: Backend remains authoritative for room state, game state, scoring, timers, and authorization; hidden coordinates and answer-revealing provider metadata never leave the backend before allowed reveal; room commands use idempotency/rate limits where retryable or abuse-sensitive; WebSocket events are versioned hints and clients must refetch on reconnect/version mismatch; frontend visible copy must be localized in `en` and `ar` with RTL support.

**Scale/Scope**: Private rooms only; guest and registered users can create/join; 2 to 50 players per room per existing schema, 1 to 10 rounds, timer 10 to 600 seconds or untimed; no public matchmaking, teams, spectator mode, voice/video, free-form chat, or ranked multiplayer in this phase.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Architecture boundaries**: PASS. Backend work stays in `backend/internal/rooms`, `backend/internal/realtime`, targeted extensions to `backend/internal/games`, route wiring in `backend/internal/app`, infrastructure helpers in `backend/internal/platform/redis`, and any schema changes in Goose migrations. Frontend work stays in localized `client/app/[locale]/rooms`, `client/features/rooms`, `client/features/game`, `client/lib/api`, and `client/messages`. No GORM AutoMigrate.
- **Framework guidance**: PASS. Relevant installed Next.js 16 docs were read: `01-app/01-getting-started/05-server-and-client-components.md`, `06-fetching-data.md`, `02-guides/data-security.md`, `02-guides/forms.md`, and `02-guides/internationalization.md`. Plan keeps pages/server data in Server Components/server-only modules and uses Client Components only for browser WebSocket, countdown, presence, forms with pending/error state, and gameplay interaction.
- **Testing gates**: PASS. Plan requires backend unit tests for code generation/state machines/authorization/event envelopes/deadlines, handler tests for every room command and realtime upgrade rejection path, repository tests for room/game transactions, Redis integration tests for presence/reconnect/pubsub/locks, OpenAPI lint, frontend lint/typecheck/build, room component tests, and two-session browser validation.
- **UX consistency**: PASS. Plan covers create/join/lobby/game/reconnect/loading/empty/error/disabled/kicked/full/expired/cancelled/success/result states, accessible names, keyboard focus, live-region behavior for roster/reconnect changes, and use of existing Tailwind/shadcn/Radix patterns.
- **Localization and RTL**: PASS. All room, lobby, presence, timer, reconnect, host-control, error, and result copy is planned in `client/messages/en.json` and `client/messages/ar.json`; Arabic layout must use existing direction handling.
- **Performance budgets**: PASS. Budgets are stated for room commands, live event delivery, initial room load, reconnect restore, bounded database queries, Redis event/snapshot cost, and frontend bundle impact. Measurement comes from backend tests/metrics/logs plus frontend build and browser validation.
- **Contracts and data**: PASS. Existing room paths in `backend/openapi/openapi.yaml` will be completed/expanded; the feature contract artifact documents REST responses and WebSocket event envelope. Durable tables already exist for MVP, with migrations only for additional indexes/columns discovered during implementation.
- **Operational readiness**: PASS. Plan covers `.env.example` additions for realtime URL/grace/heartbeat settings, CORS/origin checks, rate limits, structured logs without hidden coordinates or tokens, metrics for rooms/realtime/guesses/reconnects, readiness impact for Redis, and degraded behavior when live delivery is unavailable.

## Project Structure

### Documentation (this feature)

```text
specs/006-room-realtime-reconnection/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── rooms-realtime-openapi.md
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
│   ├── health/
│   ├── middleware/
│   ├── platform/
│   ├── realtime/
│   └── rooms/
├── migrations/
└── openapi/

client/
├── app/
│   └── [locale]/
│       └── rooms/
├── features/
│   ├── game/
│   └── rooms/
├── lib/
│   └── api/
├── messages/
└── stores/

specs/006-room-realtime-reconnection/
└── [planning and validation artifacts]
```

**Structure Decision**: Use feature packages for `rooms` and `realtime` because private room commands, lobby state, presence, event delivery, and reconnect windows are distinct from solo game logic. Keep shared gameplay facts and scoring in `games`; expose only narrow interfaces such as room game creation/start/guess progression to avoid `games -> rooms -> games` import cycles. Frontend room routes live under localized `client/app/[locale]/rooms`, with server-only API helpers and narrow Client Components for realtime/browser state.

## Complexity Tracking

No constitution violations.

## Phase 0 Research

See [research.md](./research.md).

## Phase 1 Design

See [data-model.md](./data-model.md), [contracts/rooms-realtime-openapi.md](./contracts/rooms-realtime-openapi.md), and [quickstart.md](./quickstart.md).

## Post-Design Constitution Check

- **Architecture boundaries**: PASS. Design artifacts keep REST/realtime contracts in planning, durable facts in existing PostgreSQL tables plus any future Goose migrations, active coordination in Redis-backed abstractions, backend behavior in `internal/rooms`, `internal/realtime`, and targeted `internal/games` changes, and frontend room UI in localized app/features/lib/message paths.
- **Framework guidance**: PASS. Design follows the installed Server/Client Component, data fetching, data security, forms, and internationalization guidance read during planning.
- **Testing gates**: PASS. Design maps room lifecycle, host permissions, hidden-coordinate safety, WebSocket event ordering/reconnect, timer deadlines, and UI states to automated or recorded validation.
- **UX consistency**: PASS. Design identifies room create/join/lobby/game/reconnect/error/result states and accessibility expectations for roster updates, reconnect announcements, controls, and timers.
- **Localization and RTL**: PASS. Design keeps all visible room copy in `en`/`ar` message catalogs and uses existing direction handling for Arabic.
- **Performance budgets**: PASS. Data model and contracts keep room snapshots/event versions compact, use Redis for active state, avoid polling, and retain bounded PostgreSQL reads for authoritative reloads.
- **Contracts and data**: PASS. Contract artifact lists REST room updates, WebSocket event envelope, error codes, and OpenAPI updates. Durable schema reuse is explicit; migration needs are bounded to implementation-discovered gaps.
- **Operational readiness**: PASS. Research, data model, and quickstart cover config, origin/rate-limit/security behavior, logs/metrics/redaction, readiness impact, and degraded mode when live delivery is unavailable.
