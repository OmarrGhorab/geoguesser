# Implementation Plan: Game Mode Pages

**Branch**: 014-game-mode-pages | **Date**: 2026-07-19 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from /specs/014-game-mode-pages/spec.md

## Summary

Deliver complete localized entry, gameplay, recovery, and result pages for the
eleven player-facing modes already exposed in the authenticated navigation:
Classic Solo, Practice, Daily Challenge, Quick Play, Party Lobby, and Casual or
Ranked Solo/Duo/Squad.

The frontend will use a capability-driven gameplay shell and separate domain
controllers for games, rooms, parties, matchmaking, and matchplay. Initial
authoritative reads stay in Server Components through server-only native-fetch
modules; Server Actions handle low-frequency commands; narrow Client Components
own maps, timers, WebSockets, chat, and other browser interactions. Existing
Daily Mission UI is extracted and reused rather than copied.

Most backend behavior already exists. Contract-first backend work closes the
confirmed gaps: operational Quick Play with backend-owned defaults, Party Lobby
self-leave and revealed-round recovery, production-safe room realtime origins,
accurate matchmaking party_version documentation, and aligned idempotency
contracts. Legacy aliases remain recoverable but never selectable.

## Technical Context

**Language/Version**: TypeScript 5.9, React 19.2, Next.js 16.2.10; Go 1.25
module targeting Go 1.24+ production conventions

**Primary Dependencies**: Next.js App Router, next-intl 4, Tailwind CSS 4,
shadcn/Base UI, Zod 4, native fetch, Google Maps JavaScript API; Chi 5, GORM,
coder/websocket, Redis, Prometheus

**Storage**: PostgreSQL for games, rooms, attempts, parties, matches, results,
progression, and durable creation receipts; Redis for queues, presence,
short-lived claims, fanout, and one-time realtime tickets

**Testing**: Go test/Testify/integration tests and benchmarks; Vitest,
Testing Library-compatible component tests, Playwright multi-context E2E,
OpenAPI validation, ESLint, TypeScript, Next production build

**Target Platform**: Modern desktop/mobile browsers and Linux-hosted Next.js and
Go services; localized English and Arabic with RTL

**Project Type**: Full-stack web application with a Next.js frontend and Go API

**Performance Goals**: Entry shells render within 2 seconds p95; ordinary API
commands within 1 second p95; Party/Practice snapshots below 300 ms p95; guesses
below 500 ms p95; assignment visible within 3 seconds; authorized realtime
fanout below 250 ms and visible UI updates within 1 second

**Constraints**: No hidden-answer leakage; no duplicate create/queue/guess/start
effects; HTTP snapshots remain authoritative; bounded queries only; no canonical
server state in Zustand; added initial client JavaScript below 25 KB gzip per
entry route and shared active-game shell below 70 KB gzip excluding Maps SDK;
existing extracted mode icons total about 36 KB

**Scale/Scope**: Eleven mode entries, five new localized route families plus the
existing Daily family, guest and registered sessions, Party Lobby up to 50
players, Practice histories beyond 1,000 rounds through cursor paging, and six
matchmaking combinations with team sizes 1, 2, and 4

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Architecture boundaries — PASS**: Feature keeps frontend work in client/,
  backend work in backend/, planning in specs/, backend packages under internal/,
  executable setup in cmd/api, infrastructure adapters in internal/platform/,
  and database changes in Goose migrations. Production code does not use GORM
  AutoMigrate.
- **Framework guidance — PASS**: Implementation must re-check the installed
  Next.js guides for layouts/pages, Server and Client Components, data fetching,
  data mutation, dynamic routes, navigation, and error handling under
  client/node_modules/next/dist/docs/01-app/ before framework-sensitive edits.
  This plan follows Next 16 async params/searchParams and leaf-level client
  boundaries.
- **Testing gates — PASS**: Unit, contract, PostgreSQL/Redis integration,
  component, accessibility, and browser-flow coverage is defined below and in
  quickstart.md. Required commands are backend go test ./..., frontend pnpm
  test:unit, pnpm test:e2e, pnpm lint, pnpm typecheck, and pnpm build, plus
  OpenAPI validation and Linux race checks for concurrent backend packages.
- **UX consistency — PASS**: Plan covers loading, empty, error, disabled,
  success, reconnecting, stale, and terminal states; keyboard focus, semantic
  markup, accessible names, and reuse of Tailwind v4, shadcn/ui, and Base UI
  patterns before adding primitives.
- **Localization and RTL — PASS**: All copy is added to en and ar message
  catalogs. Logical spacing/alignment, mirrored directional controls, stable
  numeric score reading, focus order, dialogs, maps, and responsive layouts are
  explicitly tested in RTL.
- **Performance budgets — PASS**: The measurable API, render, bundle, realtime,
  query, and asset budgets above are checked through backend benchmarks,
  Prometheus histograms, production bundle output, Lighthouse/Playwright traces,
  and query-count assertions.
- **Contracts and data — PASS**: Endpoint and schema changes update
  backend/openapi/openapi.yaml. Durable creation receipts use one forward-only
  Goose migration; existing quick_play database mode support requires no enum
  migration.
- **Operational readiness — PASS**: New Quick Play defaults and corrected room
  realtime origin configuration are documented in backend/.env.example and
  validated at startup. Rollout flags, safe bounded telemetry, security,
  readiness, and rollback behavior are covered below.

### Post-design re-check

Phase 1 artifacts retain all PASS results. The route contract assigns one
backend authority per surface, data-model.md identifies the only new durable
record, and quickstart.md contains every required verification command. There
are no accepted constitution violations.

## Delivery Phases

### Phase A — Contract and mode registry foundation

1. Add one typed player-facing registry for the eleven canonical choices,
   capabilities, route builders, session requirements, team sizes, and localized
   message keys. Normalize private_room, ranked_standard, and ranked only at
   inbound recovery boundaries.
2. Correct OpenAPI drift before frontend consumers are added: narrow ordinary
   game creation modes, add party_version to team queue entry, align actually
   enforced idempotency headers, and document Party Lobby leave/reveal recovery
   and Quick Play.
3. Add Zod DTOs and fixtures matching OpenAPI for maps, games, Practice history,
   daily metadata/attempts, rooms, parties, queue status, match snapshots,
   revealed rounds, terminal results, and realtime tickets.
4. Replace the current unconditional generic game-to-daily redirect with an
   authoritative mode-aware dispatcher.

### Phase B — Backend gaps and hardening

1. Add GameModeQuickPlay and POST /games/quick-play. It resolves the configured
   active public map, creates and starts a five-round, 60-second game atomically,
   persists mode quick_play, and returns the resolved game configuration. It
   requires a stable idempotency key and a durable command receipt so retrying
   cannot create a second game.
2. Apply the same receipt mechanism to ordinary game and room creation where
   OpenAPI promises create idempotency. Keep state-idempotent start/end/leave
   behavior explicit and remove unenforced header promises.
3. Add DELETE /rooms/{roomCode}/players/me for member self-leave and
   DELETE /rooms/{roomCode} for host cancellation while still in the lobby.
   Active-host departure continues through the existing disconnect/grace and
   terminal policies; self-leave never silently strands host authority.
4. Mount and document participant-authorized
   GET /games/{gameId}/rounds/{roundId}/results so refresh during Party Lobby
   reveal reconstructs the exact shared result.
5. Make ROOM_REALTIME_ALLOWED_HOST drive the room WebSocket origin allow-list,
   preserve guest cookie sessions safely, and keep ticketed party/match sockets
   on one-time subprotocol tickets.
6. Add bounded room and daily metrics and regression tests for privacy,
   idempotency, authorization, version gaps, disabled modes, and concurrency.
7. Keep the existing five-games-per-day Daily policy; expose clear played/total
   progress and only show the completed day state after game five.

### Phase C — Shared gameplay and result foundation

1. Extract the Google Maps loader, Street View scene, guess map, round HUD,
   countdown, controls, reveal overlay, attribution, and result map from the
   current Daily components into client/features/gameplay.
2. Drive the shell through explicit capabilities: timed, open-ended,
   shared-reveal, teams, speed bonus, progression, chat, and host controls.
   Avoid a single giant mode switch.
3. Build pure lifecycle reducers for game, room, queue, party, and match states.
   Event versions are hints: ignore old/duplicate versions and refetch an HTTP
   snapshot on gaps, reconnect, or degraded realtime.
4. Add shared loading/error/not-found boundaries, localized safe error mapping,
   retry/exit affordances, aria-live updates, focus restoration, reduced motion,
   media fallback, and mobile/RTL layouts.

### Phase D — Solo, Quick Play, Daily, and Practice pages

1. Build /[locale]/play with validated mode selection. Classic Solo exposes map,
   round, and timer settings; Practice exposes map choice and its untimed rules;
   Quick Play exposes one primary action and displays the server-resolved five
   rounds/60 seconds/map configuration before gameplay.
2. Build generic /[locale]/games/[gameId] and
   /[locale]/games/[gameId]/results pages for Solo and Quick Play, while
   dispatching Daily to its semantic route and rendering active Practice as a
   history/next/end workflow.
3. Preserve /[locale]/daily-mission, its play route, and its result route as the
   canonical Daily experience; adapt them to shared primitives and display
   progress out of five.
4. Cover guest and registered ownership, refresh at every round phase,
   double-submission prevention, unavailable maps/media, paged Practice history,
   and explicit Practice end.

### Phase E — Party Lobby pages

1. Build /[locale]/rooms for create and join-by-code and
   /[locale]/rooms/[roomCode] as a state-driven lobby, active game, reveal, and
   standings page.
2. Implement settings, readiness, share/copy, start, kick, leave/cancel, current
   player role, capacity, expiry, and legacy private_room recovery.
3. Hydrate from HTTP first, then connect realtime. On connection loss or version
   gap, announce degraded state and repair from room/game snapshots.
4. Verify two-session and 50-player behavior, host/member permissions, shared
   reveal privacy, deterministic ties, and terminal refresh.

### Phase F — Matchmaking, premade parties, and matchplay

1. Build /[locale]/matchmaking with six stable mode selections. The page always
   reads current queue and party state before enabling entry and redirects an
   existing assignment to its server-provided destination.
2. Build /[locale]/parties/[partyId] for Duo/Squad invitations, exact roster
   readiness, leadership, eligibility, version conflicts, and queue entry.
3. Build /[locale]/matches/[matchId] and results pages from snapshot-first,
   ticketed realtime state. Cover countdown, active guess, submitted/spectating,
   shared reveal, next round, reconnect grace, leave/forfeit, terminal state,
   and ranked progression pending/applied.
4. Add Duo/Squad team totals and optional chat/mute/report UI. Gate image chat
   separately and never delay core matchmaking delivery on it.
5. Roll out Casual, then Ranked Solo, then Ranked teams by configuration. A
   disabled entry remains explainable and active durable assignments remain
   recoverable during rollback.

### Phase G — Verification and release

1. Run contract, unit, integration, concurrency, component, accessibility, RTL,
   responsive, multi-context E2E, performance, and required repository gates.
2. Capture Prometheus and bundle evidence against stated budgets; run live
   multi-client reconnect and Party Lobby soak checks.
3. Deploy backend contracts and recovery routes before enabling their frontend
   entries. Enable Quick Play and multiplayer flags progressively.
4. Roll back by disabling new entry flags while retaining all durable reads and
   deep-link recovery; never down-migrate creation receipts or existing
   multiplayer migrations while referenced data exists.

## Project Structure

### Documentation (this feature)

    specs/014-game-mode-pages/
    ├── plan.md
    ├── research.md
    ├── data-model.md
    ├── quickstart.md
    ├── contracts/
    │   └── game-mode-pages-contract.md
    ├── checklists/
    │   └── requirements.md
    └── tasks.md                 # Created later by /speckit-tasks

### Source Code (repository root)

    backend/
    ├── cmd/api/
    ├── internal/
    │   ├── app/
    │   ├── challenges/
    │   ├── config/
    │   ├── games/
    │   ├── matchmaking/
    │   ├── matchplay/
    │   ├── parties/
    │   ├── realtime/
    │   └── rooms/
    ├── migrations/
    └── openapi/openapi.yaml

    client/
    ├── app/[locale]/
    │   ├── play/
    │   ├── games/[gameId]/
    │   ├── daily-mission/
    │   ├── rooms/
    │   ├── parties/[partyId]/
    │   ├── matchmaking/
    │   └── matches/[matchId]/
    ├── app/api/
    │   ├── games/
    │   ├── matches/
    │   └── realtime/
    ├── components/ui/
    ├── features/
    │   ├── gameplay/
    │   ├── play/
    │   ├── mission/
    │   ├── rooms/
    │   ├── party/
    │   ├── matchmaking/
    │   └── matchplay/
    ├── lib/api/
    ├── messages/
    └── tests/e2e/

**Structure Decision**: Use the established two-project GeoGuess layout. Backend
domains remain authoritative and independent; the frontend composes them
through feature folders and a shared gameplay presentation layer. Route files
stay thin and localized, server-only data modules own initial fetches, Server
Actions own low-frequency commands, and same-origin handlers bridge repeated
browser mutations without exposing backend session cookies.

## Complexity Tracking

No constitution violations are accepted.
