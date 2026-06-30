# Tasks: Room Realtime Reconnection

**Input**: Design documents from `specs/006-room-realtime-reconnection/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/rooms-realtime-openapi.md](./contracts/rooms-realtime-openapi.md), [quickstart.md](./quickstart.md)

**Tests**: Required by the GeoGuess constitution for backend behavior, API contracts, Redis coordination, frontend UI states, localization, accessibility, and end-to-end room flows. Story test tasks are listed before implementation tasks and should fail before the matching implementation when the failure can be reproduced locally.

**Organization**: Tasks are grouped by user story so each story can be implemented and validated as an independent increment after the shared foundation is complete.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with other tasks in the same phase because it touches different files or only depends on completed foundation work
- **[Story]**: User story label for traceability; setup, foundation, and polish tasks do not use story labels
- **File paths**: Every task includes concrete repository paths

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add required dependencies, configuration, and test harnesses before feature code starts.

- [X] T001 Add `github.com/coder/websocket` dependency for backend realtime transport in `backend/go.mod` and `backend/go.sum`
- [X] T002 [P] Add frontend component test harness and scripts for room UI tests in `client/package.json`, `client/vitest.config.ts`, and `client/test/setup.ts`
- [X] T003 [P] Add backend room realtime configuration defaults and validation in `backend/internal/config/config.go`, `backend/internal/config/config_test.go`, and `backend/.env.example`
- [X] T004 [P] Add browser realtime URL configuration in `client/lib/env.ts` and `client/.env.example`
- [X] T005 [P] Add room membership lookup indexes with Goose in `backend/migrations/00010_room_realtime_indexes.sql`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared room, realtime, Redis, contract, and wiring primitives that every user story depends on.

**Critical**: No user story work should begin until this phase is complete.

- [X] T006 [P] Define room domain models, constants, and state transition helpers in `backend/internal/rooms/model.go`
- [X] T007 [P] Define room request/response DTOs with hidden-coordinate-safe shapes in `backend/internal/rooms/dto.go`
- [X] T008 [P] Define stable room service errors and HTTP error mapping helpers in `backend/internal/rooms/errors.go`
- [X] T009 [P] Define realtime event envelope, event types, and defensive JSON validation in `backend/internal/realtime/event.go`
- [X] T010 [P] Implement Redis room coordination primitives for versions, snapshots, presence, reconnect windows, ready state, locks, pub/sub, and idempotency in `backend/internal/platform/redis/rooms.go`
- [X] T011 [P] Add unit tests for Redis room coordination key formats, TTLs, lock behavior, and version increments in `backend/internal/platform/redis/rooms_test.go`
- [X] T012 [P] Add privacy-safe room and realtime metrics recorders in `backend/internal/rooms/metrics.go` and `backend/internal/realtime/metrics.go`
- [X] T013 Implement base room repository queries and transactions against existing `rooms`, `room_players`, `games`, and `game_players` tables in `backend/internal/rooms/repository.go`
- [X] T014 Implement base room service constructor, dependencies, identity extraction, validation helpers, and snapshot builder in `backend/internal/rooms/service.go`
- [X] T015 Implement base room HTTP handler constructor, request parsing, response helpers, and route registration in `backend/internal/rooms/handler.go`
- [X] T016 Implement base realtime hub and WebSocket handler constructor in `backend/internal/realtime/hub.go` and `backend/internal/realtime/handler.go`
- [X] T017 Wire room and realtime dependencies into application startup and router signatures in `backend/cmd/api/main.go`, `backend/internal/app/server.go`, and `backend/internal/app/routes.go`
- [X] T018 Expand shared OpenAPI room schemas, error codes, and realtime documentation anchors in `backend/openapi/openapi.yaml`
- [X] T019 [P] Add frontend room API types and server-only REST helpers in `client/features/rooms/types.ts` and `client/lib/api/rooms.ts`
- [X] T020 [P] Add frontend realtime envelope types and reducer utilities in `client/features/rooms/realtime-types.ts` and `client/features/rooms/room-state.ts`

**Checkpoint**: Foundation ready. Story phases can now proceed in priority order or in parallel by separate implementers.

---

## Phase 3: User Story 1 - Host A Private Realtime Room (Priority: P1) MVP

**Goal**: A guest or signed-in host can create a private room, share a room code, and see lobby membership and presence update live as players join, disconnect, reconnect, or leave.

**Independent Test**: Create a room in one browser session, join from a second session, and confirm both sessions show the same code, settings, host identity, player list, presence status, and room state without manual refresh.

### Tests for User Story 1

- [X] T021 [P] [US1] Add failing room service tests for code generation, create room, join room, duplicate identity prevention, and guest/registered identity matching in `backend/internal/rooms/service_test.go`
- [X] T022 [P] [US1] Add failing room repository tests for create/join/rejoin transactions, room-code uniqueness, capacity checks, and room history preservation in `backend/internal/rooms/repository_test.go`
- [X] T023 [P] [US1] Add failing realtime handler tests for participant authentication, initial `room.snapshot`, heartbeat handling, and disconnect events in `backend/internal/realtime/handler_test.go`
- [X] T024 [P] [US1] Add failing HTTP handler tests for `POST /api/v1/rooms`, `POST /api/v1/rooms/join`, and `GET /api/v1/rooms/{roomCode}` in `backend/internal/rooms/handler_test.go`
- [X] T025 [P] [US1] Add failing frontend tests for create, join, roster, copy invite, and live presence states in `client/features/rooms/lobby.test.tsx`

### Implementation for User Story 1

- [X] T026 [US1] Implement non-enumerable uppercase room code generation and create-room service behavior in `backend/internal/rooms/service.go`
- [X] T027 [US1] Implement create-room transaction that creates `games`, host `game_players`, `rooms`, and `room_players` records in `backend/internal/rooms/repository.go`
- [X] T028 [US1] Implement join/rejoin transaction with capacity, status, kicked-player, duplicate registered user, and duplicate guest identity checks in `backend/internal/rooms/repository.go`
- [X] T029 [US1] Implement authoritative room state loading with safe roster, presence, settings, version, and no hidden coordinates in `backend/internal/rooms/service.go`
- [X] T030 [US1] Implement create, join, and get room HTTP handlers with stable error responses in `backend/internal/rooms/handler.go`
- [X] T031 [US1] Publish room snapshots, version increments, join events, leave events, and presence events through Redis in `backend/internal/platform/redis/rooms.go`
- [X] T032 [US1] Implement WebSocket connect, participant authorization, initial snapshot write, heartbeat read loop, and room pub/sub fanout in `backend/internal/realtime/handler.go` and `backend/internal/realtime/hub.go`
- [X] T033 [US1] Register room REST routes, realtime WebSocket route, and room creation/join/connection rate limits in `backend/internal/app/routes.go`
- [X] T034 [US1] Implement localized room create/join page in `client/app/[locale]/rooms/page.tsx`
- [X] T035 [US1] Implement localized room lobby page with server-loaded room state in `client/app/[locale]/rooms/[roomCode]/page.tsx`
- [X] T036 [US1] Implement browser room channel client with reconnecting/degraded connection states and authoritative refetch callback in `client/features/rooms/realtime-client.ts`
- [X] T037 [US1] Implement create/join form, invite copy control, lobby roster, and presence indicators in `client/features/rooms/create-join-form.tsx`, `client/features/rooms/lobby.tsx`, and `client/features/rooms/presence-badge.tsx`
- [X] T038 [US1] Add English and Arabic copy for room create, join, lobby, invite, roster, and presence states in `client/messages/en.json` and `client/messages/ar.json`

**Checkpoint**: User Story 1 is independently functional and testable.

---

## Phase 4: User Story 2 - Enforce Host Controls And Room Rules (Priority: P1)

**Goal**: The host can update lobby settings, remove players, ready/reset readiness, and start the room while non-host players cannot mutate privileged controls.

**Independent Test**: Attempt settings changes, player removal, ready state updates, and start commands as host and non-host sessions; only host-authorized actions succeed and all connected clients receive updated state.

### Tests for User Story 2

- [X] T039 [P] [US2] Add failing service tests for host authorization, settings validation, settings lock after start, remove-player rules, ready resets, and start idempotency in `backend/internal/rooms/service_test.go`
- [X] T040 [P] [US2] Add failing handler tests for `PATCH /api/v1/rooms/{roomCode}/settings`, `POST /api/v1/rooms/{roomCode}/start`, and `DELETE /api/v1/rooms/{roomCode}/players/{playerId}` in `backend/internal/rooms/handler_test.go`
- [X] T041 [P] [US2] Add failing frontend tests for host controls, non-host disabled states, ready toggles, and removed-player behavior in `client/features/rooms/host-controls.test.tsx`

### Implementation for User Story 2

- [X] T042 [US2] Implement host authorization, lobby-only settings validation, ready state reset, remove-player, and start command orchestration in `backend/internal/rooms/service.go`
- [X] T043 [US2] Implement settings update, player removal, role transfer support, and room-start persistence helpers in `backend/internal/rooms/repository.go`
- [X] T044 [US2] Implement settings, ready, start, and remove-player handlers with `Idempotency-Key` enforcement for start in `backend/internal/rooms/handler.go`
- [X] T045 [US2] Implement Redis ready-state storage, ready reset, host-command idempotency, and room lock usage in `backend/internal/platform/redis/rooms.go`
- [X] T046 [US2] Publish `room.settings_updated`, `room.ready_updated`, `room.ready_reset`, `room.player_removed`, and `room.started` events in `backend/internal/rooms/service.go`
- [X] T047 [US2] Expand OpenAPI request/response schemas and error codes for settings, start, ready, and remove-player commands in `backend/openapi/openapi.yaml`
- [X] T048 [US2] Implement frontend host settings, ready toggle, start room, remove player, and non-host read-only controls in `client/features/rooms/host-controls.tsx`
- [X] T049 [US2] Add frontend server actions or form action wrappers for room host commands in `client/features/rooms/actions.ts`
- [X] T050 [US2] Add English and Arabic copy for host controls, forbidden errors, removed-player states, ready states, and locked settings in `client/messages/en.json` and `client/messages/ar.json`

**Checkpoint**: User Stories 1 and 2 are independently functional and testable.

---

## Phase 5: User Story 3 - Play Synchronized Multiplayer Rounds (Priority: P1)

**Goal**: Players in an active private room receive synchronized round starts, server-owned countdowns, aggregate guess progress, result reveal, next-round transitions, and final results.

**Independent Test**: Start a two-player room, submit guesses from both sessions, and confirm round start time, deadline, guess count, reveal, next round, and final scores stay consistent across both clients.

### Tests for User Story 3

- [X] T051 [P] [US3] Add failing multiplayer game progression tests for shared start time, all-submitted completion, deadline completion, final results, and zero-point missed rounds in `backend/internal/games/multiplayer_test.go`
- [X] T052 [P] [US3] Add failing room service tests for room start, first round event publication, guess progress event publication, and final completion in `backend/internal/rooms/service_test.go`
- [X] T053 [P] [US3] Add failing hidden-coordinate contract tests for room state, `round.started`, and `round.guess_count_changed` payloads in `backend/internal/rooms/handler_test.go`
- [X] T054 [P] [US3] Add failing frontend tests for room gameplay, countdown, aggregate guess progress, reveal, next-round transition, and scoreboard states in `client/features/rooms/room-game.test.tsx`

### Implementation for User Story 3

- [X] T055 [US3] Add private-room game creation and start helpers that preserve synchronized server timestamps in `backend/internal/games/multiplayer.go` and `backend/internal/games/repository.go`
- [X] T056 [US3] Extend room start flow to create/link a private-room game, create rounds, activate round one, and publish safe start events in `backend/internal/rooms/service.go`
- [X] T057 [US3] Extend guess submission for `private_room` mode to accept one guess per player without completing the round until all active eligible players submit or the deadline closes in `backend/internal/games/service.go` and `backend/internal/games/multiplayer.go`
- [X] T058 [US3] Add multiplayer repository transactions for guess progress, all-submitted checks, deadline closure, missed-round zeroes, next-round activation, and final game completion in `backend/internal/games/repository.go`
- [X] T059 [US3] Publish `round.started`, `round.guess_count_changed`, `round.ended`, `round.results_revealed`, and `game.completed` events from room/game progression in `backend/internal/rooms/service.go`
- [X] T060 [US3] Add safe current-round and guess-progress DTO mapping that excludes `location_id`, answer coordinates, and answer-revealing metadata before reveal in `backend/internal/rooms/dto.go`
- [X] T061 [US3] Update game and room HTTP handlers to expose active private-room state and results only to authorized room participants in `backend/internal/games/handler.go` and `backend/internal/rooms/handler.go`
- [X] T062 [US3] Expand OpenAPI schemas for `RoomCurrentRound`, `RoomGuessProgress`, round events, result reveal, and hidden-coordinate guarantees in `backend/openapi/openapi.yaml`
- [X] T063 [US3] Implement active room gameplay surface using existing Street View and guess map primitives in `client/features/rooms/room-game.tsx`
- [X] T064 [US3] Implement server-timestamp countdown and deadline state display in `client/features/rooms/room-countdown.tsx`
- [X] T065 [US3] Implement aggregate guess progress and hidden-answer-safe status display in `client/features/rooms/guess-progress.tsx`
- [X] T066 [US3] Implement round reveal, next-round transition, and final scoreboard components in `client/features/rooms/room-results.tsx`
- [X] T067 [US3] Integrate lobby-to-game and completed-result rendering on the room page in `client/app/[locale]/rooms/[roomCode]/page.tsx`
- [X] T068 [US3] Add English and Arabic copy for room start, round countdown, guessing, progress, reveal, score, next round, and final result states in `client/messages/en.json` and `client/messages/ar.json`

**Checkpoint**: User Stories 1, 2, and 3 provide the complete P1 private-room gameplay path.

---

## Phase 6: User Story 4 - Recover From Reconnects And Refreshes (Priority: P2)

**Goal**: A player can refresh or briefly lose realtime connectivity and return to the correct room, round, guess, and score state without duplicating their participant slot or blocking gameplay.

**Independent Test**: Disconnect one player during lobby and active round states, reconnect before and after the round deadline, and confirm restored state, preserved guesses, missed-round scoring, and roster status.

### Tests for User Story 4

- [X] T069 [P] [US4] Add failing Redis presence and reconnect-window tests for 30-second grace, heartbeat expiry, matching identity, and stale window rejection in `backend/internal/platform/redis/rooms_test.go`
- [X] T070 [P] [US4] Add failing realtime reconnect tests for reconnect snapshot, duplicate connection replacement, stale version handling, and non-participant rejection in `backend/internal/realtime/handler_test.go`
- [X] T071 [P] [US4] Add failing room service tests for lobby refresh, active-round refresh, host disconnect transfer, duplicate-tab start races, and preserved submitted guesses in `backend/internal/rooms/service_test.go`
- [X] T072 [P] [US4] Add failing frontend tests for reconnecting, degraded, refetch-on-version-mismatch, already-submitted, and missed-round states in `client/features/rooms/realtime-client.test.ts` and `client/features/rooms/recovery.test.tsx`

### Implementation for User Story 4

- [X] T073 [US4] Implement Redis reconnect windows, heartbeat TTL refresh, participant presence expiry, duplicate connection replacement, and identity-bound reconnect checks in `backend/internal/platform/redis/rooms.go`
- [X] T074 [US4] Implement realtime reconnect flow, version mismatch snapshot behavior, stale-event deduplication, and degraded close codes in `backend/internal/realtime/handler.go`
- [X] T075 [US4] Implement room service refresh/reconnect semantics that restore the same participant slot for registered and guest identities in `backend/internal/rooms/service.go`
- [X] T076 [US4] Implement lobby host transfer to earliest joined active player after reconnect grace and active-gameplay no-pause behavior in `backend/internal/rooms/service.go` and `backend/internal/rooms/repository.go`
- [X] T077 [US4] Ensure timed round closure assigns missed-round zeroes without blocking connected players in `backend/internal/games/multiplayer.go`
- [X] T078 [US4] Implement frontend recovery banner, reconnect retry, degraded-state refresh, and unsafe-action disabling in `client/features/rooms/recovery.tsx`
- [X] T079 [US4] Implement client event deduplication, version-gap detection, and authoritative refetch behavior in `client/features/rooms/realtime-client.ts` and `client/features/rooms/room-state.ts`
- [X] T080 [US4] Add English and Arabic copy for reconnecting, degraded, disconnected, restored, missed-round, and host-transfer states in `client/messages/en.json` and `client/messages/ar.json`

**Checkpoint**: User Story 4 reconnect behavior is independently functional and testable.

---

## Phase 7: User Story 5 - Handle Room Failure And Recovery States (Priority: P3)

**Goal**: Players see clear localized and accessible states for invalid codes, expired rooms, full rooms, authorization failures, reconnecting, disconnected, kicked, loading, empty, disabled, success, and unexpected errors.

**Independent Test**: Exercise each boundary condition and confirm the UI shows localized, accessible copy with an appropriate next action and no unsafe state mutation.

### Tests for User Story 5

- [X] T081 [P] [US5] Add failing handler tests for invalid code, expired room, full room, active room, cancelled room, kicked player, identity mismatch, and realtime origin rejection in `backend/internal/rooms/handler_test.go` and `backend/internal/realtime/handler_test.go`
- [X] T082 [P] [US5] Add failing rate-limit tests for room creation, join attempts, host commands, realtime connection attempts, and guess submission in `backend/internal/app/routes_test.go`
- [X] T083 [P] [US5] Add failing frontend tests for invalid, expired, full, kicked, authorization, loading, empty, disabled, success, Arabic RTL, and accessible status states in `client/features/rooms/room-states.test.tsx`

### Implementation for User Story 5

- [X] T084 [US5] Complete stable error mapping for room and realtime failures in `backend/internal/rooms/errors.go` and `backend/internal/realtime/errors.go`
- [X] T085 [US5] Enforce room-code normalization, enumeration-resistant join failures, and abuse rate limits in `backend/internal/rooms/handler.go` and `backend/internal/app/routes.go`
- [X] T086 [US5] Add realtime origin checks, auth-required failures, kicked/removed connection closure, and privacy-safe error events in `backend/internal/realtime/handler.go`
- [X] T087 [US5] Implement shared frontend room state surfaces for invalid code, expired, full, kicked, unauthorized, loading, empty, disabled, success, and unexpected errors in `client/features/rooms/room-states.tsx`
- [X] T088 [US5] Ensure room controls expose accessible names, keyboard focus, disabled semantics, live-region announcements, and non-color-only status indicators in `client/features/rooms/create-join-form.tsx`, `client/features/rooms/lobby.tsx`, `client/features/rooms/host-controls.tsx`, and `client/features/rooms/recovery.tsx`
- [X] T089 [US5] Complete English and Arabic translations for all room failure and recovery states in `client/messages/en.json` and `client/messages/ar.json`

**Checkpoint**: All user stories are independently functional and testable.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final contract, security, performance, operations, and release-readiness checks across all stories.

- [X] T090 [P] Add privacy-safe structured logs for create, join, disconnect, reconnect, start, round transitions, rejected late guesses, rejected room codes, and realtime delivery failures in `backend/internal/rooms/service.go`, `backend/internal/realtime/handler.go`, and `backend/internal/games/multiplayer.go`
- [X] T091 [P] Add Prometheus metrics for active rooms, active realtime connections, joins, disconnects, reconnects, room starts, round transitions, guess outcomes, and realtime delivery failures in `backend/internal/rooms/metrics.go`, `backend/internal/realtime/metrics.go`, and `backend/internal/platform/observability/observability.go`
- [X] T092 [P] Update Phase 06 quickstart with exact Docker, migration, backend, frontend, and two-browser validation evidence steps in `specs/006-room-realtime-reconnection/quickstart.md`
- [X] T093 [P] Run and record OpenAPI validation with `npx pnpm@10.24.0 check:openapi` from `package.json`
- [X] T094 Run backend formatting and full test gate with `gofmt` and `go test ./...` for files under `backend/`
- [X] T095 Run backend lint gate with `golangci-lint run` using `backend/.golangci.yml`
- [X] T096 Run frontend lint gate with `npx pnpm@10.24.0 --dir client lint` for files under `client/`
- [X] T097 Run frontend typecheck gate with `npx pnpm@10.24.0 --dir client typecheck` for files under `client/`
- [X] T098 Run frontend build gate with `npx pnpm@10.24.0 --dir client build` for files under `client/`
- [X] T099 Run frontend room component tests with `npx pnpm@10.24.0 --dir client test` for `client/features/rooms/`
- [X] T100 Run backend migrations against the configured `.env` PostgreSQL and verify Redis connectivity using `backend/migrations/` and `backend/internal/health/pingers.go`
- [X] T101 Run Docker backend validation and confirm the updated API serves room and realtime routes using `docker-compose.yml`, `backend/cmd/api/main.go`, and `backend/internal/app/routes.go`
- [X] T102 Record two-session happy path, reconnect path, hidden-coordinate check, English localization check, and Arabic RTL/accessibility check evidence in `specs/006-room-realtime-reconnection/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- Phase 1 Setup has no dependencies and can start immediately.
- Phase 2 Foundational depends on Phase 1 and blocks every user story.
- Phase 3 US1 depends on Phase 2 and is the first MVP increment.
- Phase 4 US2 depends on Phase 2 and integrates naturally after US1 lobby state.
- Phase 5 US3 depends on Phase 2 plus the room lifecycle and host start behavior from US1/US2.
- Phase 6 US4 depends on Phase 2 and benefits from US1/US3 state flows.
- Phase 7 US5 depends on Phase 2 and can harden each completed story increment.
- Phase 8 Polish depends on the selected story set being complete.

### User Story Dependencies

- US1: Can start after Foundation and delivers private room create/join/lobby realtime MVP.
- US2: Can start after Foundation, but host-control UI is most useful after US1 lobby exists.
- US3: Requires private room lifecycle and host start path from US1/US2 to deliver synchronized gameplay.
- US4: Requires room membership, presence, and gameplay state paths from US1/US3.
- US5: Can be developed alongside later stories, but final acceptance needs all error-producing paths.

### Within Each User Story

- Write failing tests first when behavior is locally reproducible.
- Backend model/DTO/error primitives before repository/service/handler implementation.
- Service logic before route wiring and frontend integration.
- Realtime event publication before frontend live-state application.
- Localized copy and accessibility states before story checkpoint validation.

---

## Parallel Opportunities

- Setup tasks T002, T003, T004, and T005 can run in parallel after T001 is understood.
- Foundational tasks T006 through T012, T019, and T020 can run in parallel after setup.
- Tests within each user story marked [P] can be written in parallel.
- Backend and frontend implementation tasks in a story can proceed in parallel once shared contracts are stable.
- US4 reconnect and US5 recovery hardening can overlap after US1 establishes room membership and realtime events.

### Parallel Example: User Story 1

```text
Task: T021 service tests in backend/internal/rooms/service_test.go
Task: T022 repository tests in backend/internal/rooms/repository_test.go
Task: T023 realtime tests in backend/internal/realtime/handler_test.go
Task: T024 handler tests in backend/internal/rooms/handler_test.go
Task: T025 frontend lobby tests in client/features/rooms/lobby.test.tsx
```

### Parallel Example: User Story 3

```text
Task: T051 multiplayer progression tests in backend/internal/games/multiplayer_test.go
Task: T052 room service event tests in backend/internal/rooms/service_test.go
Task: T053 hidden-coordinate handler tests in backend/internal/rooms/handler_test.go
Task: T054 frontend gameplay tests in client/features/rooms/room-game.test.tsx
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1 Setup.
2. Complete Phase 2 Foundational.
3. Complete Phase 3 User Story 1.
4. Stop and validate: two sessions can create/join a private room and receive live lobby updates.

### P1 Complete Private Room Gameplay

1. Add Phase 4 User Story 2 host controls.
2. Add Phase 5 User Story 3 synchronized multiplayer rounds.
3. Validate the full two-player happy path from room creation through final results.

### Recovery And Release Readiness

1. Add Phase 6 User Story 4 reconnect/refresh behavior.
2. Add Phase 7 User Story 5 failure, localization, RTL, and accessibility states.
3. Complete Phase 8 gates and record evidence in `quickstart.md`.

---

## Notes

- Keep room and game authoritative writes on HTTP endpoints; realtime events are versioned hints.
- Do not expose `location_id`, answer coordinates, or answer-revealing provider metadata before authorized reveal.
- Do not use GORM AutoMigrate; all schema changes belong in Goose migrations under `backend/migrations/`.
- Use `client/node_modules/next/dist/docs/` as the source of truth before changing Next.js App Router APIs.
- Use Redis only for ephemeral active coordination; PostgreSQL remains the durable source of room and game facts.
