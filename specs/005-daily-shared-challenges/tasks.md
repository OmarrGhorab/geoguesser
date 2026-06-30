# Tasks: Daily And Shared Challenges

**Input**: Design documents from `/specs/005-daily-shared-challenges/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/challenges-openapi.md`, `quickstart.md`

**Tests**: Automated verification is required by the constitution. Backend behavior gets unit, service, handler, repository, and contract coverage. Frontend work gets lint, typecheck, build, component/flow checks where practical, plus recorded browser validation for daily/shared flows, spoiler behavior, Arabic RTL, and countdown behavior.

**Organization**: Tasks are grouped by independently testable user story, after shared setup and foundational work.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files or depends only on completed setup/foundation tasks
- **[Story]**: User story label for traceability, required only in user story phases
- Every task includes concrete file paths

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm repository guidance, create feature skeletons, and pin shared validation points before behavior work begins.

- [X] T001 Confirm Spec Kit constitution gates for this feature in `.specify/memory/constitution.md`
- [X] T002 Confirm Phase 05 plan constraints and performance budgets in `specs/005-daily-shared-challenges/plan.md`
- [X] T003 [P] Confirm local Next.js Server/Client Component guidance in `client/node_modules/next/dist/docs/01-app/01-getting-started/05-server-and-client-components.md`
- [X] T004 [P] Confirm local Next.js data fetching guidance in `client/node_modules/next/dist/docs/01-app/01-getting-started/06-fetching-data.md`
- [X] T005 [P] Confirm local Next.js route handler guidance in `client/node_modules/next/dist/docs/01-app/01-getting-started/15-route-handlers.md`
- [X] T006 [P] Confirm local Next.js caching, data security, and internationalization guidance in `client/node_modules/next/dist/docs/01-app/01-getting-started/08-caching.md`, `client/node_modules/next/dist/docs/01-app/02-guides/data-security.md`, and `client/node_modules/next/dist/docs/01-app/02-guides/internationalization.md`
- [X] T007 Create backend challenge package skeleton in `backend/internal/challenges/`
- [X] T008 [P] Create frontend challenge feature skeleton in `client/features/challenges/`
- [X] T009 [P] Create frontend challenge route skeleton in `client/app/[locale]/challenges/`
- [X] T010 [P] Add challenge API helper skeleton in `client/lib/api/challenges.ts`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared database, API, DTO, identity, and wiring foundations that all challenge stories need.

**CRITICAL**: No user story implementation should begin until this phase is complete.

- [X] T011 Create core challenge Goose migration for challenges, challenge_locations, and challenge_attempts in `backend/migrations/00006_challenges_core.sql`
- [X] T012 [P] Add challenge domain models for Challenge, ChallengeLocation, and ChallengeAttempt in `backend/internal/challenges/model.go`
- [X] T013 [P] Add challenge DTO shells and spoiler-safe response shape placeholders in `backend/internal/challenges/dto.go`
- [X] T014 [P] Add challenge domain errors and HTTP error mapping helpers in `backend/internal/challenges/errors.go`
- [X] T015 Add challenge repository interfaces and PostgreSQL implementation skeleton in `backend/internal/challenges/repository.go`
- [X] T016 Add challenge service skeleton with constructor dependencies for games, maps, locations, clock, idempotency, logger, and metrics in `backend/internal/challenges/service.go`
- [X] T017 Add challenge HTTP handler skeleton and route registration method in `backend/internal/challenges/handler.go`
- [X] T018 Wire challenge repository, service, and handler construction in `backend/cmd/api/main.go`
- [X] T019 Wire challenge handler dependency through server construction in `backend/internal/app/server.go`
- [X] T020 Register challenge routes under `/api/v1/challenges`, `/api/v1/missions`, and `/api/v1/streaks` in `backend/internal/app/routes.go`
- [X] T021 Add challenge configuration keys for reset boundary and defaults in `backend/internal/config/config.go`
- [X] T022 [P] Document challenge configuration defaults in `backend/.env.example`
- [X] T023 [P] Add frontend challenge API environment notes in `client/.env.example`
- [X] T024 Add base OpenAPI tags, shared schemas, and error components for Challenges, Missions, and Streaks in `backend/openapi/openapi.yaml`
- [X] T025 [P] Add base frontend TypeScript types for challenge summaries, attempts, leaderboards, missions, and streaks in `client/features/challenges/types.ts`
- [X] T026 [P] Add server-only challenge API helpers for metadata, attempts, results, leaderboards, missions, and streaks in `client/lib/api/challenges.ts`
- [X] T027 [P] Add baseline English challenge namespace placeholders in `client/messages/en.json`
- [X] T028 [P] Add baseline Arabic challenge namespace placeholders in `client/messages/ar.json`
- [X] T029 Add challenge observability metric names and structured log field conventions in `backend/internal/challenges/service.go`

**Checkpoint**: Foundation ready. User story work can now proceed in priority order or in parallel by story.

---

## Phase 3: User Story 1 - Play A Deterministic Daily Challenge (Priority: P1) MVP

**Goal**: A guest or signed-in player opens today's daily challenge, sees the date seed, locked rules, countdown, streak summary, and starts identical rounds to every other player for that day.

**Independent Test**: Open the daily page in two browser sessions on the same date, start both attempts, and confirm seed, settings, round count, and ordered locations match while guesses and scores remain independent.

### Tests for User Story 1

- [X] T030 [P] [US1] Add deterministic seed and reset-boundary unit tests in `backend/internal/challenges/seed_test.go`
- [X] T031 [P] [US1] Add daily materialization service tests for immutable selected locations and insufficient-location errors in `backend/internal/challenges/service_test.go`
- [X] T032 [P] [US1] Add daily attempt handler tests for GET `/challenges/daily` and POST `/challenges/daily/attempts` in `backend/internal/challenges/handler_test.go`
- [X] T033 [P] [US1] Add repository integration tests for daily challenge uniqueness and challenge location ordering in `backend/internal/challenges/repository_test.go`
- [X] T034 [P] [US1] Add frontend daily page loading, unavailable, locked settings, and countdown tests in `client/features/challenges/daily.test.tsx`

### Implementation for User Story 1

- [X] T035 [P] [US1] Implement deterministic daily seed derivation and reset-window helpers in `backend/internal/challenges/seed.go`
- [X] T036 [US1] Implement daily challenge materialization and immutable location selection in `backend/internal/challenges/service.go`
- [X] T037 [US1] Implement daily challenge persistence queries and unique daily lookup in `backend/internal/challenges/repository.go`
- [X] T038 [US1] Implement GET `/challenges/daily` and POST `/challenges/daily/attempts` handlers in `backend/internal/challenges/handler.go`
- [X] T039 [US1] Integrate challenge attempt start/resume with the existing solo game loop in `backend/internal/challenges/service.go`
- [X] T040 [US1] Add OpenAPI paths and schemas for GET `/api/v1/challenges/daily` and POST `/api/v1/challenges/daily/attempts` in `backend/openapi/openapi.yaml`
- [X] T041 [P] [US1] Implement frontend daily route page using Server Components in `client/app/[locale]/challenges/daily/page.tsx`
- [X] T042 [P] [US1] Implement daily challenge summary, locked settings, and attempt state components in `client/features/challenges/daily-panel.tsx`
- [X] T043 [P] [US1] Implement browser countdown client component from server reset timestamps in `client/features/challenges/countdown.tsx`
- [X] T044 [US1] Implement daily attempt start/resume form actions or server actions boundary in `client/features/challenges/actions.ts`
- [X] T045 [US1] Add English copy for daily seed, locked settings, countdown, loading, unavailable, disabled, and start/resume states in `client/messages/en.json`
- [X] T046 [US1] Add Arabic copy for daily seed, locked settings, countdown, loading, unavailable, disabled, and start/resume states in `client/messages/ar.json`
- [X] T047 [US1] Add structured logs and metrics for daily materialization and daily attempt start in `backend/internal/challenges/service.go`

**Checkpoint**: User Story 1 works independently as the MVP.

---

## Phase 4: User Story 2 - Share A Stable Challenge Link (Priority: P1)

**Goal**: A player creates or opens a shared challenge link that always resolves to the same seed, map pool, settings, and ordered rounds.

**Independent Test**: Create one shared challenge link, open it in two browser sessions, and confirm both sessions load identical metadata and rounds with separate attempts.

### Tests for User Story 2

- [X] T048 [P] [US2] Add shared challenge creation idempotency and stable seed service tests in `backend/internal/challenges/service_test.go`
- [X] T049 [P] [US2] Add shared challenge repository tests for unique code lookup and immutable selected locations in `backend/internal/challenges/repository_test.go`
- [X] T050 [P] [US2] Add handler tests for POST `/challenges/shared`, GET `/challenges/shared/{code}`, and POST `/challenges/{challengeId}/attempts` in `backend/internal/challenges/handler_test.go`
- [X] T051 [P] [US2] Add frontend shared challenge create/open/invalid-link tests in `client/features/challenges/shared.test.tsx`

### Implementation for User Story 2

- [X] T052 [US2] Extend shared challenge fields, indexes, and idempotency metadata in `backend/migrations/00006_challenges_core.sql`
- [X] T053 [US2] Implement shared challenge code generation and stable seed creation in `backend/internal/challenges/seed.go`
- [X] T054 [US2] Implement shared challenge creation, validation, and immutable location snapshot logic in `backend/internal/challenges/service.go`
- [X] T055 [US2] Implement shared challenge lookup and owner/session attempt queries in `backend/internal/challenges/repository.go`
- [X] T056 [US2] Implement POST `/challenges/shared`, GET `/challenges/shared/{code}`, and POST `/challenges/{challengeId}/attempts` handlers in `backend/internal/challenges/handler.go`
- [X] T057 [US2] Add Redis-backed idempotency handling for shared challenge creation and attempt start in `backend/internal/challenges/service.go`
- [X] T058 [US2] Add OpenAPI paths and examples for shared challenge create, lookup, and attempt start in `backend/openapi/openapi.yaml`
- [X] T059 [P] [US2] Implement shared challenge route page in `client/app/[locale]/challenges/[challengeId]/page.tsx`
- [X] T060 [P] [US2] Implement shared challenge create form and share link controls in `client/features/challenges/shared-create.tsx`
- [X] T061 [P] [US2] Implement shared challenge metadata and invalid/unavailable states in `client/features/challenges/shared-panel.tsx`
- [X] T062 [US2] Add English copy for shared link creation, copied, invalid link, unavailable, and start states in `client/messages/en.json`
- [X] T063 [US2] Add Arabic copy for shared link creation, copied, invalid link, unavailable, and start states in `client/messages/ar.json`
- [X] T064 [US2] Add structured logs and metrics for shared creation, shared lookup, and shared attempt start in `backend/internal/challenges/service.go`

**Checkpoint**: User Stories 1 and 2 both work independently.

---

## Phase 5: User Story 3 - Compare Challenge Results And Leaderboards (Priority: P2)

**Goal**: A player finishes a daily or shared challenge and sees a stable result summary, rank context, leaderboard, and spoiler-safe comparison view.

**Independent Test**: Complete the same challenge as multiple players, confirm deterministic ordering and tie-breakers, and confirm unfinished players cannot receive answer spoilers.

### Tests for User Story 3

- [X] T065 [P] [US3] Add leaderboard ordering and tie-breaker unit tests in `backend/internal/challenges/leaderboard_test.go`
- [X] T066 [P] [US3] Add spoiler-safe result DTO service tests for completed and unfinished players in `backend/internal/challenges/service_test.go`
- [X] T067 [P] [US3] Add result and leaderboard handler tests for GET `/challenges/{challengeId}/results` and GET `/challenges/{challengeId}/leaderboard` in `backend/internal/challenges/handler_test.go`
- [X] T068 [P] [US3] Add repository integration tests for result snapshots and paginated leaderboard reads in `backend/internal/challenges/repository_test.go`
- [X] T069 [P] [US3] Add frontend result, leaderboard, empty, hidden-spoiler, and pagination tests in `client/features/challenges/results.test.tsx`

### Implementation for User Story 3

- [X] T070 [US3] Create result and leaderboard Goose migration in `backend/migrations/00007_challenge_results_leaderboards.sql`
- [X] T071 [US3] Add ChallengeResult and LeaderboardEntry models in `backend/internal/challenges/model.go`
- [X] T072 [US3] Implement result snapshot creation and leaderboard eligibility rules in `backend/internal/challenges/service.go`
- [X] T073 [US3] Implement deterministic leaderboard ranking, tie-breakers, and bounded pagination queries in `backend/internal/challenges/repository.go`
- [X] T074 [US3] Implement spoiler-safe result and leaderboard DTO mapping in `backend/internal/challenges/dto.go`
- [X] T075 [US3] Implement GET `/challenges/{challengeId}/results` and GET `/challenges/{challengeId}/leaderboard` handlers in `backend/internal/challenges/handler.go`
- [X] T076 [US3] Hook challenge result finalization to existing game completion/results flow in `backend/internal/challenges/service.go`
- [X] T077 [US3] Add OpenAPI schemas and examples for results, rank context, pagination, and spoiler-protected responses in `backend/openapi/openapi.yaml`
- [X] T078 [P] [US3] Implement result summary component with per-round score and distance display in `client/features/challenges/result-summary.tsx`
- [X] T079 [P] [US3] Implement leaderboard component with empty, loading, hidden, pagination, and current-player states in `client/features/challenges/leaderboard.tsx`
- [X] T080 [US3] Integrate result and leaderboard surfaces into daily and shared routes in `client/app/[locale]/challenges/daily/page.tsx`
- [X] T081 [US3] Integrate result and leaderboard surfaces into shared route in `client/app/[locale]/challenges/[challengeId]/page.tsx`
- [X] T082 [US3] Add English copy for results, ranks, participant counts, hidden spoilers, and leaderboard states in `client/messages/en.json`
- [X] T083 [US3] Add Arabic copy for results, ranks, participant counts, hidden spoilers, and leaderboard states in `client/messages/ar.json`
- [X] T084 [US3] Add structured logs and metrics for result finalization, leaderboard reads, and spoiler-guard rejections in `backend/internal/challenges/service.go`

**Checkpoint**: Comparison, results, and daily leaderboard work without leaking spoilers.

---

## Phase 6: User Story 4 - Maintain Streaks And Complete Missions (Priority: P2)

**Goal**: Daily challenge completions update streaks, challenge actions update mission progress, and players can view clear progress, reward/status, and guest persistence limits.

**Independent Test**: Complete daily challenges across simulated dates and shared challenge actions, then confirm streak mutations, protection behavior, mission progress, completion, and guest limits.

### Tests for User Story 4

- [X] T085 [P] [US4] Add streak date-bound transition tests for start, increment, break, and protection state in `backend/internal/challenges/streaks_test.go`
- [X] T086 [P] [US4] Add mission progress tests for daily completion, shared participation, score threshold, leaderboard milestone, streak milestone, and round accuracy events in `backend/internal/challenges/missions_test.go`
- [X] T087 [P] [US4] Add handler tests for GET `/streaks/daily`, GET `/missions`, and POST `/missions/{missionId}/claim` in `backend/internal/challenges/handler_test.go`
- [X] T088 [P] [US4] Add repository integration tests for idempotent streak events and mission progress events in `backend/internal/challenges/repository_test.go`
- [X] T089 [P] [US4] Add frontend streak and mission state tests for guest/account, empty, completed, claimable, and expired states in `client/features/challenges/missions.test.tsx`

### Implementation for User Story 4

- [X] T090 [US4] Create streak and mission Goose migration in `backend/migrations/00008_challenge_streaks_missions.sql`
- [X] T091 [US4] Add Streak, StreakEvent, Mission, MissionProgress, and MissionProgressEvent models in `backend/internal/challenges/model.go`
- [X] T092 [US4] Implement daily streak transition and protection-state logic in `backend/internal/challenges/streaks.go`
- [X] T093 [US4] Implement mission definitions, event application, progress completion, and claim behavior in `backend/internal/challenges/missions.go`
- [X] T094 [US4] Implement streak and mission repository queries with idempotency constraints in `backend/internal/challenges/repository.go`
- [X] T095 [US4] Hook streak and mission updates into challenge result finalization in `backend/internal/challenges/service.go`
- [X] T096 [US4] Implement GET `/streaks/daily`, GET `/missions`, and POST `/missions/{missionId}/claim` handlers in `backend/internal/challenges/handler.go`
- [X] T097 [US4] Add OpenAPI schemas and examples for streaks, missions, progress, guest limits, and claim errors in `backend/openapi/openapi.yaml`
- [X] T098 [P] [US4] Implement streak summary component with protection and guest limit states in `client/features/challenges/streak-summary.tsx`
- [X] T099 [P] [US4] Implement missions panel with active, empty, progress, completed, claimable, claimed, and expired states in `client/features/challenges/missions-panel.tsx`
- [X] T100 [US4] Integrate streak and mission surfaces into daily and shared challenge pages in `client/app/[locale]/challenges/daily/page.tsx`
- [X] T101 [US4] Integrate streak and mission surfaces into shared challenge page in `client/app/[locale]/challenges/[challengeId]/page.tsx`
- [X] T102 [US4] Add English copy for streak status, protection, missions, rewards/status, guest limits, empty, completed, and claimed states in `client/messages/en.json`
- [X] T103 [US4] Add Arabic copy for streak status, protection, missions, rewards/status, guest limits, empty, completed, and claimed states in `client/messages/ar.json`
- [X] T104 [US4] Add structured logs and metrics for streak updates, mission progress application, and mission claim outcomes in `backend/internal/challenges/service.go`

**Checkpoint**: Streaks and missions are durable for accounts, session-safe for guests, and visible after qualifying challenge actions.

---

## Phase 7: User Story 5 - Resume Challenge State Across Reloads (Priority: P3)

**Goal**: A player can reload or revisit a daily/shared challenge and receive the correct pending, active, completed, expired, or unavailable state without changing rules or losing progress.

**Independent Test**: Start a challenge, complete at least one round, reload the route, and confirm the same challenge identity, locked settings, current round, mission progress, and streak context are restored.

### Tests for User Story 5

- [X] T105 [P] [US5] Add attempt resume and completed-result replay service tests in `backend/internal/challenges/service_test.go`
- [X] T106 [P] [US5] Add handler tests for active, completed, expired, and unavailable challenge reload states in `backend/internal/challenges/handler_test.go`
- [X] T107 [P] [US5] Add frontend reload-state tests for pending, active, completed, unavailable, and history access states in `client/features/challenges/resume.test.tsx`

### Implementation for User Story 5

- [X] T108 [US5] Implement attempt state restoration, completed result replay, and duplicate daily leaderboard-credit safeguards in `backend/internal/challenges/service.go`
- [X] T109 [US5] Implement repository queries for current actor attempt, completed attempt, and challenge history lookup in `backend/internal/challenges/repository.go`
- [X] T110 [US5] Implement challenge history or completed access endpoint behavior in `backend/internal/challenges/handler.go`
- [X] T111 [US5] Add OpenAPI examples for resume, already-completed, expired, unavailable, and history/result access states in `backend/openapi/openapi.yaml`
- [X] T112 [P] [US5] Implement reload-aware state coordinator component for daily/shared pages in `client/features/challenges/challenge-state.tsx`
- [X] T113 [US5] Integrate pending, active, completed, unavailable, and history states into daily route in `client/app/[locale]/challenges/daily/page.tsx`
- [X] T114 [US5] Integrate pending, active, completed, unavailable, and history states into shared route in `client/app/[locale]/challenges/[challengeId]/page.tsx`
- [X] T115 [US5] Add English copy for resume, already completed, history, expired, and unavailable states in `client/messages/en.json`
- [X] T116 [US5] Add Arabic copy for resume, already completed, history, expired, and unavailable states in `client/messages/ar.json`
- [X] T117 [US5] Add structured logs and metrics for attempt resume and completed-result replay paths in `backend/internal/challenges/service.go`

**Checkpoint**: Challenge reload and revisit flows preserve the correct state.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Validate contracts, performance, accessibility, localization, observability, and release readiness across all stories.

- [X] T118 [P] Update Phase 05 quickstart evidence placeholders for completed validation in `specs/005-daily-shared-challenges/quickstart.md`
- [X] T119 [P] Update Phase 05 contract notes to match final OpenAPI details in `specs/005-daily-shared-challenges/contracts/challenges-openapi.md`
- [X] T120 Run backend unit and integration tests with `go test ./...` and record evidence in `specs/005-daily-shared-challenges/quickstart.md`
- [X] T121 Run targeted backend challenge and game tests with `go test ./internal/challenges/... ./internal/games/...` and record evidence in `specs/005-daily-shared-challenges/quickstart.md`
- [X] T122 Run OpenAPI lint with `npx pnpm@10.24.0 check:openapi` and record evidence in `specs/005-daily-shared-challenges/quickstart.md`
- [X] T123 Run frontend lint with `npx pnpm@10.24.0 --dir client lint` and record evidence in `specs/005-daily-shared-challenges/quickstart.md`
- [X] T124 Run frontend typecheck with `npx pnpm@10.24.0 --dir client typecheck` and record evidence in `specs/005-daily-shared-challenges/quickstart.md`
- [X] T125 Run frontend production build with `npx pnpm@10.24.0 --dir client build` and record evidence in `specs/005-daily-shared-challenges/quickstart.md`
- [X] T126 Run manual two-browser daily challenge determinism validation and record evidence in `specs/005-daily-shared-challenges/quickstart.md`
- [X] T127 Run manual shared challenge link stability validation and record evidence in `specs/005-daily-shared-challenges/quickstart.md`
- [X] T128 Run manual leaderboard spoiler-protection validation and record evidence in `specs/005-daily-shared-challenges/quickstart.md`
- [X] T129 Run manual streak and mission validation with simulated dates or fixtures and record evidence in `specs/005-daily-shared-challenges/quickstart.md`
- [X] T130 Run English and Arabic RTL browser validation for daily, shared, result, leaderboard, mission, streak, empty, error, disabled, and success states in `specs/005-daily-shared-challenges/quickstart.md`
- [X] T131 Verify backend logs and metrics redact hidden coordinates, tokens, and private profile details in `backend/internal/challenges/service.go`
- [X] T132 Verify all challenge UI controls have accessible names, keyboard focus, disabled states, and non-color-only indicators in `client/features/challenges/`
- [X] T133 Verify challenge metadata and attempt start meet p95 performance budgets from the plan using evidence in `specs/005-daily-shared-challenges/quickstart.md`
- [X] T134 Verify migration rollback/readiness notes for challenge tables and indexes in `backend/migrations/00006_challenges_core.sql`
- [X] T135 Verify migration rollback/readiness notes for result and leaderboard tables in `backend/migrations/00007_challenge_results_leaderboards.sql`
- [X] T136 Verify migration rollback/readiness notes for streak and mission tables in `backend/migrations/00008_challenge_streaks_missions.sql`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup and blocks every user story.
- **US1 Daily Challenge (Phase 3)**: Depends on Foundational; this is the MVP.
- **US2 Shared Challenge (Phase 4)**: Depends on Foundational and reuses attempt-start foundations from US1 where helpful.
- **US3 Results And Leaderboards (Phase 5)**: Depends on challenge attempts from US1/US2 and result completion from the game loop.
- **US4 Streaks And Missions (Phase 6)**: Depends on challenge completion/result events from US3.
- **US5 Resume State (Phase 7)**: Depends on challenge attempts from US1/US2 and completed results from US3.
- **Polish (Phase 8)**: Depends on the implemented stories selected for release.

### User Story Dependencies

- **US1 (P1)**: Can start after Foundational and has no dependency on other stories.
- **US2 (P1)**: Can start after Foundational; link creation is independent, but shared attempt start should reuse the common attempt path.
- **US3 (P2)**: Requires completed attempts from US1/US2 for meaningful validation.
- **US4 (P2)**: Requires completion/result events from US3 to update streaks and missions reliably.
- **US5 (P3)**: Requires attempt lifecycle and result replay behavior from US1/US2/US3.

### Within Each User Story

- Write tests first and confirm they fail when the behavior is missing.
- Complete models/migrations before repository queries.
- Complete repository/service logic before handlers.
- Complete backend contracts before frontend integration.
- Complete copy and UI states before story checkpoint validation.

---

## Parallel Opportunities

- T003-T006, T008-T010 can run in parallel after T001-T002.
- T012-T014, T022-T028 can run in parallel after T011 and T021 are understood.
- US1 test tasks T030-T034 can run in parallel.
- US2 test tasks T048-T051 can run in parallel.
- US3 test tasks T065-T069 can run in parallel.
- US4 test tasks T085-T089 can run in parallel.
- US5 test tasks T105-T107 can run in parallel.
- Frontend component tasks marked [P] can run alongside backend implementation once server DTOs are stable.
- Documentation and contract polish tasks T118-T119 can run in parallel with final verification.

## Parallel Example: User Story 1

```bash
# Backend tests and frontend UI tests can be prepared together:
Task: "T030 [P] [US1] Add deterministic seed and reset-boundary unit tests in backend/internal/challenges/seed_test.go"
Task: "T032 [P] [US1] Add daily attempt handler tests for GET /challenges/daily and POST /challenges/daily/attempts in backend/internal/challenges/handler_test.go"
Task: "T034 [P] [US1] Add frontend daily page loading, unavailable, locked settings, and countdown tests in client/features/challenges/daily.test.tsx"
```

## Parallel Example: User Story 2

```bash
Task: "T048 [P] [US2] Add shared challenge creation idempotency and stable seed service tests in backend/internal/challenges/service_test.go"
Task: "T049 [P] [US2] Add shared challenge repository tests for unique code lookup and immutable selected locations in backend/internal/challenges/repository_test.go"
Task: "T051 [P] [US2] Add frontend shared challenge create/open/invalid-link tests in client/features/challenges/shared.test.tsx"
```

## Parallel Example: User Story 3

```bash
Task: "T065 [P] [US3] Add leaderboard ordering and tie-breaker unit tests in backend/internal/challenges/leaderboard_test.go"
Task: "T066 [P] [US3] Add spoiler-safe result DTO service tests for completed and unfinished players in backend/internal/challenges/service_test.go"
Task: "T069 [P] [US3] Add frontend result, leaderboard, empty, hidden-spoiler, and pagination tests in client/features/challenges/results.test.tsx"
```

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1 setup.
2. Complete Phase 2 foundation.
3. Complete Phase 3 daily challenge behavior.
4. Validate the two-browser daily challenge determinism scenario.
5. Stop and review before adding shared links, comparison, streaks, missions, or resume polish.

### Incremental Delivery

1. Deliver US1 daily challenge MVP.
2. Add US2 shared stable links.
3. Add US3 results and leaderboards with spoiler safety.
4. Add US4 streaks and missions.
5. Add US5 reload/revisit state hardening.
6. Run Phase 8 verification before release.

### Notes

- Do not use GORM AutoMigrate; all data changes belong in Goose migrations under `backend/migrations/`.
- Keep canonical server state out of Zustand; frontend challenge data should flow through server-only helpers in `client/lib/api/challenges.ts`.
- Keep hidden coordinates and answer-revealing details out of unfinished-player DTOs and logs.
- Use Redis for short-lived idempotency/rate-limit/read-cache behavior only; durable challenge, result, leaderboard, streak, and mission facts belong in PostgreSQL.
- Preserve English and Arabic copy parity for every user-facing state.
