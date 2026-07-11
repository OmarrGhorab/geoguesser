# Tasks: Friends and Social Graph Backend

**Input**: Design documents from `/specs/009-friends-social-graph/`  
**Scope**: Backend only  
**Total tasks**: 61

## Phase 1: Setup

- [x] T001 Create the `009-friends-social-graph` feature specification from the approved requirements in `specs/009-friends-social-graph/spec.md`
- [x] T002 Create the technical plan, research decisions, data model, API contract, and validation guide in `specs/009-friends-social-graph/plan.md`, `research.md`, `data-model.md`, `contracts/friends-openapi.md`, and `quickstart.md`
- [x] T003 Create the friends package skeleton in `backend/internal/friends/model.go`, `dto.go`, `errors.go`, `repository.go`, `service.go`, `handler.go`, and `metrics.go`
- [x] T004 [P] Create test skeletons in `backend/internal/friends/service_test.go`, `repository_test.go`, `handler_test.go`, and `metrics_test.go`
- [x] T005 Update the managed Spec Kit plan reference in `AGENTS.md`

---

## Phase 2: Foundational Data and Contracts

- [x] T006 [P] Add failing PostgreSQL tests for sorted pairs, self-pair rejection, pair uniqueness, foreign keys, and lifecycle constraints in `backend/internal/friends/repository_test.go`
- [x] T007 Create reversible Goose migration `backend/migrations/00015_friends_social_graph.sql` with the `friendships` table, lifecycle checks, timestamps, foreign keys, and indexes
- [x] T008 Implement friendship constants, GORM model, UUID pair normalization, and relationship state validation in `backend/internal/friends/model.go`
- [x] T009 [P] Define public-safe friend, request, list, block, page, and command DTOs in `backend/internal/friends/dto.go`
- [x] T010 [P] Define domain errors and stable HTTP error mappings for validation, conflicts, privacy-safe not-found, and dependency failures in `backend/internal/friends/errors.go`
- [x] T011 Define the consumer-side repository interface and service dependencies in `backend/internal/friends/service.go`
- [x] T012 Implement opaque cursor encoding, decoding, validation, default limit 20, and maximum limit 100 in `backend/internal/friends/cursor.go`
- [x] T013 [P] Add bounded command, list, outcome, dependency-failure, and rate-limit metrics in `backend/internal/friends/metrics.go`
- [x] T014 Add friends handler/service/repository dependency slots to `backend/internal/app/server.go` and `backend/cmd/api/main.go`
- [x] T015 Add foundational friendship schemas, parameters, security requirements, and shared errors to `backend/openapi/openapi.yaml`

---

## Phase 3: User Story 1 — Send and Resolve Friend Requests (P1)

### Tests

- [x] T016 [P] [US1] Service tests for registered-session enforcement, UUID validation, self-request rejection, missing/inactive targets, duplicate requests, reciprocal requests, accept authorization, and decline deletion
- [x] T017 [P] [US1] PostgreSQL integration tests for create/accept/decline/conflict flows (concurrency extras when DB available)
- [x] T018 [P] [US1] Repository pagination tests for incoming and outgoing pending requests
- [x] T019 [P] [US1] Handler tests for JSON, status codes, privacy-safe bodies
- [x] T020 [P] [US1] Router tests for registered authentication, CSRF, hashed per-user request limits, and guest rejection

### Implementation

- [x] T021–T027 [US1] Repository, service, handler, routes, OpenAPI for friend requests

---

## Phase 4: User Story 2 — View and Remove Accepted Friends (P1)

- [x] T028–T035 [US2] Accepted list, removal, tests, routes, OpenAPI

---

## Phase 5: User Story 3 — Block and Unblock Users (P1)

- [x] T036–T044 [US3] Block/unblock/list blocked, tests, routes, OpenAPI

---

## Phase 6: User Story 4 — Friends Leaderboard (P2)

- [x] T045–T052 [US4] Friends leaderboard cohort query, service, handler route, OpenAPI, unit/route tests

---

## Phase 7: Polish and Release Readiness

- [x] T053 [P] Metric registration tests
- [x] T054 [P] Privacy regression checks in handler tests (email/password/block ownership)
- [x] T055 Structured logging and metrics on service commands/lists
- [x] T056 Integration query coverage exercised against disposable Postgres (friends/leaderboards package tests)
- [x] T057 Migration `00015` up + down + up validated (local Docker PG + configured DB)
- [x] T058 `gofmt`, `go vet`, `golangci-lint`, unit `go test ./...`, integration friends/leaderboards/matchmaking/games/redis/app, `go build ./cmd/api`
- [~] T059 Linux `go test -race` — not available on this Windows host (`-race requires cgo`); covered by CI on ubuntu-latest
- [x] T060 OpenAPI Redocly lint (valid, pre-existing warnings only) + frontend lint/typecheck/build
- [x] T061 quickstart + phase-10 status updated with verification evidence
