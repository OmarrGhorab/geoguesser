# Tasks: Friends and Social Graph Backend

**Input**: Design documents from `/specs/009-friends-social-graph/`
**Scope**: Backend only
**Total tasks**: 61

## Phase 1: Setup

- [X] T001 Create the `009-friends-social-graph` feature specification from the approved requirements in `specs/009-friends-social-graph/spec.md`
- [X] T002 Create the technical plan, research decisions, data model, API contract, and validation guide in `specs/009-friends-social-graph/plan.md`, `research.md`, `data-model.md`, `contracts/friends-openapi.md`, and `quickstart.md`
- [X] T003 Create the friends package skeleton in `backend/internal/friends/model.go`, `dto.go`, `errors.go`, `repository.go`, `service.go`, `handler.go`, and `metrics.go`
- [X] T004 [P] Create test skeletons in `backend/internal/friends/service_test.go`, `repository_test.go`, `handler_test.go`, and `metrics_test.go`
- [X] T005 Update the managed Spec Kit plan reference in `AGENTS.md`

## Phase 2: Foundational Data and Contracts

- [X] T006 [P] Add failing PostgreSQL tests for sorted pairs, self-pair rejection, pair uniqueness, foreign keys, and lifecycle constraints in `backend/internal/friends/repository_test.go`
- [X] T007 Create reversible Goose migration `backend/migrations/00015_friends_social_graph.sql` with the `friendships` table, lifecycle checks, timestamps, foreign keys, and indexes
- [X] T008 Implement friendship constants, GORM model, UUID pair normalization, and relationship state validation in `backend/internal/friends/model.go`
- [X] T009 [P] Define public-safe friend, request, list, block, page, and command DTOs in `backend/internal/friends/dto.go`
- [X] T010 [P] Define domain errors and stable HTTP error mappings for validation, conflicts, privacy-safe not-found, and dependency failures in `backend/internal/friends/errors.go`
- [X] T011 Define the consumer-side repository interface and service dependencies in `backend/internal/friends/service.go`
- [X] T012 Implement opaque cursor encoding, decoding, validation, default limit 20, and maximum limit 100 in `backend/internal/friends/cursor.go`
- [X] T013 [P] Add bounded command, list, outcome, dependency-failure, and rate-limit metrics in `backend/internal/friends/metrics.go`
- [X] T014 Add friends handler/service/repository dependency slots to `backend/internal/app/server.go` and `backend/cmd/api/main.go`
- [X] T015 Add foundational friendship schemas, parameters, security requirements, and shared errors to `backend/openapi/openapi.yaml`

## Phase 3: User Story 1 — Send and Resolve Friend Requests (P1)

- [X] T016 [P] [US1] Add service tests for registered-session enforcement, UUID validation, self-request rejection, missing/inactive targets, duplicate requests, reciprocal requests, accept authorization, decline deletion, and disabled-caller rejection in `backend/internal/friends/service_test.go`
- [X] T017 [P] [US1] Add PostgreSQL integration tests for stable user lock order, concurrent create/accept, reverse-request races, transactional accept, decline deletion, and rollback behavior in `backend/internal/friends/repository_test.go`
- [X] T018 [P] [US1] Add repository pagination tests for incoming and outgoing pending requests in `backend/internal/friends/repository_test.go`
- [X] T019 [P] [US1] Add handler tests for strict JSON, request IDs, pagination, `201`, `200`, `204`, `400`, `401`, `404`, `409` responses in `backend/internal/friends/handler_test.go`
- [X] T020 [P] [US1] Add router tests for registered authentication, CSRF, hashed per-user request limits, and guest rejection in `backend/internal/app/routes_test.go`
- [X] T021 [US1] Implement active-target lookup and stable sorted user-row locking helpers in `backend/internal/friends/repository.go`
- [X] T022 [US1] Implement transactional create-request, accept-request, and decline-delete operations with shared lock order in `backend/internal/friends/repository.go`
- [X] T023 [US1] Implement cursor-paginated incoming and outgoing request queries with joined public profile data in `backend/internal/friends/repository.go`
- [X] T024 [US1] Implement request validation, active-caller checks, duplicate/conflict behavior, privacy-safe blocked/missing behavior, and authorization in `backend/internal/friends/service.go`
- [X] T025 [US1] Implement request creation, incoming/outgoing list, accept, and decline handlers in `backend/internal/friends/handler.go`
- [X] T026 [US1] Mount friend-request routes with registered authentication, CSRF, 10-per-minute request limits, 30-per-minute action limits, and 120-per-minute read limits in `backend/internal/app/routes.go`
- [X] T027 [US1] Complete friend-request request/response and error contracts including `400`/`503` in `backend/openapi/openapi.yaml`

## Phase 4: User Story 2 — View and Remove Accepted Friends (P1)

- [X] T028 [P] [US2] Add service tests for accepted-list visibility, symmetric access, idempotent removal, inactive-user exclusion, disabled-caller rejection, and DTO privacy in `backend/internal/friends/service_test.go`
- [X] T029 [P] [US2] Add PostgreSQL tests for accepted friendship pagination, joined profile loading, deterministic cursors, and symmetric removal in `backend/internal/friends/repository_test.go`
- [X] T030 [P] [US2] Add handler and route tests for accepted lists, pagination validation, authenticated delete, CSRF, and idempotent `204` responses in `backend/internal/friends/handler_test.go` and `backend/internal/app/routes_test.go`
- [X] T031 [US2] Implement cursor-paginated accepted-friend queries with public-safe profile projection in `backend/internal/friends/repository.go`
- [X] T032 [US2] Implement transactional accepted-friend removal by either participant in `backend/internal/friends/repository.go`
- [X] T033 [US2] Implement accepted-list and remove-friend business rules with active-caller checks in `backend/internal/friends/service.go`
- [X] T034 [US2] Implement accepted-list and `DELETE /friends/{userId}` handlers and routes in `backend/internal/friends/handler.go` and `backend/internal/app/routes.go`
- [X] T035 [US2] Complete accepted-list, pagination, and remove-friend contracts including `503` in `backend/openapi/openapi.yaml`

## Phase 5: User Story 3 — Block and Unblock Users (P1)

- [X] T036 [P] [US3] Add service tests for direct blocking, replacing pending/accepted relationships, same-blocker idempotency, other-blocker privacy, unblock ownership, blocked request concealment, and disabled-caller rejection in `backend/internal/friends/service_test.go`
- [X] T037 [P] [US3] Add PostgreSQL tests for block transitions, blocker preservation, unauthorized unblock no-op, blocked-list pagination ordered by `updated_at`, and concurrent block/request races in `backend/internal/friends/repository_test.go`
- [X] T038 [P] [US3] Add handler and route tests for block/unblock authentication, CSRF, privacy-safe `204`, blocked-list visibility, and rate limits in `backend/internal/friends/handler_test.go` and `backend/internal/app/routes_test.go`
- [X] T039 [US3] Implement serialized block replacement that clears pending/accepted state and records `blocked_by_user_id` in `backend/internal/friends/repository.go`
- [X] T040 [US3] Implement privacy-safe unblock that changes state only when the caller created the block in `backend/internal/friends/repository.go`
- [X] T041 [US3] Implement cursor-paginated blocked-user listing restricted to blocks created by the caller ordered by block time in `backend/internal/friends/repository.go`
- [X] T042 [US3] Implement block/unblock authorization, concealment, and idempotency rules with active-caller checks in `backend/internal/friends/service.go`
- [X] T043 [US3] Implement block, unblock, and blocked-list handlers and routes in `backend/internal/friends/handler.go` and `backend/internal/app/routes.go`
- [X] T044 [US3] Complete block/unblock privacy and response contracts including `503` in `backend/openapi/openapi.yaml`

## Phase 6: User Story 4 — Friends Leaderboard (P2)

- [X] T045 [P] [US4] Add repository tests for self inclusion, accepted-friend inclusion, pending/nonfriend/blocked/inactive exclusion, cohort-relative rank, and cursor pagination in `backend/internal/leaderboards/repository_test.go`
- [X] T046 [P] [US4] Add service tests for registered-session enforcement, disabled-account rejection, limit/cursor validation, direct PostgreSQL reads, and public-safe responses in `backend/internal/leaderboards/service_test.go`
- [X] T047 [P] [US4] Add handler and router tests for authenticated friends leaderboard reads, per-user rate limiting, pagination, and error mapping in `backend/internal/leaderboards/handler_test.go` and `backend/internal/app/routes_test.go`
- [X] T048 [US4] Add the friends leaderboard repository contract and cohort query without importing friends package internals in `backend/internal/leaderboards/service.go` and `backend/internal/leaderboards/repository.go`
- [X] T049 [US4] Implement filtered cohort ranking with existing score, duration, completion-time, and stable-user ordering in `backend/internal/leaderboards/repository.go`
- [X] T050 [US4] Implement authenticated friends leaderboard service behavior with active-caller validation and without Redis page caching in `backend/internal/leaderboards/service.go`
- [X] T051 [US4] Implement `GET /leaderboards/friends` and mount it with registered authentication and hashed per-user rate limiting in `backend/internal/leaderboards/handler.go` and `backend/internal/app/routes.go`
- [X] T052 [US4] Add friends leaderboard schemas, authentication, pagination, error responses, and examples to `backend/openapi/openapi.yaml`

## Phase 7: Polish and Release Readiness

- [X] T053 [P] Add metric registration, bounded-label, command-outcome, rate-limit, and dependency-failure tests in `backend/internal/friends/metrics_test.go`
- [X] T054 [P] Add privacy regression tests proving responses exclude email, private preferences, decline history, block ownership, tokens, and inactive-account distinctions in `backend/internal/friends/handler_test.go`
- [X] T055 Add structured operational logging and metrics for request, accept, decline, remove, block, unblock, list, and friends-leaderboard outcomes in `backend/internal/friends/service.go`, `backend/internal/friends/metrics.go`, and `backend/internal/leaderboards/metrics.go`
- [X] T056 Add query benchmarks for friend list pages in `backend/internal/friends/repository_test.go` and record performance notes in `specs/009-friends-social-graph/quickstart.md`
- [X] T057 Validate migration `00015_friends_social_graph.sql` with full up, down/up, lifecycle constraints, foreign-key behavior, and intended indexes against disposable PostgreSQL
- [X] T058 Run backend `gofmt`, `go vet ./...`, `golangci-lint run`, unit `go test ./...`, targeted PostgreSQL integration tests, and `go build ./cmd/api`
- [ ] T059 Run the authoritative Linux `go test -race` gate for friends, leaderboards, and app route packages through `.github/workflows/ci.yml`
- [X] T060 Validate `backend/openapi/openapi.yaml` with pinned Redocly and run frontend `pnpm lint`, `pnpm typecheck`, and `pnpm build` as constitution regression gates
- [X] T061 Record completed scenarios, performance evidence, residual risks, and Phase 10 implementation status in `specs/009-friends-social-graph/quickstart.md` and `docs/phases/backend/phase-10-friends-social-graph-and-access-controls.md`

## Fixed Assumptions

- Backend-only implementation.
- Targets are identified by existing public user UUIDs.
- Declines delete pending rows.
- Blocking applies only to friend APIs and friends leaderboards.
- Only the blocker may unblock.
- Friends rank is relative to self plus accepted friends.
- Active account revalidation is required on every friends and friends-leaderboard operation.
- No friendship Redis cache, realtime notifications, profile hiding, room restrictions, or matchmaking exclusions.
