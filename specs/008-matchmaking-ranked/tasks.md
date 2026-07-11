# Tasks: Matchmaking And Ranked Foundations

**Input**: Design documents from `/specs/008-matchmaking-ranked/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/matchmaking-openapi.md](./contracts/matchmaking-openapi.md), [quickstart.md](./quickstart.md)

**Tests**: Automated verification is mandatory under the project constitution. Behavior tests are scheduled before their implementation, with Redis/PostgreSQL integration coverage for concurrency and recorded browser evidence for multi-session, localization, RTL, and accessibility flows.

**Organization**: Tasks are grouped by user story so each increment can be implemented and validated independently after the shared foundation is complete.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it uses different files and has no dependency on another incomplete task in the same group.
- **[Story]**: Maps the task to a user story in [spec.md](./spec.md).
- Every task names the exact file or directory it changes or validates.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish feature boundaries, verify framework guidance, and capture a clean baseline before behavior changes.

- [X] T001 Confirm implementation-sensitive Next.js guidance in `client/node_modules/next/dist/docs/01-app/01-getting-started/05-server-and-client-components.md`, `client/node_modules/next/dist/docs/01-app/01-getting-started/06-fetching-data.md`, `client/node_modules/next/dist/docs/01-app/02-guides/data-security.md`, `client/node_modules/next/dist/docs/01-app/02-guides/forms.md`, and `client/node_modules/next/dist/docs/01-app/02-guides/internationalization.md`
- [X] T002 Create the matchmaking backend skeleton in `backend/internal/matchmaking/model.go`, `backend/internal/matchmaking/dto.go`, `backend/internal/matchmaking/errors.go`, `backend/internal/matchmaking/repository.go`, `backend/internal/matchmaking/service.go`, `backend/internal/matchmaking/handler.go`, and `backend/internal/matchmaking/metrics.go`
- [X] T003 [P] Create backend matchmaking test skeletons in `backend/internal/matchmaking/service_test.go`, `backend/internal/matchmaking/handler_test.go`, `backend/internal/matchmaking/repository_test.go`, and `backend/internal/matchmaking/metrics_test.go`
- [X] T004 [P] Create frontend matchmaking skeletons in `client/features/matchmaking/types.ts`, `client/features/matchmaking/actions.ts`, `client/features/matchmaking/matchmaking-panel.tsx`, `client/features/matchmaking/matchmaking-states.tsx`, and matching `client/features/matchmaking/*.test.tsx` files
- [X] T005 [P] Create localized matchmaking and ranked-game route skeletons in `client/app/[locale]/matchmaking/page.tsx`, `client/app/[locale]/matchmaking/loading.tsx`, `client/app/[locale]/games/[gameId]/page.tsx`, and `client/app/[locale]/games/[gameId]/loading.tsx`
- [X] T006 [P] Create server integration skeletons in `client/lib/api/matchmaking.ts`, `client/lib/api/games.ts`, and `client/app/api/matchmaking/status/route.ts`
- [X] T007 Record clean baseline results for `go test ./...`, `npx pnpm@10.24.0 --dir client test`, `lint`, `typecheck`, `build`, and `npx pnpm@10.24.0 check:openapi` in `specs/008-matchmaking-ranked/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared configuration, durable entities, state types, Redis boundaries, middleware support, and observability required by every story.

**⚠️ CRITICAL**: No user story implementation begins until this phase is complete.

- [X] T008 [P] Add failing configuration tests for map UUID, queue lease, claim TTL, start delay, round count, timer, and scan bounds in `backend/internal/config/config_test.go`
- [X] T009 [P] Add failing migration integrity tests for match lifecycle checks, unique formation/game links, participant uniqueness, and partial active-user uniqueness in `backend/internal/matchmaking/repository_test.go`
- [X] T010 [P] Add failing state-transition and DTO-union tests for queue, claim, match, participant, and public status states in `backend/internal/matchmaking/service_test.go`
- [X] T011 [P] Add failing per-user rate-limit key tests that exclude raw cookies/tokens and isolate users sharing an IP in `backend/internal/middleware/rate_limit_test.go`
- [X] T012 Implement validated matchmaking configuration fields and defaults in `backend/internal/config/config.go`
- [X] T013 Document `MATCHMAKING_DEFAULT_MAP_ID`, `MATCHMAKING_QUEUE_LEASE_SECONDS`, `MATCHMAKING_CLAIM_TTL_SECONDS`, `MATCHMAKING_START_DELAY_SECONDS`, `MATCHMAKING_ROUND_COUNT`, `MATCHMAKING_TIMER_SECONDS`, and `MATCHMAKING_CANDIDATE_SCAN_LIMIT` in `backend/.env.example`
- [X] T014 Create Goose migration `backend/migrations/00014_matchmaking_ranked.sql` with `matches`, `match_players`, lifecycle checks, foreign keys, formation/game uniqueness, participant indexes, and partial unique active assignment enforcement
- [X] T015 Define GORM match and participant models with explicit table names in `backend/internal/matchmaking/model.go`
- [X] T016 Define discriminated public status DTOs, internal queue/claim records, and strict join request DTO in `backend/internal/matchmaking/dto.go`
- [X] T017 Define matchmaking domain errors and safe shared-envelope mappings in `backend/internal/matchmaking/errors.go`
- [X] T018 Define queue coordinator, location selector, clock, durable store, and transaction-aware game lifecycle interfaces in `backend/internal/matchmaking/service.go`
- [X] T019 Create the Redis matchmaking coordinator skeleton, versioned key builders, and script result parsers in `backend/internal/platform/redis/matchmaking.go`
- [X] T020 [P] Add Redis coordinator integration-test harness and availability skip behavior in `backend/internal/platform/redis/matchmaking_test.go`
- [X] T021 Implement a hashed per-user rate-limit key function using resolved registered session identity in `backend/internal/middleware/rate_limit.go`
- [X] T022 Define bounded-label queue command, status, formation, recovery, stale-entry, dependency-failure, and rate-limit metrics in `backend/internal/matchmaking/metrics.go`
- [X] T023 Wire matchmaking configuration validation coverage and environment documentation assertions in `backend/internal/config/config_test.go`
- [X] T024 Verify migration up/down/up and integrity constraints against a disposable PostgreSQL database using `backend/migrations/00014_matchmaking_ranked.sql` and record results in `specs/008-matchmaking-ranked/quickstart.md`

**Checkpoint**: Configuration, durable schema, domain types, Redis boundary, rate-limit identity, and metrics are ready for story work.

---

## Phase 3: User Story 1 - Join Ranked Matchmaking (Priority: P1) 🎯 MVP

**Goal**: An eligible registered player can enter one ranked queue, retry safely without losing priority, and observe a privacy-safe searching state.

**Independent Test**: Sign in as one active registered player, join `ranked_standard` concurrently from two tabs, and confirm both calls return one searching entry with the same original start time while guests, disabled accounts, unsupported modes, and conflicting active games are rejected.

### Tests for User Story 1

- [X] T025 [P] [US1] Add service tests for active-account eligibility, guest denial, disabled-account denial, unsupported mode, active-game conflict, first join, and duplicate join priority preservation in `backend/internal/matchmaking/service_test.go`
- [X] T026 [P] [US1] Add Redis integration tests for atomic join, cross-mode uniqueness, concurrent duplicate joins, stable sorted-set score, lease creation, and stale self-entry cleanup in `backend/internal/platform/redis/matchmaking_test.go`
- [X] T027 [P] [US1] Add handler tests for strict JSON decoding, 202 searching response, 400/401/403/409/422/429/503 mappings, and response privacy in `backend/internal/matchmaking/handler_test.go`
- [X] T028 [P] [US1] Add router tests for route mounting, registered auth, guest rejection, POST CSRF enforcement, and per-user command/status budgets in `backend/internal/app/routes_test.go`
- [X] T029 [P] [US1] Add frontend tests for join action validation, not-queued rendering, pending/disabled join controls, searching rendering, and safe backend error mapping in `client/features/matchmaking/actions.test.ts` and `client/features/matchmaking/matchmaking-panel.test.tsx`
- [X] T030 [P] [US1] Add status proxy tests for cookie forwarding, safe payload forwarding, 429 `Retry-After`, and unavailable mapping in `client/app/api/matchmaking/status/route.test.ts`

### Implementation for User Story 1

- [X] T031 [US1] Implement atomic join and searching-status/lease-renew scripts with original priority preservation in `backend/internal/platform/redis/matchmaking.go`
- [X] T032 [US1] Implement active-user/profile lookup, conflicting active-game check, and active durable-assignment lookup in `backend/internal/matchmaking/repository.go`
- [X] T033 [US1] Implement registered-session parsing, mode validation, eligibility checks, idempotent join, and searching status mapping in `backend/internal/matchmaking/service.go`
- [X] T034 [US1] Implement POST queue and basic GET status handlers with strict request decoding and stable error envelopes in `backend/internal/matchmaking/handler.go`
- [X] T035 [US1] Register matchmaking metrics, repository, Redis coordinator, service, and handler dependencies in `backend/cmd/api/main.go`
- [X] T036 [US1] Add matchmaking handler parameters and dependency forwarding in `backend/internal/app/server.go`
- [X] T037 [US1] Mount registered-only matchmaking routes with CSRF-aware command limits, polling-safe status limits, and metric observers in `backend/internal/app/routes.go`
- [X] T038 [US1] Replace guest/quick-play join and status placeholders with registered `ranked_standard` security, schemas, responses, and error codes in `backend/openapi/openapi.yaml`
- [X] T039 [P] [US1] Define strict TypeScript matchmaking request/status unions in `client/features/matchmaking/types.ts`
- [X] T040 [US1] Implement server-only join and initial-status helpers with `cache: "no-store"` in `client/lib/api/matchmaking.ts`
- [X] T041 [US1] Implement validated join Server Action with serializable safe action state in `client/features/matchmaking/actions.ts`
- [X] T042 [US1] Implement the same-origin backend status proxy in `client/app/api/matchmaking/status/route.ts`
- [X] T043 [US1] Implement localized server-loaded not-queued/searching page and narrow join/search component in `client/app/[locale]/matchmaking/page.tsx` and `client/features/matchmaking/matchmaking-panel.tsx`
- [X] T044 [US1] Add initial English and Arabic join/searching/eligibility/error copy in `client/messages/en.json` and `client/messages/ar.json`

**Checkpoint**: User Story 1 works independently as a registered-only, duplicate-safe ranked waiting queue even before pairing is enabled.

---

## Phase 4: User Story 2 - Form A Fair Match (Priority: P1)

**Goal**: The oldest compatible pair is claimed exactly once and receives one durable, playable ranked game destination.

**Independent Test**: Queue two compatible active players, run concurrent formation attempts, and confirm exactly one match, one ranked game, two game players, two match participants, and the same matched status for both users; retries and failures never duplicate assignments.

### Tests for User Story 2

- [X] T045 [P] [US2] Add Redis tests for bounded oldest-candidate scanning, stale pruning, exact two-player claim, incompatible/missing candidate rejection, and two-coordinator claim races in `backend/internal/platform/redis/matchmaking_test.go`
- [X] T046 [P] [US2] Add repository tests for stable user-row lock order, active-account revalidation, full ranked bundle rollback, unique formation replay, exactly two participants, and partial active-assignment conflicts in `backend/internal/matchmaking/repository_test.go`
- [X] T047 [P] [US2] Add service tests for second-join formation, location shortage, candidate ineligibility, database failure requeue, Redis-finalize failure with durable recovery, and both-player matched responses in `backend/internal/matchmaking/service_test.go`
- [X] T048 [P] [US2] Add concurrent formation race tests using real Redis and PostgreSQL in `backend/internal/matchmaking/concurrency_test.go`
- [X] T049 [P] [US2] Add ranked multiplayer authorization, scheduled-start, early-guess rejection, all-player/deadline round advancement, hidden-coordinate, and final-result tests in `backend/internal/games/multiplayer_test.go` and `backend/internal/games/service_test.go`
- [X] T050 [P] [US2] Add handler/contract tests for matched status shape and absence of opponent/private/hidden data in `backend/internal/matchmaking/handler_test.go`
- [X] T051 [P] [US2] Add frontend tests for non-overlapping searching polls, searching-to-matched transition, polling stop, and locale-aware navigation in `client/features/matchmaking/matchmaking-panel.test.tsx`
- [X] T052 [P] [US2] Add ranked-game component tests for scheduled countdown, disabled early guess, one submission, round refresh, and results transition in `client/features/game/ranked-game.test.tsx`

### Implementation for User Story 2

- [X] T053 [US2] Implement atomic bounded pair claim, exact-token claim finalization, and claim release/requeue scripts in `backend/internal/platform/redis/matchmaking.go`
- [X] T054 [US2] Implement location selection and full game/match/participant formation transaction with stable user locks and formation-key replay in `backend/internal/matchmaking/repository.go`
- [X] T055 [US2] Implement bounded request-driven formation orchestration, candidate revalidation, failure release, durable-first response, and safe logs/metrics in `backend/internal/matchmaking/service.go`
- [X] T056 [US2] Generalize multiplayer-mode detection and participant authorization from private rooms to ranked games in `backend/internal/games/state.go` and `backend/internal/games/service.go`
- [X] T057 [US2] Generalize multiplayer round reads, guess submission, all-player completion, deadline closure, and next-round activation for ranked mode in `backend/internal/games/multiplayer.go`, `backend/internal/games/repository.go`, and `backend/internal/games/service.go`
- [X] T058 [US2] Enforce scheduled round start and hidden-coordinate-safe pre-reveal behavior for ranked mode in `backend/internal/games/service.go` and `backend/internal/games/dto.go`
- [X] T059 [US2] Return durable matched status after formation and repair stale searching state best-effort in `backend/internal/matchmaking/service.go`
- [X] T060 [US2] Complete matched status OpenAPI schemas and ranked game behavior notes in `backend/openapi/openapi.yaml`
- [X] T061 [P] [US2] Define ranked game frontend request/response types in `client/features/game/ranked-types.ts`
- [X] T062 [US2] Implement server-only ranked game, current round, guess, and result helpers in `client/lib/api/games.ts`
- [X] T063 [US2] Implement ranked guess/reload Server Actions with idempotency keys and safe action errors in `client/features/game/ranked-actions.ts`
- [X] T064 [US2] Implement two-second non-overlapping status polling with matched stop/navigation behavior in `client/features/matchmaking/matchmaking-panel.tsx`
- [X] T065 [US2] Implement localized ranked game destination and narrow interactive game component in `client/app/[locale]/games/[gameId]/page.tsx` and `client/features/game/ranked-game.tsx`
- [X] T066 [US2] Add English and Arabic matched/countdown/round/guess/result copy in `client/messages/en.json` and `client/messages/ar.json`

**Checkpoint**: User Story 2 forms one authoritative two-player match and both players can complete its standard ranked game without rating changes.

---

## Phase 5: User Story 3 - Leave Or Recover Queue State (Priority: P2)

**Goal**: A player can leave safely before assignment and recover correct searching/matched/unavailable state across refreshes, expiry, races, and dependency failures.

**Independent Test**: Join and leave repeatedly, race leave against pair claim, expire a lease, simulate crashes before/after PostgreSQL commit, and confirm every result converges on not queued, searching, matched, or temporarily unavailable without corrupting another player.

### Tests for User Story 3

- [X] T067 [P] [US3] Add Redis tests for idempotent leave, exact-entry deletion, leave-versus-claim linearization, lease renewal, expiry, missing hash pruning, claim recovery index, and original-priority requeue in `backend/internal/platform/redis/matchmaking_test.go`
- [X] T068 [P] [US3] Add service tests for leave absent/searching/claimed, durable-first status, refresh recovery, Redis outage with/without durable match, expired state, and bounded lazy claim reconciliation in `backend/internal/matchmaking/service_test.go`
- [X] T069 [P] [US3] Add crash-window integration tests before database commit, after commit/before Redis finalize, and formation-key replay in `backend/internal/matchmaking/concurrency_test.go`
- [X] T070 [P] [US3] Add DELETE/status handler tests for 204 idempotency, 409 claim race, 200 unavailable state, 429 retry metadata, and 503 mapping in `backend/internal/matchmaking/handler_test.go`
- [X] T071 [P] [US3] Add frontend tests for leave pending/success/race recovery, refresh restoration, no-overlap polling, `Retry-After`, bounded backoff, retry, and stop conditions in `client/features/matchmaking/matchmaking-panel.test.tsx`

### Implementation for User Story 3

- [X] T072 [US3] Implement atomic idempotent leave, exact state validation, lease renewal, stale cleanup, and claim recovery operations in `backend/internal/platform/redis/matchmaking.go`
- [X] T073 [US3] Implement durable formation-key and active-assignment recovery queries in `backend/internal/matchmaking/repository.go`
- [X] T074 [US3] Implement leave, durable-first status, lazy expired-claim reconciliation, eligible requeue, and temporarily-unavailable semantics in `backend/internal/matchmaking/service.go`
- [X] T075 [US3] Implement DELETE queue and complete GET status handlers in `backend/internal/matchmaking/handler.go`
- [X] T076 [US3] Complete leave/status response and error contracts including 204, 409, 429, and 503 in `backend/openapi/openapi.yaml`
- [X] T077 [US3] Implement server-only leave helper and leave/retry Server Actions in `client/lib/api/matchmaking.ts` and `client/features/matchmaking/actions.ts`
- [X] T078 [US3] Implement leave, refresh recovery, unavailable, retry, `Retry-After`, and bounded backoff behavior in `client/features/matchmaking/matchmaking-panel.tsx`
- [X] T079 [US3] Implement safe status proxy error/header propagation without backend detail leakage in `client/app/api/matchmaking/status/route.ts`
- [X] T080 [US3] Add English and Arabic leave/cancelled/delayed/unavailable/rate-limited/retry copy in `client/messages/en.json` and `client/messages/ar.json`

**Checkpoint**: User Story 3 survives refresh, abandonment, leave/claim races, crash windows, and Redis uncertainty without false absence or duplicate state.

---

## Phase 6: User Story 4 - Preserve Ranked Match Records (Priority: P2)

**Goal**: Formed, active, completed, cancelled, and failed-to-start matches retain stable participants and game outcomes suitable for future rating calculation.

**Independent Test**: Form, start, complete, cancel, and fail ranked matches; confirm match/game/player transitions are atomic, terminal outcomes are distinguishable, replay is idempotent, and non-results never qualify as completed ranked outcomes.

### Tests for User Story 4

- [X] T081 [P] [US4] Add repository tests for matched-to-active-to-completed and matched/active-to-cancelled/failed transitions, timestamp checks, idempotent replay, and terminal immutability in `backend/internal/matchmaking/repository_test.go`
- [X] T082 [P] [US4] Add game transaction tests proving match and participant lifecycle changes commit or roll back with ranked game start/completion/cancellation in `backend/internal/games/repository_test.go`
- [X] T083 [P] [US4] Add query tests proving only completed match plus completed game qualifies for future ranked-result reads in `backend/internal/matchmaking/repository_test.go`
- [X] T084 [P] [US4] Add migration deletion/retention tests for restricted game, game-player, and user relationships in `backend/internal/matchmaking/repository_test.go`

### Implementation for User Story 4

- [X] T085 [US4] Implement guarded match/participant lifecycle transition methods and completed-result eligibility query in `backend/internal/matchmaking/repository.go`
- [X] T086 [US4] Add a transaction-aware ranked lifecycle callback seam that avoids a `games` to `matchmaking` import cycle in `backend/internal/games/repository.go` and `backend/internal/matchmaking/repository.go`
- [X] T087 [US4] Update ranked game start, terminal round completion, abandonment, and cancellation transactions to transition match participants atomically in `backend/internal/games/repository.go` and `backend/internal/games/multiplayer.go`
- [X] T088 [US4] Wire the ranked lifecycle adapter into game and matchmaking construction in `backend/cmd/api/main.go`
- [X] T089 [US4] Add bounded operational logs and lifecycle metrics without player identifiers or gameplay secrets in `backend/internal/matchmaking/service.go` and `backend/internal/matchmaking/metrics.go`
- [X] T090 [US4] Document durable lifecycle and explicit exclusion of ratings/seasons/divisions/rewards in `backend/README.md` and `docs/phases/backend/phase-09-matchmaking-and-ranked-foundations.md`

**Checkpoint**: User Story 4 provides reliable competitive facts while cancelled and failed matches remain ineligible for future rating updates.

---

## Phase 7: User Story 5 - Understand Matchmaking States Accessibly (Priority: P3)

**Goal**: English, Arabic, keyboard, and screen-reader users receive clear, complete matchmaking and ranked-game state feedback.

**Independent Test**: Exercise every queue and match state in English and Arabic with keyboard-only navigation and accessibility assertions; confirm live updates are announced, errors use alerts, controls expose names/disabled states, RTL remains usable, and focus is never stolen.

### Tests for User Story 5

- [X] T091 [P] [US5] Add accessibility tests for named controls, visible disabled states, polite search/match announcements, alert errors, non-color-only indicators, and focus preservation in `client/features/matchmaking/matchmaking-panel.test.tsx` and `client/features/matchmaking/matchmaking-states.test.tsx`
- [X] T092 [P] [US5] Add ranked-game accessibility tests for countdown announcements, guess control labels, pending/error/result semantics, and keyboard flow in `client/features/game/ranked-game.test.tsx`
- [X] T093 [P] [US5] Add English/Arabic catalog parity tests for all Matchmaking and RankedGame keys in `client/messages/messages.test.ts`
- [X] T094 [P] [US5] Add localized route rendering tests for `/en/matchmaking`, `/ar/matchmaking`, and RTL-safe state composition in `client/features/matchmaking/matchmaking-states.test.tsx`

### Implementation for User Story 5

- [X] T095 [US5] Implement reusable not-queued, joining, searching, delayed, leaving, matched, unavailable, ineligible, unauthorized, rate-limited, retry, and error views in `client/features/matchmaking/matchmaking-states.tsx`
- [X] T096 [US5] Add accessible names, visible focus, pending/disabled behavior, polite live regions, alerts, non-color-only indicators, and focus preservation in `client/features/matchmaking/matchmaking-panel.tsx`
- [X] T097 [US5] Add accessible countdown, guess, transition, error, and result semantics in `client/features/game/ranked-game.tsx`
- [X] T098 [US5] Complete equivalent English and Arabic Matchmaking and RankedGame catalogs in `client/messages/en.json` and `client/messages/ar.json`
- [X] T099 [US5] Verify localized route direction and semantic structure in `client/app/[locale]/matchmaking/page.tsx`, `client/app/[locale]/games/[gameId]/page.tsx`, and `client/app/[locale]/layout.tsx`

**Checkpoint**: All five user stories are independently functional, localized, accessible, and testable.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Close security, operational, performance, CI, documentation, and full validation gates across every story.

- [X] T100 [P] Add dependency-failure, metric-registration, bounded-label, and safe-log regression tests in `backend/internal/matchmaking/metrics_test.go` and `backend/internal/matchmaking/service_test.go`
- [X] T101 [P] Add privacy regression tests that scan matchmaking/game JSON for email, tokens, private preferences, opponent identity, hidden location IDs, and coordinates before reveal in `backend/internal/matchmaking/handler_test.go` and `backend/internal/games/handler_test.go`
- [X] T102 [P] Add benchmark/load fixtures for 10,000 queue members, 20-candidate scan bounds, endpoint latency, and formation duration in `backend/internal/platform/redis/matchmaking_benchmark_test.go` and `backend/internal/matchmaking/service_benchmark_test.go`
- [X] T103 Harden structured logging, error redaction, dependency failure behavior, metric labels, and cleanup paths in `backend/internal/matchmaking/service.go`, `backend/internal/matchmaking/metrics.go`, and `backend/internal/platform/redis/matchmaking.go`
- [X] T104 Update CI to run frontend tests and matchmaking backend/race/OpenAPI gates in `.github/workflows/ci.yml`
- [X] T105 Run `gofmt`, `go vet ./...`, `golangci-lint run`, `go test ./...`, `go test -race ./...`, and `go build ./cmd/api` from `backend/` and record results in `specs/008-matchmaking-ranked/quickstart.md`
- [X] T106 Run migration up/down/up and inspect intended indexes/query plans for active assignment and formation recovery using `backend/migrations/00014_matchmaking_ranked.sql`, then record evidence in `specs/008-matchmaking-ranked/quickstart.md`
- [X] T107 Run `npx pnpm@10.24.0 check:openapi` and reconcile all matchmaking/ranked-game contract findings in `backend/openapi/openapi.yaml`
- [X] T108 Run `npx pnpm@10.24.0 --dir client test`, `lint`, `typecheck`, and `build`, measure the matchmaking initial JavaScript delta, and record results in `specs/008-matchmaking-ranked/quickstart.md`
- [X] T109 Execute two-session happy path, multi-tab idempotency, leave race, refresh recovery, crash simulation, and Redis outage scenarios from `specs/008-matchmaking-ranked/quickstart.md`
- [X] T110 Execute English/Arabic, RTL, keyboard, focus, live-region, alert, and non-color-only browser validation and record evidence in `specs/008-matchmaking-ranked/quickstart.md`
- [X] T111 Review implemented behavior against all FR/SC items and scope exclusions in `specs/008-matchmaking-ranked/spec.md`, documenting any residual risk in `specs/008-matchmaking-ranked/plan.md`
- [X] T112 Update completion status and validation evidence in `docs/phases/backend/phase-09-matchmaking-and-ranked-foundations.md` and `specs/008-matchmaking-ranked/quickstart.md`
- [X] T113 [US3] Fail closed when either authoritative assignment lookup is unavailable, with regression coverage for absent, searching, and claimed Redis states in `backend/internal/matchmaking/service.go` and `backend/internal/matchmaking/service_test.go`
- [X] T114 [US3] Preserve claimed player pointers and recovery payloads until explicit finalize/release, including near-expiry integration coverage in `backend/internal/platform/redis/matchmaking.go` and `backend/internal/platform/redis/matchmaking_test.go`
- [X] T115 [US3] Surface critical claim release failures and record every best-effort Redis cleanup failure in `backend/internal/matchmaking/service.go` and `backend/internal/matchmaking/service_test.go`
- [X] T116 [P] Record exactly one bounded command outcome for join and leave operations in `backend/internal/matchmaking/service.go` and `backend/internal/matchmaking/service_test.go`
- [X] T117 [P] Replace calendar-sensitive Redis test times and clean benchmark entry keys in `backend/internal/platform/redis/matchmaking_test.go` and `backend/internal/platform/redis/matchmaking_benchmark_test.go`
- [X] T118 [P] Strengthen failure-code database integrity and remove the unused matchmaking lifecycle injection in `backend/migrations/00014_matchmaking_ranked.sql`, `backend/internal/matchmaking/service.go`, and `backend/cmd/api/main.go`
- [X] T119 [P] Pin Goose, golangci-lint, and Redocly versions used by CI in `.github/workflows/ci.yml`
- [X] T120 [US3] Preserve active claimed entries across duplicate join retries and route them through durable-first recovery in `backend/internal/platform/redis/matchmaking.go` and `backend/internal/matchmaking/service.go`
- [X] T121 [US3] Fail closed after pair claim or durable formation when PostgreSQL state cannot be confirmed in `backend/internal/matchmaking/service.go`
- [X] T122 [P] Add Redis and service regressions for duplicate claimed joins, post-claim PostgreSQL failures, and post-formation assignment refresh failures in `backend/internal/platform/redis/matchmaking_test.go` and `backend/internal/matchmaking/service_test.go`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; begin immediately.
- **Foundational (Phase 2)**: Depends on Setup; blocks every user story.
- **User Story 1 (Phase 3)**: Depends on Foundational; provides the MVP queue and basic status contract.
- **User Story 2 (Phase 4)**: Depends on Foundational and integrates with US1 queue/status operations to create assignments.
- **User Story 3 (Phase 5)**: Depends on Foundational; integrates with US1/US2 state but can be tested independently through leave/recovery scenarios.
- **User Story 4 (Phase 6)**: Depends on the durable formation delivered by US2.
- **User Story 5 (Phase 7)**: Depends on the visible state surfaces from US1-US3 and ranked-game surface from US2.
- **Polish (Phase 8)**: Depends on every story selected for release.

### User Story Dependency Graph

```text
Setup -> Foundation -> US1 (join/search MVP)
                    -> US2 (pair + playable match) -> US4 (durable terminal lifecycle)
                    -> US3 (leave + recovery)
US1 + US2 + US3 -----------------------------> US5 (complete accessible state UX)
US1 + US2 + US3 + US4 + US5 ----------------> Polish/release gates
```

### Within Each User Story

- Write the listed tests first and confirm they fail for the missing behavior.
- Implement Redis/database/model behavior before service orchestration.
- Implement service behavior before handlers, wiring, and frontend integration.
- Update the relevant OpenAPI contract in the same story that changes the endpoint.
- Complete the independent test before advancing to the next story checkpoint.

## Parallel Opportunities

- Setup tasks T003-T006 can run in parallel after T002 establishes naming.
- Foundational test tasks T008-T011 can run in parallel; models, DTOs, middleware, Redis harness, and metrics can then be split by file ownership.
- After Foundation, US1 queue behavior and the non-dependent parts of US2 durable schema tests can be developed by separate engineers, but US2 integration waits for the US1 queue contract.
- Redis, PostgreSQL, handler, game, and frontend tests within US2 are deliberately separated into different files.
- US3 recovery tests and US4 lifecycle tests can run in parallel after US2 formation contracts stabilize.
- US5 catalog, matchmaking accessibility, ranked-game accessibility, and localized rendering tests are parallelizable.
- Polish privacy, performance, metrics, and CI tasks can run in parallel before the final sequential validation commands.

## Parallel Example: User Story 1

```text
Task T025: Service eligibility and join idempotency tests in backend/internal/matchmaking/service_test.go
Task T026: Atomic Redis join tests in backend/internal/platform/redis/matchmaking_test.go
Task T027: Join/status HTTP contract tests in backend/internal/matchmaking/handler_test.go
Task T029: Join/search frontend tests in client/features/matchmaking/
```

## Parallel Example: User Story 2

```text
Task T045: Redis pair-claim race tests
Task T046: PostgreSQL formation transaction tests
Task T049: Ranked multiplayer game tests
Task T051: Matchmaking polling/navigation tests
Task T052: Ranked-game component tests
```

## Parallel Example: User Story 3

```text
Task T067: Redis leave/expiry/recovery tests
Task T068: Service durable-first status tests
Task T069: Cross-store crash-window tests
Task T071: Frontend leave/backoff/recovery tests
```

## Parallel Example: User Story 4

```text
Task T081: Match lifecycle repository tests
Task T082: Game transaction lifecycle tests
Task T083: Future ranked-result eligibility tests
Task T084: Retention constraint tests
```

## Parallel Example: User Story 5

```text
Task T091: Matchmaking accessibility tests
Task T092: Ranked-game accessibility tests
Task T093: English/Arabic catalog parity tests
Task T094: Localized route/RTL tests
```

## Implementation Strategy

### MVP First: User Story 1

1. Complete Setup and Foundation.
2. Implement US1 join/searching behavior and its frontend surface.
3. Validate one registered player's duplicate-safe queue flow independently.
4. Stop and review Redis correctness, auth/CSRF/rate limits, privacy, and OpenAPI before enabling pairing.

### Incremental Delivery

1. **US1**: Registered player can join and observe one queue entry.
2. **US2**: Two players form and complete one authoritative ranked game.
3. **US3**: Leave, refresh, expiry, outage, and crash recovery become robust.
4. **US4**: Durable terminal lifecycle becomes future-rating ready.
5. **US5**: Complete accessible English/Arabic state experience.
6. **Polish**: Run performance, security, CI, migration, contract, and browser gates.

### Parallel Team Strategy

After Foundation:

- Backend queue engineer: Redis scripts and US1/US3 coordinator behavior.
- Backend data/game engineer: US2 formation, ranked multiplayer, and US4 lifecycle.
- API/integration engineer: handlers, routes, OpenAPI, metrics, rate limits, and cross-store tests.
- Frontend engineer: localized matchmaking, polling proxy, ranked-game destination, accessibility, and catalogs.

## Notes

- `[P]` tasks modify separate files or test surfaces and can proceed concurrently once their phase prerequisites are met.
- User-story labels provide traceability back to [spec.md](./spec.md).
- PostgreSQL is authoritative after match commit; never report `not_queued` before checking durable assignment.
- Redis scripts, not locks, are the correctness boundary for multi-key queue transitions.
- Do not expose raw player identifiers in logs/metrics or opponent/private/hidden gameplay data in status payloads.
- Do not add ratings, seasons, divisions, rewards, parties, tournaments, or duel combat to this feature.
- Commit after each task or cohesive task group and preserve evidence for every constitution gate.
