# Tasks: Solo Game Loop

**Input**: Design documents from `/specs/004-solo-game-loop/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/games-openapi.md](./contracts/games-openapi.md), [quickstart.md](./quickstart.md)

**Tests**: Required by the GeoGuess constitution and this phase. Write behavior tests before implementation where practical, especially scoring, state transitions, handlers, repository persistence, and the complete solo game loop.

**Organization**: Tasks are grouped by user story so each story can be implemented and tested independently after the shared foundation.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish the package skeleton, schema deltas, and route/wiring touchpoints for the backend solo game module.

- [X] T001 Create the `backend/internal/games` package skeleton with `dto.go`, `errors.go`, `handler.go`, `model.go`, `repository.go`, `scoring.go`, `service.go`, and `state.go`
- [X] T002 [P] Add `backend/internal/games/scoring_test.go` with failing table-driven tests for haversine known distances, 25m full-score threshold, exponential decay, and 0..5000 score bounds
- [X] T003 [P] Add `backend/internal/games/service_test.go` with failing tests for create/start/complete solo game state transitions using fake repositories, fake map selector, and `clock.Fixed`
- [X] T004 [P] Add `backend/internal/games/handler_test.go` with failing handler tests for create game, start game, current round, submit guess, results, auth rejection, and hidden coordinate response shape
- [X] T005 [P] Add `backend/internal/games/repository_test.go` with integration test scaffolding that uses `DATABASE_URL`, skips when unset, and seeds maps/locations/games in isolated UUID fixtures
- [X] T006 Add Goose migration `backend/migrations/00005_game_idempotency_indexes.sql` for partial unique `guesses(game_player_id, idempotency_key)` where `idempotency_key IS NOT NULL` and an index supporting current-round lookup
- [X] T007 [P] Add solo game error code expectations to `specs/004-solo-game-loop/contracts/games-openapi.md` if implementation chooses exact code names differing from the draft

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core domain models, DTOs, repository contracts, and route wiring needed by all user stories.

**Critical**: No user story implementation should proceed until this phase is complete.

- [X] T008 Define game, round, game player, guess, revealed location, and result models in `backend/internal/games/model.go`
- [X] T009 Define request/response DTOs matching `backend/openapi/openapi.yaml` in `backend/internal/games/dto.go`
- [X] T010 Define domain errors for validation, ownership, transition, timeout, duplicate guess, idempotency conflict, hidden/missing game, and insufficient locations in `backend/internal/games/errors.go`
- [X] T011 Define game and round status constants plus transition helpers in `backend/internal/games/state.go`
- [X] T012 Implement haversine distance and version 1 score calculation in `backend/internal/games/scoring.go`
- [X] T013 Define repository interfaces and constructor dependencies for transactions, map selection, clock, Redis/idempotency, and logging in `backend/internal/games/service.go`
- [X] T014 Implement PostgreSQL repository query method signatures and transaction helpers in `backend/internal/games/repository.go`
- [X] T015 Update `backend/cmd/api/main.go` to construct `games.Repository`, `games.Service`, and `games.Handler` with existing db, Redis client, map service, logger, and clock dependencies
- [X] T016 Update `backend/internal/app/routes.go` and `backend/internal/app/server.go` to accept and register `games.Handler` routes under `/api/v1/games`
- [X] T017 Update `backend/internal/app/routes_test.go` to verify the solo game routes are mounted
- [X] T018 Update `backend/openapi/openapi.yaml` to reflect Phase 04 solo-only behavior, idempotency expectations, game errors, and hidden-coordinate guarantees

**Checkpoint**: Foundation ready. User story implementation can now begin.

---

## Phase 3: User Story 1 - Complete a Solo Game (Priority: P1) MVP

**Goal**: A guest or registered player can create and start a solo game, play every round, submit valid guesses, and retrieve durable final results.

**Independent Test**: A guest or registered player creates a game, starts it, reads current rounds, submits one valid guess per round, and reads final results with total distance and score.

### Tests for User Story 1

- [X] T019 [P] [US1] Add service tests for create solo game selecting fixed rounds, creating one player, starting round 1, accepting valid guesses, advancing rounds, and completing final game in `backend/internal/games/service_test.go`
- [X] T020 [P] [US1] Add scoring unit tests for total score accumulation and persisted scoring version assumptions in `backend/internal/games/scoring_test.go`
- [X] T021 [P] [US1] Add handler tests for successful `POST /games`, `POST /games/{gameId}/start`, `GET /games/{gameId}/rounds/current`, `POST /games/{gameId}/rounds/{roundId}/guesses`, and `GET /games/{gameId}/results` in `backend/internal/games/handler_test.go`
- [X] T022 [P] [US1] Add repository integration tests for creating a solo game transaction, inserting rounds, loading current round, saving guesses, updating totals, and loading final results in `backend/internal/games/repository_test.go`

### Implementation for User Story 1

- [X] T023 [US1] Implement `CreateGame` service flow in `backend/internal/games/service.go` with session owner resolution, map selection, one player row, all round rows, and insufficient-location handling
- [X] T024 [US1] Implement `StartGame` service flow in `backend/internal/games/service.go` with pending-to-active transition, server timestamps, first round activation, and timer deadline calculation
- [X] T025 [US1] Implement `GetCurrentRound` service flow in `backend/internal/games/service.go` returning playable round DTOs without hidden coordinates
- [X] T026 [US1] Implement `SubmitGuess` service flow in `backend/internal/games/service.go` with coordinate validation, distance/score calculation, round completion, total updates, next-round activation, and game completion
- [X] T027 [US1] Implement `GetResults` service flow in `backend/internal/games/service.go` returning durable per-round and total results
- [X] T028 [US1] Implement repository methods for game creation, owner lookups, current round loading, guess insertion, total updates, and result reads in `backend/internal/games/repository.go`
- [X] T029 [US1] Implement HTTP handlers and route registration for all solo game endpoints in `backend/internal/games/handler.go`
- [X] T030 [US1] Map US1 domain errors to shared API error envelopes in `backend/internal/games/handler.go`
- [X] T031 [US1] Run `go test ./internal/games/...` from `backend/` and fix US1 failures

**Checkpoint**: User Story 1 is independently functional and demoable as the MVP backend solo loop.

---

## Phase 4: User Story 2 - Enforce Fair Round Rules (Priority: P1)

**Goal**: The backend controls round selection, timing, one-guess behavior, ownership, and answer reveal so the client cannot gain an unfair advantage.

**Independent Test**: The player cannot see coordinates before reveal, cannot receive repeated locations in a game, cannot submit late guesses, cannot submit for another owner, and cannot change a completed round score.

### Tests for User Story 2

- [X] T032 [P] [US2] Add service tests for no repeated locations, not-enough-locations failure, owner mismatch rejection, and non-current round rejection in `backend/internal/games/service_test.go`
- [X] T033 [P] [US2] Add service tests for late guess rejection using `clock.Fixed`, deadline equality boundaries, and expired round reveal behavior in `backend/internal/games/service_test.go`
- [X] T034 [P] [US2] Add handler tests asserting current-round responses omit `location_id`, actual latitude, actual longitude, and provider refs before reveal in `backend/internal/games/handler_test.go`
- [X] T035 [P] [US2] Add repository integration tests for database one-guess uniqueness and non-repeated location fixtures in `backend/internal/games/repository_test.go`

### Implementation for User Story 2

- [X] T036 [US2] Enforce no-repeat selected locations and insufficient-location errors in `backend/internal/games/service.go`
- [X] T037 [US2] Enforce owner-only authorization for game state, current round, guess submission, and results in `backend/internal/games/service.go`
- [X] T038 [US2] Enforce server-owned deadline checks and late-guess rejection in `backend/internal/games/service.go`
- [X] T039 [US2] Enforce one accepted guess per round with conflict-safe handling in `backend/internal/games/service.go` and `backend/internal/games/repository.go`
- [X] T040 [US2] Ensure current-round DTO shaping excludes hidden coordinates and sensitive provider fields in `backend/internal/games/dto.go`
- [X] T041 [US2] Add rate limiting for game creation and guess submission routes using existing middleware in `backend/internal/app/routes.go`
- [X] T042 [US2] Apply and verify `backend/migrations/00005_game_idempotency_indexes.sql` against local PostgreSQL
- [X] T043 [US2] Run `go test ./internal/games/...` from `backend/` and fix US2 failures

**Checkpoint**: User Story 2 fairness guarantees are independently testable.

---

## Phase 5: User Story 3 - Retry Guess Submission Safely (Priority: P2)

**Goal**: Repeating the same guess request does not duplicate guesses or alter scoring, while conflicting retries are rejected.

**Independent Test**: Repeating a guess with the same retry identity returns the original result, and using the same retry identity with different coordinates returns a conflict without changing persisted data.

### Tests for User Story 3

- [X] T044 [P] [US3] Add service tests for same-key same-body idempotent replay and same-key different-body conflict in `backend/internal/games/service_test.go`
- [X] T045 [P] [US3] Add handler tests for `Idempotency-Key` replay and conflict responses on guess submission in `backend/internal/games/handler_test.go`
- [X] T046 [P] [US3] Add repository integration tests for partial unique idempotency key behavior in `backend/internal/games/repository_test.go`

### Implementation for User Story 3

- [X] T047 [US3] Implement idempotency key parsing and validation for create, start, and guess endpoints in `backend/internal/games/handler.go`
- [X] T048 [US3] Implement Redis-backed short-lived idempotency store abstraction in `backend/internal/games/service.go`
- [X] T049 [US3] Implement persisted guess lookup/replay by player and idempotency key in `backend/internal/games/repository.go`
- [X] T050 [US3] Enforce same-key different-body conflicts without mutating original guess or score in `backend/internal/games/service.go`
- [X] T051 [US3] Update `backend/openapi/openapi.yaml` examples/descriptions for idempotent replay and idempotency conflict
- [X] T052 [US3] Run `go test ./internal/games/...` from `backend/` and fix US3 failures

**Checkpoint**: User Story 3 retry behavior is independently functional.

---

## Phase 6: User Story 4 - Resume and Inspect Game State (Priority: P2)

**Goal**: A player can reload an in-progress or completed solo game and get authoritative current state, completed round results, and final totals.

**Independent Test**: A player starts a game, submits at least one guess, reloads game state, continues from the correct current round, completes the game, and reloads identical final results.

### Tests for User Story 4

- [X] T053 [P] [US4] Add service tests for reloading in-progress game state after one completed round in `backend/internal/games/service_test.go`
- [X] T054 [P] [US4] Add repository integration tests for batched final result reads without N+1 queries in `backend/internal/games/repository_test.go`
- [X] T055 [P] [US4] Add handler tests for `GET /games/{gameId}` and `GET /games/{gameId}/results` reload behavior in `backend/internal/games/handler_test.go`

### Implementation for User Story 4

- [X] T056 [US4] Implement `GetGame` service response for pending, active, and completed solo games in `backend/internal/games/service.go`
- [X] T057 [US4] Implement reload-safe current round resolution for active games with completed prior rounds in `backend/internal/games/service.go`
- [X] T058 [US4] Implement batched result loading for game, player, rounds, locations, and guesses in `backend/internal/games/repository.go`
- [X] T059 [US4] Implement `GET /games/{gameId}` handler response mapping in `backend/internal/games/handler.go`
- [X] T060 [US4] Ensure completed results are stable across repeated reads in `backend/internal/games/service.go`
- [X] T061 [US4] Run `go test ./internal/games/...` from `backend/` and fix US4 failures

**Checkpoint**: User Story 4 reload behavior is independently functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Verification, docs, operational checks, and final readiness across all stories.

- [X] T062 [P] Update `specs/004-solo-game-loop/quickstart.md` with any final command or API payload changes discovered during implementation
- [X] T063 [P] Update `backend/openapi/openapi.yaml` to keep final schemas, errors, examples, and idempotency behavior aligned with implementation
- [X] T064 [P] Add structured logs for create game, start game, guess accepted/rejected, round completed, and game completed without logging hidden coordinates in `backend/internal/games/service.go`
- [X] T065 Add Prometheus metric observations for guess submission latency and game completion path using existing observability patterns in `backend/internal/games/service.go` or route middleware
- [X] T066 Run `gofmt` on `backend/internal/games`, `backend/internal/app`, and `backend/cmd/api`
- [X] T067 Run `go test ./...` from `backend/`
- [X] T068 Run `pnpm check:openapi` from repository root
- [X] T069 Run `pnpm check` from repository root and record any non-feature failures
- [ ] T070 Execute the manual scenario in `specs/004-solo-game-loop/quickstart.md` against local Docker PostgreSQL and Redis

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 Setup**: No dependencies.
- **Phase 2 Foundational**: Depends on Phase 1 and blocks all user stories.
- **Phase 3 US1**: Depends on Phase 2 and is the MVP implementation.
- **Phase 4 US2**: Depends on Phase 2 and should be completed before merge because fairness is P1.
- **Phase 5 US3**: Depends on Phase 2 and integrates with guess submission from US1.
- **Phase 6 US4**: Depends on Phase 2 and integrates with game state/results from US1.
- **Phase 7 Polish**: Depends on all desired user stories.

### User Story Dependencies

- **US1 Complete a Solo Game**: Starts after foundation; no dependency on other stories.
- **US2 Enforce Fair Round Rules**: Starts after foundation; can be developed alongside US1 service tests, but final validation requires US1 guess/current-round flows.
- **US3 Retry Guess Submission Safely**: Starts after foundation; final behavior depends on US1 guess submission.
- **US4 Resume and Inspect Game State**: Starts after foundation; final behavior depends on US1 game state and results.

### Parallel Opportunities

- Setup tests T002, T003, T004, T005 can run in parallel.
- Foundation DTO/model/error/state/scoring tasks T008 through T012 can run in parallel once package skeleton exists.
- Within each story, tests marked `[P]` can be written in parallel before implementation.
- US2, US3, and US4 test design can proceed after Phase 2 while US1 implementation is underway, but shared files require coordination.

---

## Parallel Example: User Story 1

```text
Task: "T019 [US1] Add service tests for create/start/complete solo game in backend/internal/games/service_test.go"
Task: "T021 [US1] Add handler tests for game endpoints in backend/internal/games/handler_test.go"
Task: "T022 [US1] Add repository integration tests in backend/internal/games/repository_test.go"
```

## Parallel Example: User Story 2

```text
Task: "T032 [US2] Add no-repeat and owner mismatch service tests in backend/internal/games/service_test.go"
Task: "T034 [US2] Add hidden coordinate handler tests in backend/internal/games/handler_test.go"
Task: "T035 [US2] Add repository uniqueness tests in backend/internal/games/repository_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete Phase 3 (US1).
3. Run `go test ./internal/games/...`.
4. Demo a full solo game through backend API calls.

### Required Before Merge

US1 and US2 should be treated as the minimum mergeable backend feature because gameplay without fairness guarantees violates the phase goal. US3 and US4 are P2 but strongly recommended before PR because they protect timed gameplay retries and reloadability.

### Verification Gates

1. `go test ./internal/games/...`
2. `go test ./...`
3. `pnpm check:openapi`
4. `pnpm check`
5. Manual quickstart scenario against Docker PostgreSQL and Redis

## Notes

- `[P]` means different files or safely parallel work with minimal dependency coupling.
- `[US1]`, `[US2]`, `[US3]`, and `[US4]` map directly to the user stories in `spec.md`.
- Keep exact answer coordinates out of current-round DTOs, logs, and errors before reveal.
- Do not add frontend implementation tasks in this backend phase.
- Do not use GORM AutoMigrate; all schema changes must be Goose migrations.
