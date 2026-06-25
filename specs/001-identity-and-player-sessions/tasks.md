---

description: "Task list for identity and player sessions"

---

# Tasks: Identity and Player Sessions

**Input**: Design documents from `/specs/001-identity-and-player-sessions/`

**Prerequisites**: plan.md, spec.md

**Tests**: Automated verification is required for behavior changes. Include unit, handler, integration, and security tests.

## Phase 1: Database and Configuration

- [x] T001 Add Goose migration `backend/migrations/00002_oauth_accounts.sql` for `user_oauth_accounts` table and indexes
- [x] T002 Add auth secrets and OAuth config to `backend/internal/config/config.go` and `backend/.env.example`
- [x] T003 Add transaction helper in `backend/internal/platform/postgres/transaction.go`
- [x] T004 Add Redis-based rate limit store in `backend/internal/platform/redis/rate_limit.go`

## Phase 2: Foundational Auth Primitives

- [x] T005 [P] Implement password hashing in `backend/internal/auth/passwords.go`
- [x] T006 [P] Implement JWT access tokens and refresh tokens in `backend/internal/auth/tokens.go`
- [x] T007 [P] Implement HTTP-only cookie helpers in `backend/internal/auth/cookies.go`
- [x] T008 [P] Implement auth domain models in `backend/internal/auth/model.go`
- [x] T009 [P] Implement auth DTOs in `backend/internal/auth/dto.go`
- [x] T010 [P] Implement auth domain errors in `backend/internal/auth/errors.go`
- [x] T011 Implement auth repository in `backend/internal/auth/repository.go`
- [x] T012 Implement auth service in `backend/internal/auth/service.go`

## Phase 3: User Story 1 - Email Register/Login/Logout/Refresh

### Tests

- [x] T013 [P] Unit tests for password hashing and verification
- [x] T014 [P] Unit tests for token signing and verification
- [x] T015 Handler tests for `POST /auth/register` success and validation errors
- [x] T016 Handler tests for `POST /auth/login` success and invalid credentials
- [x] T017 Handler tests for `POST /auth/logout`
- [x] T018 Handler tests for `POST /auth/refresh` rotation and reuse detection
- [x] T019 Integration test for full register → me → logout → login → refresh flow

### Implementation

- [x] T020 Implement `POST /auth/register` handler
- [x] T021 Implement `POST /auth/login` handler
- [x] T022 Implement `POST /auth/logout` handler
- [x] T023 Implement `POST /auth/refresh` handler
- [x] T024 Implement `GET /auth/me` handler

## Phase 4: User Story 2 - Google and Discord OAuth

### Tests

- [x] T025 Unit tests for OAuth state token generation and validation
- [x] T026 Handler tests for OAuth initiate and callback endpoints
- [x] T027 Integration tests for Google and Discord sign-in/link flows

### Implementation

- [x] T028 Add OAuth provider config and client setup in `backend/internal/auth/oauth.go`
- [x] T029 Implement `GET /auth/oauth/{provider}` initiate endpoint
- [x] T030 Implement `GET /auth/oauth/{provider}/callback` callback endpoint
- [x] T031 Add `user_oauth_accounts` repository methods

## Phase 5: User Story 3 - Session Rotation

- [x] T032 Ensure refresh token rotation invalidates old token and updates session metadata
- [x] T033 Add reuse detection: reused revoked token revokes the session family

## Phase 6: User Story 4 - Guest Sessions

### Tests

- [x] T034 Handler tests for guest session cookie creation
- [x] T035 Middleware tests for resolving guest identity

### Implementation

- [x] T036 Implement guest session signing and validation in `backend/internal/auth/guest.go`
- [x] T037 Implement guest session cookie issuance in middleware
- [x] T038 Update `/auth/me` to return guest session summary

## Phase 7: User Story 5 - CSRF and Rate Limits

### Tests

- [x] T039 Middleware tests for CSRF validation
- [x] T040 Middleware tests for rate limiting

### Implementation

- [x] T041 Implement CSRF token middleware in `backend/internal/middleware/csrf.go`
- [x] T042 Implement auth rate limit middleware in `backend/internal/middleware/rate_limit.go`
- [x] T043 Apply CSRF and rate limit middleware to auth routes

## Phase 8: Users Package

- [x] T044 Implement `internal/users` model, repository, service, DTOs, errors
- [x] T045 Implement `GET /users/{userId}/stats` handler
- [x] T046 Add tests for users package

## Phase 9: OpenAPI and Integration

- [x] T047 Update `backend/openapi/openapi.yaml` with OAuth paths, schemas, and security schemes
- [x] T048 Wire auth and users handlers into `backend/internal/app/routes.go`
- [x] T049 Run `go test ./...` and fix failures
- [x] T050 Run `go vet ./...` and linter
- [x] T051 Update `backend/.env.example` with all new configuration keys

## Dependencies & Execution Order

- Phase 1 (DB/config) blocks everything else.
- Phase 2 (primitives) blocks Phase 3, 4, 6.
- Phase 7 (CSRF/rate limits) can be built in parallel with Phase 3/4 once Phase 2 is ready.
- Phase 8 (users) depends on Phase 1 and can run in parallel after.
- Phase 9 (OpenAPI/wiring) depends on all previous phases.

## Verification

- Backend tests pass: `go test ./...`
- OpenAPI remains valid.
- Auth endpoints respond with correct status codes and cookies.
- CSRF rejection and rate limiting verified by tests.
