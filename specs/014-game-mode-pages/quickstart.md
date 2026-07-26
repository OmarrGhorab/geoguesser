# Quickstart: Game Mode Pages

## Prerequisites

- Docker Desktop or equivalent PostgreSQL and Redis services
- Go version compatible with backend/go.mod
- Node.js compatible with Next.js 16 and pnpm 10.24.0
- Google Maps browser key for interactive gameplay
- Seeded active public maps with enough locations

Review these installed Next.js documents before framework-sensitive frontend
edits:

- client/node_modules/next/dist/docs/01-app/01-getting-started/03-layouts-and-pages.md
- client/node_modules/next/dist/docs/01-app/01-getting-started/05-server-and-client-components.md
- client/node_modules/next/dist/docs/01-app/01-getting-started/06-fetching-data.md
- client/node_modules/next/dist/docs/01-app/01-getting-started/07-mutating-data.md
- client/node_modules/next/dist/docs/01-app/01-getting-started/10-error-handling.md

## Configuration

Copy the existing environment examples and set the normal database, Redis,
session, CSRF, maps, media, and public origin values. Feature 014 adds or
requires explicit verification of:

- QUICK_PLAY_DEFAULT_MAP_ID: active public map UUID with at least five locations
- QUICK_PLAY_ROUND_COUNT: 5 by default; validate 1–10
- QUICK_PLAY_TIMER_SECONDS: 60 by default; validate 10–600
- COMMAND_RECEIPT_TTL_SECONDS: 86400 default
- ROOM_REALTIME_ALLOWED_HOST: deployed browser host allowed for room sockets
- CASUAL_MATCHMAKING_ENABLED: staged rollout, initially false
- RANKED_TEAM_MODES_ENABLED: staged rollout, initially false
- TEAM_CHAT_IMAGES_ENABLED: independent optional rollout

Update backend/.env.example for every new value. Startup validation must fail
with an actionable message when an enabled Quick Play configuration is invalid.
Health/readiness must report required PostgreSQL and Redis failures using the
existing policy; disabled optional mode entry does not make the whole API
unready.

## Recommended implementation checkpoints

### 1. Contracts and backend gaps

- Add the Goose command-receipt migration.
- Add Quick Play config, runtime mode, service, handler, route, OpenAPI, metrics,
  and tests.
- Add Party Lobby self-leave and mount shared-round result recovery.
- Fix matchmaking party_version and idempotency documentation.
- Fix room WebSocket allowed-origin configuration.

Checkpoint:

    cd backend
    go test ./internal/games ./internal/rooms ./internal/matchmaking ./internal/realtime
    go test ./internal/app

### 2. Shared gameplay

- Extract shared map/Street View and results primitives from Daily.
- Add mode registry, capabilities, DTO schemas, routes, lifecycle reducers, and
  safe error mapping.
- Fix generic game route dispatch.

Checkpoint:

    cd client
    pnpm test:unit
    pnpm lint
    pnpm typecheck

### 3. Solo, Quick Play, Daily, Practice

- Complete setup, active-game, history, and result states.
- Verify guest and registered sessions and five-game Daily progress.

### 4. Party Lobby

- Complete create/join/lobby/game/reveal/result states.
- Test two browser contexts, reconnect, leave, and host permissions.

### 5. Matchmaking and matches

- Complete six selections, premade parties, queue recovery, active match, and
  terminal result pages.
- Enable Casual first, then Ranked Solo, then Ranked teams.

## Automated verification

Run all required repository gates before release:

    cd backend
    go test ./...

    cd ../client
    pnpm test:unit
    pnpm test:e2e
    pnpm lint
    pnpm typecheck
    pnpm build

Also run:

- OpenAPI syntax/schema validation and request/response fixture checks.
- PostgreSQL integration tests for command receipts, game/room creation,
  Practice history, Party Lobby reveal recovery, and match terminal recovery.
- Redis integration tests for queue claims, presence, reconnect, ticket
  single-use, and disposable-state recovery.
- Linux CI race tests for matchmaking, realtime, rooms, and matchplay.
- Benchmarks/Prometheus assertions for stated p95 budgets.
- Bundle analyzer or Next production build evidence for per-route and shared
  gameplay JavaScript budgets.

## Primary browser journeys

Use English and Arabic variants where marked. Exercise mobile, desktop, keyboard
only, reduced motion, and 200% zoom across the suite.

1. Classic Solo: choose map/rounds/timer, start, guess, reveal, refresh, finish,
   inspect results, replay.
2. Quick Play: double-click/retry the one-action start, verify one active game,
   resolved 5x60 configuration, finish, refresh result.
3. Practice: start untimed, guess/reveal, create exactly one next round, paginate
   history, refresh after 1,000+ seeded rounds, end explicitly.
4. Daily: start/resume, timeout and guess paths, complete games 1–5, verify
   progress and day-complete state across devices.
5. Party Lobby: two contexts create/join, settings, readiness, host start,
   participant leave, socket loss/version gap, shared reveal, tied standings,
   terminal refresh, legacy private_room recovery.
6. Casual Solo and Ranked Solo: join/leave/reload queue, receive assignment,
   countdown/timer rules, submit/spectate/reveal, disconnect grace, terminal and
   progression states.
7. Casual and Ranked Duo: create exact party, invite, ready, stale party version,
   queue, team score/reveal/chat, leave/forfeit, terminal.
8. Casual and Ranked Squad: repeat with four members and validate roster,
   capacity, team privacy, and responsive layout.
9. Recovery/security: unauthorized deep links, missing/expired sessions,
   duplicate idempotency keys, rate limits, disabled flags, Redis interruption,
   no hidden answers before reveal, and no sensitive log fields.

## Manual accessibility and UX checks

- Traverse every action by keyboard with visible focus.
- Verify dialogs trap/restore focus and async changes use polite/assertive live
  regions appropriately.
- Confirm map controls have accessible names and non-map fallback instructions.
- Confirm Arabic uses logical layout, sensible score ordering, mirrored
  direction-sensitive icons, and equivalent focus order.
- Confirm loading, empty, error, disabled, success, reconnecting, stale,
  progression-pending, and terminal states are visually distinct.
- Confirm media attribution remains visible and unavailable imagery does not
  block safe exit/retry.

## Performance and operational evidence

Record:

- p50/p95/p99 latency by bounded mode class and operation.
- Queue formation time, realtime fanout delay, reconnect/snapshot-repair time,
  and stale-version count.
- Party Lobby snapshot size at 50 players and Practice query time at 1,000+
  rounds.
- Initial route JavaScript and shared gameplay JavaScript excluding the
  separately loaded Maps SDK.
- Error/rate-limit/degraded-realtime counts with bounded labels.

Never record raw session material, idempotency keys, room codes, ticket values,
Redis keys, chat bodies, user identities, precise guesses, hidden coordinates,
or provider secrets.

## Rollout and rollback

1. Deploy migration and backend contract corrections.
2. Enable internal Quick Play and validate create/retry/recovery.
3. Release Solo/Daily/Practice pages.
4. Release Party Lobby and run multi-client soak.
5. Enable Casual, then Ranked Solo, then Ranked teams.
6. Enable chat images separately after moderation/storage verification.

Rollback disables entry/creation flags while preserving reads, reconnects, and
terminal results for existing resources. Do not down-migrate command receipts,
00019, or 00020 while referenced games, parties, matches, chat, ratings, Party
Lobby, or Practice data exists.
