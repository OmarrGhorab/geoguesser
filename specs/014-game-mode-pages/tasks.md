# Tasks: Game Mode Pages

**Input**: Design documents from /specs/014-game-mode-pages/

**Prerequisites**: plan.md, spec.md, research.md, data-model.md,
contracts/game-mode-pages-contract.md, quickstart.md

**Tests**: Tests are required by FR-030 and the project constitution. Within each
story, test tasks precede implementation and should fail for the missing
behavior before production code is added.

**Organization**: Tasks are grouped by user story so each story can be
implemented, exercised, and accepted as an independent increment.

## Format

- [P] means the task can run in parallel with adjacent tasks because it targets
  separate files and does not depend on unfinished work.
- [US1] through [US4] map directly to the four stories in spec.md.
- Every task includes the concrete implementation or evidence path.

## Phase 1: Setup and Baseline

**Purpose**: Confirm framework guidance, local dependencies, and current quality
state before changing shared code.

- [ ] T001 Re-read the installed Next.js 16 layouts/pages, Server/Client Components, fetching, mutation, navigation, dynamic-route, and error-handling guides and update any changed guidance references in specs/014-game-mode-pages/quickstart.md
- [ ] T002 Run the existing backend and frontend baseline gates and record command, result, date, and pre-existing failures in specs/014-game-mode-pages/implementation-evidence.md
- [ ] T003 [P] Verify PostgreSQL, Redis, eligible map seeds, Google Maps browser configuration, and current mode rollout flags and record the reproducible local setup in specs/014-game-mode-pages/implementation-evidence.md

**Checkpoint**: Baseline behavior and environment are documented without changing
feature behavior.

---

## Phase 2: Foundational Contracts and Shared Infrastructure

**Purpose**: Establish canonical modes, durable create idempotency, shared
frontend contracts, and reusable gameplay primitives before any story page is
implemented.

**Critical**: User story implementation begins only after this phase passes its
targeted tests.

### Foundation tests

- [X] T004 [P] Add exact-eleven-mode, legacy-alias exclusion, capability, and localized-route tests in client/features/gameplay/modes.test.ts
- [X] T005 [P] Add success, nullable-field, stable-status, legacy-recovery, and safe-error fixture tests in client/features/gameplay/schemas.test.ts
- [X] T006 [P] Add lifecycle and version-gap reducer tests for game, room, queue, party, and match projections in client/features/gameplay/state.test.ts
- [ ] T007 [P] Add durable replay, conflicting-body, actor-isolation, expiry, and raw-secret exclusion tests in backend/internal/commandreceipts/repository_test.go
- [ ] T008 [P] Add migration up/down, uniqueness, expiry-index, and resource-kind constraint coverage in backend/internal/app/game_mode_pages_migration_test.go

### Foundation implementation

- [ ] T009 Add the command_receipts table, uniqueness constraint, bounded expiry index, and safe down migration in backend/migrations/00021_game_mode_command_receipts.sql
- [ ] T010 Implement CommandReceipt model, actor/key/request hashing, replay/conflict rules, and repository transactions in backend/internal/commandreceipts/model.go and backend/internal/commandreceipts/repository.go
- [ ] T011 Define the creation-receipt service interface and wire its repository through backend/internal/commandreceipts/service.go and backend/cmd/api/main.go
- [X] T012 Implement the canonical eleven-entry registry, capabilities, legacy normalization, session requirements, team sizes, and message keys in client/features/gameplay/modes.ts
- [X] T013 [P] Implement localized route builders and safe backend-destination localization in client/features/gameplay/routes.ts
- [X] T014 Implement shared Zod DTOs for games, rounds, Practice history, rooms, parties, queues, matches, results, and realtime tickets in client/features/gameplay/schemas.ts and client/features/gameplay/types.ts
- [X] T015 [P] Implement localized stable API error categories and recovery actions in client/features/gameplay/errors.ts
- [X] T016 Extract the safe Google Maps loader and provider types from client/features/mission/google-maps.ts into client/features/gameplay/google-maps.ts and update existing Daily imports
- [X] T017 Extract SceneViewer, GuessMap, RoundHud, Countdown, RevealOverlay, MapControls, attribution, and media fallback from client/features/mission/components/daily-game-screen.tsx into client/features/gameplay/components/scene-viewer.tsx, client/features/gameplay/components/guess-map.tsx, client/features/gameplay/components/round-hud.tsx, client/features/gameplay/components/reveal-overlay.tsx, and client/features/gameplay/components/map-controls.tsx
- [X] T018 Extract reusable result mapping and round breakdowns from client/features/mission/components/daily-results-map.tsx and client/features/mission/components/daily-results-screen.tsx into client/features/gameplay/components/results-map.tsx and client/features/gameplay/components/round-breakdown.tsx
- [X] T019 Implement pure capability-driven lifecycle reducers and snapshot-repair rules in client/features/gameplay/state.ts
- [X] T020 Implement a shared same-origin backend mutation proxy with cookie, CSRF, no-store, Zod response, idempotency-key, Retry-After, and safe-error handling in client/lib/api/mutation-proxy.ts

**Checkpoint**: Canonical modes, contracts, create receipts, shared gameplay
primitives, and reducers pass their targeted tests.

---

## Phase 3: User Story 1 — Start a Solo Experience (Priority: P1) MVP

**Goal**: Complete Classic Solo, Quick Play, Practice, and the existing
five-game Daily Challenge through setup, active rounds, refresh recovery, and
appropriate results/continuation.

**Independent Test**: From the Play menu, start each of the four experiences,
complete at least one round, refresh during play/reveal, and verify its distinct
timer, progression, reveal, continuation, and terminal rules.

### Tests for User Story 1

- [ ] T021 [P] [US1] Add Quick Play constant, policy, default-validation, and Solo-compatible reveal/scoring tests in backend/internal/games/quick_play_test.go
- [ ] T022 [P] [US1] Add atomic create/start, same-key replay, conflict, unavailable-map, and guest/account isolation integration tests in backend/internal/games/quick_play_repository_test.go
- [ ] T023 [P] [US1] Add POST /games/quick-play request, auth, CSRF, rate-limit, status, and safe-error handler tests in backend/internal/games/handler_test.go
- [ ] T024 [P] [US1] Add ordinary Solo/Practice create receipt replay and conflict regression tests in backend/internal/games/service_test.go
- [X] T025 [P] [US1] Add play-mode query, form validation, action result, and route-destination tests in client/features/play/schemas.test.ts and client/features/play/routes.test.ts
- [X] T026 [P] [US1] Add Solo, Quick Play, Practice, and Daily capability-adapter tests in client/features/play/game-adapter.test.ts
- [X] T027 [P] [US1] Add generic game dispatch tests for solo, quick_play, practice, daily, private_room, ranked, unsupported, forbidden, and terminal records in client/features/play/dispatch.test.ts
- [ ] T028 [P] [US1] Add five-game Daily progress and completion regression fixtures in client/features/mission/data.test.ts and client/features/mission/schemas.test.ts
- [ ] T029 [P] [US1] Add English/Arabic Playwright journeys for Solo, Quick Play retry, Practice history/end, Daily resume, keyboard flow, and refresh checkpoints in client/tests/e2e/solo-modes.spec.ts

### Backend implementation for User Story 1

- [ ] T030 [P] [US1] Add Quick Play defaults, feature enablement, receipt TTL parsing, and startup validation in backend/internal/config/config.go and backend/internal/config/config_test.go
- [ ] T031 [P] [US1] Document QUICK_PLAY_DEFAULT_MAP_ID, QUICK_PLAY_ROUND_COUNT, QUICK_PLAY_TIMER_SECONDS, and COMMAND_RECEIPT_TTL_SECONDS in backend/.env.example
- [ ] T032 [US1] Add GameModeQuickPlay and Solo-compatible lifecycle, reveal, scoring, progression, and multiplayer classification rules in backend/internal/games/state.go
- [ ] T033 [US1] Add QuickPlayResponse DTO and atomic create/start repository transaction in backend/internal/games/dto.go and backend/internal/games/repository.go
- [ ] T034 [US1] Implement StartQuickPlay with configured map eligibility, five-round/60-second defaults, durable receipt replay/conflict, safe logs, and metrics in backend/internal/games/service.go and backend/internal/games/metrics.go
- [ ] T035 [US1] Wire POST /api/v1/games/quick-play with session, CSRF, rate limiting, and error mapping in backend/internal/games/handler.go and backend/internal/app/routes.go
- [ ] T036 [US1] Apply durable creation receipts to existing POST /games Solo/Practice creation and align create/start/end idempotency behavior in backend/internal/games/service.go and backend/internal/games/handler.go
- [ ] T037 [US1] Narrow CreateGameRequest.mode, add Quick Play operation/schemas/errors, and align idempotency requirements in backend/openapi/openapi.yaml

### Frontend implementation for User Story 1

- [X] T038 [P] [US1] Implement map/game/round/history/result server-only reads and Zod validation in client/features/play/data.ts and client/features/play/schemas.ts
- [X] T039 [P] [US1] Implement stable-key create/start/next/end Server Actions with typed expected errors in client/features/play/actions.ts
- [X] T040 [P] [US1] Implement same-origin guess and Daily timeout handlers in client/app/api/games/[gameId]/rounds/[roundId]/guesses/route.ts and client/app/api/games/[gameId]/rounds/[roundId]/timeout/route.ts
- [X] T041 [US1] Build the localized Classic Solo, Quick Play, and Practice setup page with rules, eligibility, map/configuration states, and double-submit protection in client/app/[locale]/play/page.tsx and client/features/play/components/play-setup-screen.tsx
- [X] T042 [US1] Implement Solo, Quick Play, and Practice capability adapters over the shared gameplay shell in client/features/play/game-adapter.ts and client/features/play/components/solo-game-screen.tsx
- [X] T043 [US1] Replace the unconditional Daily redirect with authoritative mode-aware dispatch and render active Solo/Quick Play/Practice states in client/app/[locale]/games/[gameId]/page.tsx and client/features/play/dispatch.ts
- [X] T044 [US1] Build fixed-game results and active/ended Practice history, next-round, pagination, and end-session states in client/app/[locale]/games/[gameId]/results/page.tsx and client/features/play/components/game-results-screen.tsx
- [X] T045 [US1] Refactor Daily play/results onto shared gameplay components while preserving semantic routes, timeouts, five-of-five progress, and completed-day behavior in client/features/mission/components/daily-game-screen.tsx and client/features/mission/components/daily-results-screen.tsx
- [X] T046 [US1] Add loading, error, and not-found boundaries for play and generic game/result routes in client/app/[locale]/play/loading.tsx, client/app/[locale]/play/error.tsx, client/app/[locale]/games/[gameId]/loading.tsx, client/app/[locale]/games/[gameId]/error.tsx, and client/app/[locale]/games/[gameId]/not-found.tsx
- [X] T047 [US1] Add all Solo/Quick Play/Practice/Daily visible copy, statuses, validation, accessible names, media alternatives, and Arabic RTL strings in client/messages/en.json and client/messages/ar.json

**Checkpoint**: User Story 1 passes backend tests and the four-mode independent
browser journey and is suitable as the first deployable MVP.

---

## Phase 4: User Story 2 — Enter Casual and Ranked Matchmaking (Priority: P1)

**Goal**: Complete all six registered-only queue choices, exact premade
Duo/Squad preparation, assignment recovery, active matchplay, team reveal, and
Casual/Ranked terminal results.

**Independent Test**: Exercise each Casual/Ranked Solo/Duo/Squad combination
with valid and invalid rosters, reload queue/assignment state, play a match, and
verify team scoring, hidden-answer policy, leave/forfeit, and progression.

### Tests for User Story 2

- [ ] T048 [P] [US2] Add OpenAPI regression coverage for canonical playlist/format, required Duo/Squad party_id plus party_version, status unions, destinations, and no false idempotency promise in backend/internal/app/matchmaking_openapi_contract_test.go
- [ ] T049 [P] [US2] Add queue selection, eligibility, active-ticket override, and safe destination tests in client/features/matchmaking/schemas.test.ts and client/features/matchmaking/routes.test.ts
- [ ] T050 [P] [US2] Add party roster/version/readiness and typed action-state tests in client/features/party/schemas.test.ts and client/features/party/actions.test.ts
- [ ] T051 [P] [US2] Add queue and match lifecycle tests for duplicate/old/gapped events, HTTP repair, progression pending, reconnect grace, and terminal replay in client/features/matchplay/state.test.ts
- [ ] T052 [P] [US2] Add pre-reveal redaction, unauthorized not-found, revealed-round, terminal, and rating fixture tests in client/features/matchplay/schemas.test.ts
- [ ] T053 [P] [US2] Add Server Action and Route Handler tests for queue join/leave, party commands, guess, realtime ticket, leave, chat, mute, and report in client/features/matchmaking/actions.test.ts and client/app/api/matches/match-routes.test.ts
- [ ] T054 [P] [US2] Add multi-context Playwright journeys for Casual/Ranked Solo, Duo, and Squad including invalid rosters, reload, assignment, reveal, reconnect, forfeit, and results in client/tests/e2e/matchmaking-modes.spec.ts

### Backend contract alignment for User Story 2

- [ ] T055 [US2] Add party_version to EnterMatchmakingRequest, narrow canonical mode examples, and align queue join/leave/status idempotency and error documentation in backend/openapi/openapi.yaml
- [ ] T056 [P] [US2] Add bounded mode/playlist/format/operation/outcome metrics and redaction assertions for queue, party, match, and realtime paths in backend/internal/matchmaking/metrics.go and backend/internal/matchplay/metrics.go
- [ ] T057 [US2] Verify six-mode disabled/enabled behavior and durable active-assignment recovery through configuration regression tests in backend/internal/app/casual_ranked_workers_test.go and backend/internal/config/config_test.go

### Frontend implementation for User Story 2

- [ ] T058 [P] [US2] Implement canonical matchmaking, party, match snapshot, round result, terminal result, chat, and ticket Zod contracts in client/features/matchmaking/schemas.ts, client/features/party/schemas.ts, and client/features/matchplay/schemas.ts
- [ ] T059 [P] [US2] Implement server-only queue/status, current party/invites, match snapshot, revealed round, terminal result, and competitive profile reads in client/features/matchmaking/data.ts, client/features/party/data.ts, and client/features/matchplay/data.ts
- [ ] T060 [P] [US2] Implement queue join/leave and premade party create/invite/readiness/kick/leave/disband Server Actions with current party_version in client/features/matchmaking/actions.ts and client/features/party/actions.ts
- [ ] T061 [P] [US2] Implement same-origin match guess, leave, realtime-ticket, chat, mute, report, and attachment handlers in client/app/api/matches/[matchId]/rounds/[roundId]/guesses/route.ts, client/app/api/matches/[matchId]/leave/route.ts, client/app/api/matches/[matchId]/messages/route.ts, client/app/api/matches/[matchId]/mutes/[userId]/route.ts, client/app/api/matches/[matchId]/messages/[messageId]/report/route.ts, client/app/api/matches/[matchId]/messages/[messageId]/attachment/route.ts, and client/app/api/realtime/tickets/route.ts
- [ ] T062 [US2] Build the six-choice localized matchmaking page with eligibility, roster requirements, disabled flags, not-queued/searching/matched/unavailable states, and server-destination recovery in client/app/[locale]/matchmaking/page.tsx and client/features/matchmaking/components/matchmaking-screen.tsx
- [ ] T063 [US2] Build the Duo/Squad party preparation page with invites, exact roster, readiness, leadership, stale-version recovery, and queue controls in client/app/[locale]/parties/[partyId]/page.tsx and client/features/party/components/party-screen.tsx
- [ ] T064 [US2] Implement snapshot-first ticketed WebSocket connection, version sequencing, reconnect backoff, and HTTP repair in client/features/matchplay/realtime.ts and client/features/matchplay/state.ts
- [ ] T065 [US2] Build the active match screen for countdown, timer policy, guess locking, submitted spectating, team scores, shared reveal, reconnect grace, and leave/forfeit in client/app/[locale]/matches/[matchId]/page.tsx and client/features/matchplay/components/match-screen.tsx
- [ ] T066 [US2] Build terminal Casual and Ranked result states including per-round team breakdown, rating delta, progression_pending retry, and safe replay navigation in client/app/[locale]/matches/[matchId]/results/page.tsx and client/features/matchplay/components/match-results-screen.tsx
- [ ] T067 [US2] Add Duo/Squad team chat, mute, report, attachment gating, keyboard focus, and post-terminal retention UI in client/features/matchplay/components/team-chat.tsx
- [ ] T068 [US2] Add loading, error, and not-found boundaries in client/app/[locale]/matchmaking/loading.tsx, client/app/[locale]/matchmaking/error.tsx, client/app/[locale]/parties/[partyId]/loading.tsx, client/app/[locale]/parties/[partyId]/error.tsx, client/app/[locale]/matches/[matchId]/loading.tsx, client/app/[locale]/matches/[matchId]/error.tsx, and client/app/[locale]/matches/[matchId]/not-found.tsx
- [ ] T069 [US2] Add all Casual/Ranked/party/queue/match/chat visible copy, asynchronous announcements, errors, and Arabic RTL strings in client/messages/en.json and client/messages/ar.json

**Checkpoint**: All six queue modes are independently selectable and recoverable,
with Solo/Duo/Squad match journeys passing against authoritative backend state.

---

## Phase 5: User Story 3 — Host or Join a Party Lobby (Priority: P1)

**Goal**: Complete guest/account Party Lobby creation, join, controls, realtime
recovery, shared game, self-leave/host cancellation, and deterministic standings.

**Independent Test**: Use two browser sessions to create/join a lobby, exercise
host/member permissions, disconnect/reconnect, play every round, leave/cancel
where allowed, and verify tied final standings.

### Tests for User Story 3

- [ ] T070 [P] [US3] Add member self-leave, host-action-required, host cancellation, terminal replay, authorization, and concurrency service tests in backend/internal/rooms/service_test.go
- [ ] T071 [P] [US3] Add room leave/cancel request, status, CSRF, rate-limit, and privacy-safe handler tests in backend/internal/rooms/handler_test.go
- [ ] T072 [P] [US3] Add mounted shared-round result participant authorization, pre-reveal conflict, redaction, and refresh-recovery tests in backend/internal/games/handler_test.go and backend/internal/app/party_lobby_practice_modes_test.go
- [ ] T073 [P] [US3] Add configured-origin allow/deny, guest cookie, reconnect, version-gap, and localhost-development regression tests in backend/internal/realtime/handler_test.go
- [ ] T074 [P] [US3] Add room create receipt replay/conflict and actor isolation integration tests in backend/internal/rooms/repository_test.go
- [ ] T075 [P] [US3] Add room DTO, state reducer, role permission, legacy private_room, and shared-reveal fixture tests in client/features/rooms/schemas.test.ts and client/features/rooms/state.test.ts
- [ ] T076 [P] [US3] Add Server Action tests for create/join/settings/ready/start/kick/leave/cancel and stable retry keys in client/features/rooms/actions.test.ts
- [ ] T077 [P] [US3] Add English/Arabic two-context Playwright coverage for create/join/ready/start/reconnect/reveal/leave/cancel/tied standings in client/tests/e2e/party-lobby.spec.ts

### Backend implementation for User Story 3

- [ ] T078 [US3] Implement non-host self-leave and lobby-host cancellation transactions, version updates, idempotent terminal behavior, and safe events in backend/internal/rooms/repository.go and backend/internal/rooms/service.go
- [ ] T079 [US3] Add DELETE /rooms/{roomCode}/players/me and DELETE /rooms/{roomCode} handlers, rate limits, CSRF behavior, and error mapping in backend/internal/rooms/handler.go and backend/internal/app/routes.go
- [ ] T080 [US3] Apply durable command receipts to POST /rooms creation and return replay/conflict results in backend/internal/rooms/service.go and backend/internal/rooms/handler.go
- [ ] T081 [US3] Mount GET /games/{gameId}/rounds/{roundId}/results through the real router and retain participant-only reveal policy in backend/internal/app/routes.go and backend/internal/games/handler.go
- [ ] T082 [US3] Replace hard-coded room WebSocket origin checks with validated ROOM_REALTIME_ALLOWED_HOST configuration and safe guest/account session handling in backend/internal/realtime/handler.go and backend/internal/config/config.go
- [ ] T083 [US3] Add bounded room snapshot/command/reconnect metrics without room codes or identity labels in backend/internal/rooms/metrics.go and backend/cmd/api/main.go
- [ ] T084 [US3] Document room leave/cancel, shared-round recovery, room receipt idempotency, origin security, events, and errors in backend/openapi/openapi.yaml

### Frontend implementation for User Story 3

- [ ] T085 [P] [US3] Implement server-only map and room snapshot reads plus room DTO validation in client/features/rooms/data.ts and client/features/rooms/schemas.ts
- [ ] T086 [P] [US3] Implement create/join/settings/ready/start/kick/self-leave/cancel Server Actions with stable keys and expected-error states in client/features/rooms/actions.ts
- [ ] T087 [US3] Build localized Party Lobby create and join-by-code forms with validation, map/settings defaults, capacity, loading, and double-submit protection in client/app/[locale]/rooms/page.tsx and client/features/rooms/components/room-entry-screen.tsx
- [ ] T088 [US3] Implement HTTP-first room WebSocket state, monotonic versions, reconnect announcements, stale/gap repair, and presence handling in client/features/rooms/realtime.ts and client/features/rooms/state.ts
- [ ] T089 [US3] Build the state-driven lobby/active/reveal/completed page with share code, host/member controls, readiness, progress, shared gameplay, standings, leave, and cancellation in client/app/[locale]/rooms/[roomCode]/page.tsx and client/features/rooms/components/room-screen.tsx
- [ ] T090 [US3] Add loading, error, not-found, expired, full, cancelled, media-fallback, and terminal boundaries in client/app/[locale]/rooms/[roomCode]/loading.tsx, client/app/[locale]/rooms/[roomCode]/error.tsx, and client/app/[locale]/rooms/[roomCode]/not-found.tsx
- [ ] T091 [US3] Add all Party Lobby rules, roles, statuses, controls, announcements, safe errors, and Arabic RTL strings in client/messages/en.json and client/messages/ar.json

**Checkpoint**: Party Lobby is independently playable and recoverable for guest
and registered users without relying on matchmaking-party UI.

---

## Phase 6: User Story 4 — Navigate and Recover Across Modes (Priority: P2)

**Goal**: Make the eleven-mode system consistent across navigation, deep links,
locales, viewports, accessibility modes, stale sessions, and degraded networks.

**Independent Test**: Deep-link to every setup, active, queue, lobby, and result
state in English and Arabic; refresh it; verify safe authorization/recovery,
responsive layout, keyboard operation, focus, and exit navigation.

### Tests for User Story 4

- [ ] T092 [P] [US4] Extend exact-eleven-destination, two-interaction reachability, disabled-state, and legacy-alias exclusion tests in client/features/home/nav-config.test.ts
- [ ] T093 [P] [US4] Add cross-mode deep-link dispatch, locale preservation, unsupported-mode, terminal redirect, and safe destination tests in client/features/gameplay/routes.test.ts
- [ ] T094 [P] [US4] Add loading/empty/error/disabled/success/reconnecting/stale/terminal component-state and keyboard/focus tests in client/features/gameplay/components/gameplay-states.test.tsx
- [ ] T095 [P] [US4] Add automated English/Arabic RTL, mobile/tablet/desktop, reduced-motion, keyboard-only, and 200-percent-zoom journeys in client/tests/e2e/game-mode-recovery.spec.ts
- [ ] T096 [P] [US4] Add unauthorized/missing/expired/legacy/degraded-realtime security regression fixtures with hidden-data assertions in client/tests/e2e/game-mode-security.spec.ts

### Implementation for User Story 4

- [ ] T097 [US4] Reconcile authenticated navbar and home-card destinations with the canonical registry and feature availability in client/features/home/nav-config.ts and client/features/home/components/game-mode-menu.tsx
- [ ] T098 [US4] Complete generic game dispatch for Daily, Party Lobby/private_room, canonical matches, legacy ranked, completed sessions, and unsupported values in client/app/[locale]/games/[gameId]/page.tsx and client/features/play/dispatch.ts
- [ ] T099 [P] [US4] Implement shared semantic loading, empty, error, disabled, success, reconnecting, stale, progression-pending, and terminal components in client/features/gameplay/components/gameplay-states.tsx
- [ ] T100 [US4] Add route-family scoped error/not-found providers with localized message subsets in client/app/[locale]/games/[gameId]/layout.tsx, client/app/[locale]/rooms/[roomCode]/layout.tsx, and client/app/[locale]/matches/[matchId]/layout.tsx
- [ ] T101 [P] [US4] Apply logical CSS, mirrored directional controls, stable numeric score presentation, responsive breakpoints, 200-percent zoom support, visible focus, and reduced motion in client/features/gameplay/components/round-hud.tsx, client/features/gameplay/components/map-controls.tsx, client/features/gameplay/components/round-breakdown.tsx, client/features/gameplay/components/gameplay-states.tsx, and client/app/globals.css
- [ ] T102 [P] [US4] Add consistent localized breadcrumbs, safe exit/resume/retry actions, and async live-region announcements in client/features/play/components/play-setup-screen.tsx, client/features/play/components/solo-game-screen.tsx, client/features/rooms/components/room-screen.tsx, client/features/matchmaking/components/matchmaking-screen.tsx, and client/features/matchplay/components/match-screen.tsx
- [ ] T103 [US4] Ensure navbar selection, query mode, locale changes, deep links, and backend destinations never replace authoritative active state in client/features/gameplay/routes.ts and client/lib/i18n/navigation.ts

**Checkpoint**: Every mode is reachable, localized, accessible, responsive, and
recoverable through the same cross-mode UX rules.

---

## Phase 7: Polish, Performance, and Release Readiness

**Purpose**: Prove the complete feature against constitutional quality,
performance, security, operations, and rollout gates.

- [ ] T104 [P] Add OpenAPI parse/reference/path/schema validation for every changed operation in backend/internal/app/openapi_contract_test.go
- [ ] T105 [P] Add Practice 1,000-round cursor and Party Lobby 50-player snapshot/query benchmarks in backend/internal/app/party_lobby_practice_benchmark_test.go
- [ ] T106 [P] Add queue/status/formation, guess, authorized fanout, reconnect, and snapshot-repair performance assertions in backend/internal/app/casual_ranked_benchmark_test.go and backend/internal/matchmaking/service_benchmark_test.go
- [ ] T107 [P] Add bounded Daily and Party Lobby mode/operation/outcome/degraded-realtime dashboards or recording-rule documentation in docs/observability/game-mode-pages.md
- [ ] T108 Update backend/postman/GeoGuess-API.postman_collection.json and backend/postman/README.md with Quick Play, room leave/cancel, round recovery, party_version, and all eleven page journeys without overwriting unrelated existing examples
- [ ] T109 Run migration 00021 up/down/up against a disposable PostgreSQL database and record evidence and production rollback constraints in specs/014-game-mode-pages/implementation-evidence.md
- [ ] T110 Run backend go test ./..., targeted Linux race checks for matchmaking/realtime/rooms/matchplay, and OpenAPI validation and record results in specs/014-game-mode-pages/implementation-evidence.md
- [ ] T111 Run frontend pnpm test:unit, pnpm test:e2e, pnpm lint, pnpm typecheck, and pnpm build and record results in specs/014-game-mode-pages/implementation-evidence.md
- [ ] T112 Measure entry-route and shared-gameplay JavaScript, Maps lazy loading, 50-player room rendering, reconnect UI latency, and stated API p95 budgets and record evidence in specs/014-game-mode-pages/implementation-evidence.md
- [ ] T113 Perform manual keyboard, screen-reader, focus restoration, Arabic RTL, reduced-motion, mobile, desktop, 200-percent zoom, and media-attribution checks and record evidence in specs/014-game-mode-pages/implementation-evidence.md
- [ ] T114 Audit logs, metrics, errors, WebSocket handshakes, and client payloads for tokens, raw keys, room codes, identity, chat bodies, guesses, precise coordinates, and hidden answers and record findings in specs/014-game-mode-pages/implementation-evidence.md
- [ ] T115 Document staged enablement for Quick Play, Casual, Ranked Solo, Ranked teams, and optional chat images plus read-compatible rollback in docs/runbooks/game-mode-pages-rollout.md
- [ ] T116 Reconcile completed implementation evidence against every FR/SC and mark the feature status ready only after all required gates pass in specs/014-game-mode-pages/spec.md and specs/014-game-mode-pages/implementation-evidence.md

---

## Dependencies and Execution Order

### Phase dependencies

- Phase 1 has no dependencies.
- Phase 2 depends on the baseline and blocks every user story.
- US1, US2, and US3 all depend on Phase 2. Their story-specific backend and
  frontend work can proceed in parallel once shared contracts and components are
  stable.
- US4 depends on the route families delivered by the selected P1 stories. It can
  begin registry/navigation tests after Phase 2, but final cross-mode acceptance
  requires US1, US2, and US3.
- Phase 7 depends on every story selected for release.

### User story dependency graph

    Setup
      |
    Foundation
      |-----------------------|-----------------------|
     US1                     US2                     US3
      |-----------------------|-----------------------|
                              US4
                               |
                            Release

### Within each story

- Write and run its behavior tests first; confirm they fail for the missing
  behavior before implementing it.
- Complete model/repository/config work before services.
- Complete services before handlers, routes, and OpenAPI.
- Complete schemas/data/actions before page composition.
- Hydrate realtime pages from HTTP before applying WebSocket hints.
- Pass the independent story checkpoint before beginning release polish.

## Parallel Execution Examples

### User Story 1

- Backend tasks T021–T024 can be authored in parallel with frontend tests
  T025–T029.
- After Quick Play config/state is available, T033–T037 proceed in order while
  T038–T040 can run in parallel.
- Page composition T041–T047 follows the shared gameplay foundation and client
  data/action contracts.

### User Story 2

- Contract, party, reducer, schema, action, and Playwright tests T048–T054 can be
  authored concurrently in separate files.
- Backend alignment T055–T057 can proceed alongside frontend contracts, data,
  actions, and route handlers T058–T061.
- Matchmaking, party, active match, results, and chat pages T062–T069 integrate
  after their respective data/action layers exist.

### User Story 3

- Backend behavior tests T070–T074 and frontend tests T075–T077 can be written
  concurrently.
- Room lifecycle work T078–T084 can proceed alongside client data/actions
  T085–T086.
- Entry, realtime, lobby/game, boundary, and copy tasks T087–T091 then compose
  the independently tested pieces.

### User Story 4

- Navigation, routing, component-state, accessibility, and security tests
  T092–T096 can be written concurrently.
- Shared UX tasks T099, T101, and T102 target separate files and can proceed in
  parallel before final route integration T103.

## Implementation Strategy

### MVP first

1. Complete Setup and Foundation.
2. Complete US1 only.
3. Stop and run the US1 independent test across Classic Solo, Quick Play,
   Practice, and Daily.
4. Release internally if its backend, accessibility, localization, performance,
   and required repository gates pass.

### Incremental delivery

1. Foundation plus US1: complete solo-family MVP.
2. Add US3: guest/account play-with-friends Party Lobby.
3. Add US2 in rollout slices: Casual Solo, Ranked Solo, Casual teams, then Ranked
   teams; keep chat images separate.
4. Complete US4 after each added route family and run full release readiness.

### Parallel team strategy

After Foundation, separate teams can own US1 games/challenges, US2
parties/matchmaking/matchplay, and US3 rooms/realtime. All teams consume the same
mode registry, schemas, gameplay primitives, error categories, and snapshot
repair policy. Coordinate edits to backend/openapi/openapi.yaml,
client/messages/en.json, client/messages/ar.json, client/app/globals.css, and
shared gameplay files serially to avoid merge conflicts.

## Completion Rules

- A task is complete only when its closest useful automated tests pass.
- Story completion requires its independent test, English and Arabic copy,
  required UI states, keyboard behavior, privacy, and performance evidence.
- PostgreSQL remains authoritative after durable creation; Redis/realtime loss
  must recover through HTTP snapshots.
- Do not enable a mode entry until its backend contract and deep-link recovery
  are deployed.
- Do not down-migrate 00019, 00020, or 00021 while referenced durable data
  exists.
- Commit only reviewed source, contracts, migrations, tests, and evidence; do
  not include secrets or generated local artifacts.
