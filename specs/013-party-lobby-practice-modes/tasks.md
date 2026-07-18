# Tasks: Party Lobby And Practice Modes

**Input**: Design documents from `specs/013-party-lobby-practice-modes/`

**Tests**: Required by the feature specification and project constitution. Behavior-protecting tests precede implementation tasks.

## Phase 1: Setup

- [x] T001 Add Party Lobby and Practice feature constants/default validation tests in `backend/internal/games/state_test.go` and `backend/internal/rooms/service_test.go`
- [x] T002 Document canonical modes, defaults, and compatibility behavior in `backend/openapi/openapi.yaml` and `specs/013-party-lobby-practice-modes/contracts/party-lobby-practice-openapi.md`
- [x] T003 [P] Add saved response fixtures for Party Lobby defaults/standings and Practice guess/history under `backend/postman/saved-responses/`

---

## Phase 2: Foundational

- [x] T004 Add failing migration tests for new mode constraints, legacy `private_room` preservation, unknown-mode rejection, and protected down behavior in `backend/internal/app/party_lobby_practice_migration_test.go`
- [x] T005 Implement additive Goose migration `backend/migrations/00020_party_lobby_practice_modes.sql`
- [x] T006 [P] Add canonical `party_lobby`, legacy alias, `practice`, open-ended, multiplayer, timer, and progression-neutral helpers in `backend/internal/games/state.go` with tests in `backend/internal/games/state_test.go`
- [x] T007 [P] Define additive mode/default/history/standing DTOs and stable errors in `backend/internal/games/dto.go`, `backend/internal/games/errors.go`, and `backend/internal/rooms/dto.go`
- [x] T008 Extend repository/service/handler interfaces for Practice advancement/history/end and Party Lobby standings in `backend/internal/games/service.go`, `backend/internal/games/handler.go`, and `backend/internal/rooms/service.go`
- [x] T009 Add bounded mode/outcome metrics and preserve redaction coverage in `backend/internal/games/metrics.go` and `backend/internal/app/casual_ranked_redaction_test.go`

**Checkpoint**: Mode vocabulary, schema constraints, DTOs, interfaces, and telemetry are ready for story implementation.

---

## Phase 3: User Story 1 - Host A Large Free-For-All Party Lobby (Priority: P1)

**Goal**: Create, join, start, play, and finish an unranked 2–50 player free-for-all with five rounds and a 180-second default timer.

**Independent Test**: Create a default room, join multiple identities, complete timed rounds, and verify durable individual standings and zero competitive progression.

- [x] T010 [P] [US1] Add room-service tests for omitted defaults, 2–50 capacity, canonical mode, guest/registered joins, and no team slots in `backend/internal/rooms/service_test.go`
- [x] T011 [P] [US1] Add multiplayer game tests for shared 180-second deadlines, early all-submit close, zero missing guesses, individual totals, and terminal free-for-all results in `backend/internal/games/party_lobby_test.go`
- [x] T012 [P] [US1] Add PostgreSQL standing-query tests for 50 players, score/distance ordering, multi-way ties, bounded queries, and deterministic reloads in `backend/internal/rooms/repository_test.go`
- [x] T013 [P] [US1] Add service/contract tests for omitted room settings, canonical mode, and standing response fields in `backend/internal/rooms/service_test.go` and `backend/openapi/openapi.yaml`
- [x] T014 [US1] Default omitted room settings and create new room games as canonical `party_lobby` in `backend/internal/rooms/service.go` and `backend/internal/rooms/repository.go`
- [x] T015 [US1] Extend multiplayer mode handling and reveal/score comments for Party Lobby compatibility in `backend/internal/games/service.go`, `backend/internal/games/repository.go`, and `backend/internal/games/speed_scoring.go`
- [x] T016 [US1] Implement bounded individual standing projection and attach it to active/completed room snapshots in `backend/internal/rooms/repository.go` and `backend/internal/rooms/service.go`
- [x] T017 [US1] Ensure Party Lobby completion bypasses competitive/matchmade completion hooks in `backend/internal/app/wiring_adapters.go` and `backend/internal/games/service.go`
- [x] T018 [US1] Update Party Lobby room schemas, defaults, examples, and legacy compatibility in `backend/openapi/openapi.yaml`

**Checkpoint**: Party Lobby is independently usable as an unranked free-for-all.

---

## Phase 4: User Story 3 - Play Unlimited Private Practice (Priority: P1)

**Goal**: Play an owner-only untimed Practice session with client-requested unlimited sequential rounds and bounded history.

**Independent Test**: Complete more than ten rounds, retry next concurrently, page history, operate without Redis, end the session, and verify no progression.

- [x] T019 [P] [US3] Add Practice creation/start/guess tests for one initial untimed round, immediate reveal, accuracy-only score, and continued active state in `backend/internal/games/repository_test.go` and `backend/internal/games/service_test.go`
- [x] T020 [P] [US3] Add PostgreSQL concurrency tests for locked next-round creation, unique sequential numbers, retries, incomplete/current/terminal rejection, and accumulated totals in `backend/internal/games/practice_repository_test.go`
- [x] T021 [P] [US3] Add cursor tests and PostgreSQL page tests for bounded history, stable ordering, and game-bound cursors in `backend/internal/games/service_test.go` and `backend/internal/games/repository_test.go`
- [x] T022 [P] [US3] Add handler/route tests for Practice create, next, history, end, CSRF, idempotency, ownership, strict JSON, and stable errors in `backend/internal/games/handler_test.go` and `backend/internal/app/routes_test.go`
- [x] T023 [US3] Extend game creation/start/submission semantics for `practice`, immediate reveal, no timer, no fixed completion, and no completion hooks in `backend/internal/games/service.go` and `backend/internal/games/repository.go`
- [x] T024 [US3] Implement transactionally locked Practice next-round persistence with idempotent replay in `backend/internal/games/repository.go`
- [x] T025 [US3] Implement owner-only Practice next-round service and response mapping in `backend/internal/games/service.go`
- [x] T026 [US3] Implement signed cursor encoding and bounded Practice history repository/service reads in `backend/internal/games/cursor.go`, `backend/internal/games/repository.go`, and `backend/internal/games/service.go`
- [x] T027 [US3] Implement idempotent owner-only Practice end behavior without history deletion in `backend/internal/games/repository.go` and `backend/internal/games/service.go`
- [x] T028 [US3] Register Practice next/history/end handlers and rate limits in `backend/internal/games/handler.go` and `backend/internal/app/routes.go`
- [x] T029 [US3] Update Practice request/response/history/end schemas and stable errors in `backend/openapi/openapi.yaml`

**Checkpoint**: Practice is independently usable without Redis, realtime, rooms, or matchmaking.

---

## Phase 5: User Story 2 - Control And Recover A Party Lobby (Priority: P2)

**Goal**: Preserve host authority, capacity correctness, refresh/reconnect, deadline progression, and retry safety under concurrency.

**Independent Test**: Race joins/settings/start against capacity and reconnect transitions, then verify one durable roster/game state and deadline progress.

- [x] T030 [P] [US2] Add concurrent join/start/settings/capacity and duplicate host-command tests in `backend/internal/rooms/repository_test.go` and `backend/internal/rooms/service_test.go`
- [x] T031 [P] [US2] Add multi-client realtime tests for Party Lobby default snapshot, roster, progress, reveal, completion, reconnect, and version recovery in `backend/internal/realtime/handler_test.go`
- [x] T032 [P] [US2] Add deadline worker tests proving Party Lobby rounds close without client polling and missing players receive zero in `backend/internal/games/worker_test.go`
- [x] T033 [US2] Generalize bounded multiplayer deadline sweep selection from Ranked-only to timed Party Lobby games without touching Casual or Practice in `backend/internal/games/repository.go` and `backend/internal/games/worker.go`
- [x] T034 [US2] Publish canonical Party Lobby lifecycle/result events and standings after durable commit in `backend/internal/rooms/service.go` and `backend/internal/app/wiring_adapters.go`
- [x] T035 [US2] Verify room join/start/reconnect paths preserve legacy `private_room` and new `party_lobby` active games in `backend/internal/rooms/repository.go` and `backend/internal/rooms/service.go`

**Checkpoint**: Party Lobby remains correct during concurrent commands, disconnects, and unattended deadlines.

---

## Phase 6: User Story 4 - Keep New Modes Separate And Safe (Priority: P2)

**Goal**: Enforce privacy and prevent Party Lobby/Practice from mutating party, queue, competitive, mission, daily, leaderboard, or realtime-only state.

**Independent Test**: Exercise unauthorized and cross-mode operations and compare all progression/state projections before and after completion.

- [x] T036 [P] [US4] Enforce privacy-safe owner checks for Practice and participant checks for Party Lobby in `backend/internal/games/service.go` and existing room authorization tests in `backend/internal/rooms/service_test.go`
- [x] T037 [P] [US4] Add cross-mode helper tests that keep Practice outside multiplayer/ranked/casual and Party Lobby outside ranked/team semantics in `backend/internal/games/state_test.go`
- [x] T038 [P] [US4] Exclude Practice and Party Lobby from public/profile statistics and history projections in `backend/internal/profiles/repository.go` and `backend/internal/users/repository.go`
- [x] T039 [US4] Harden mode gates and privacy-safe error mapping across `backend/internal/games/service.go`, `backend/internal/realtime/ticket_handler.go`, and `backend/internal/app/wiring_adapters.go`
- [x] T040 [US4] Document security boundaries, progression neutrality, and future localized client states in `backend/openapi/openapi.yaml` and `specs/013-party-lobby-practice-modes/contracts/party-lobby-practice-openapi.md`

**Checkpoint**: Both modes are isolated, private, and progression-neutral.

---

## Phase 7: Polish And Cross-Cutting Concerns

- [x] T041 [P] Add bounded 50-player Party Lobby snapshot and 100-item Practice page serialization benchmarks in `backend/internal/app/party_lobby_practice_benchmark_test.go`
- [x] T042 [P] Add Prometheus dashboard panels and alerts for Party Lobby deadline failures and Practice advancement failures in `backend/observability/grafana/dashboards/casual-ranked-backend.json` and `backend/observability/prometheus/casual-ranked-alerts.yml`
- [x] T043 Run `gofmt`, targeted tests, `go test -count=1 ./...`, `go vet ./...`, and `golangci-lint run ./...`; record exact evidence in `specs/013-party-lobby-practice-modes/quickstart.md`
- [x] T044 Run PostgreSQL/Redis integration, Linux race, and live multi-client backend validation where available; record blockers and residual risks in `specs/013-party-lobby-practice-modes/quickstart.md`
- [x] T045 Review the final diff for backend-only scope, OpenAPI consistency, migration safety, secrets, redaction, bounded queries, and task/spec traceability in `specs/013-party-lobby-practice-modes/tasks.md`

## Dependencies And Execution Order

- Phase 1 → Phase 2 blocks all stories.
- US1 and US3 are both P1 and can proceed independently after Phase 2.
- US2 depends on US1 Party Lobby behavior.
- US4 depends on US1 and US3 mode behavior.
- Polish depends on every story phase.

## Parallel Opportunities

- T003 can run independently of code setup.
- T006, T007, and T009 touch separate concerns after migration contract decisions.
- US1 service, game, repository, and handler tests can be authored in parallel.
- US3 service, repository, history, and HTTP tests can be authored in parallel.
- US2 concurrency, realtime, and deadline tests can be authored in parallel.
- US4 security, failure, and progression tests can be authored in parallel.

## Implementation Strategy

1. Establish vocabulary/schema compatibility and test contracts.
2. Ship Party Lobby core as the hosted free-for-all MVP.
3. Ship Practice independently as the second P1 slice.
4. Harden lobby concurrency/recovery.
5. Prove cross-mode isolation and progression neutrality.
6. Complete operational assets and all available backend gates.
