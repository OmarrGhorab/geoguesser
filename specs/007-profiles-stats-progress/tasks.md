# Tasks: Profiles Stats Progress

**Input**: Design documents from `/specs/007-profiles-stats-progress/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/profiles-openapi.md](./contracts/profiles-openapi.md), [quickstart.md](./quickstart.md)

**Tests**: Required by the GeoGuess constitution for backend profile behavior, public stats privacy, saved progress/history queries, API contracts, frontend UI states, localization, accessibility, and browser validation. Story test tasks are listed before implementation tasks and should fail before the matching implementation when the failure can be reproduced locally.

**Organization**: Tasks are grouped by user story so each story can be implemented and validated as an independent increment after the shared foundation is complete.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with other tasks in the same phase because it touches different files or only depends on completed foundation work
- **[Story]**: User story label for traceability; setup, foundation, and polish tasks do not use story labels
- **File paths**: Every task includes concrete repository paths

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the Phase 07 package/frontend skeletons, confirm framework guidance, and prepare shared validation surfaces before feature code starts.

- [ ] T001 Confirm relevant Next.js guidance for Server Components, data fetching, data security, forms, and internationalization in `client/node_modules/next/dist/docs/01-app/01-getting-started/05-server-and-client-components.md`, `client/node_modules/next/dist/docs/01-app/01-getting-started/06-fetching-data.md`, `client/node_modules/next/dist/docs/01-app/02-guides/data-security.md`, `client/node_modules/next/dist/docs/01-app/02-guides/forms.md`, and `client/node_modules/next/dist/docs/01-app/02-guides/internationalization.md`
- [ ] T002 Create the `backend/internal/profiles` package skeleton with `model.go`, `dto.go`, `errors.go`, `repository.go`, `service.go`, `handler.go`, `metrics.go`, `repository_test.go`, `service_test.go`, and `handler_test.go`
- [ ] T003 [P] Create frontend profile feature skeleton files in `client/features/profile/types.ts`, `client/features/profile/profile-form.tsx`, `client/features/profile/profile-summary.tsx`, `client/features/profile/public-stats.tsx`, `client/features/profile/game-history.tsx`, `client/features/profile/profile-states.tsx`, and matching `client/features/profile/*.test.tsx` files
- [ ] T004 [P] Add profile API helper skeletons and action wrappers in `client/lib/api/profile.ts` and `client/features/profile/actions.ts`
- [ ] T005 [P] Create localized route placeholders in `client/app/[locale]/profile/page.tsx`, `client/app/[locale]/profile/loading.tsx`, `client/app/[locale]/users/[userId]/page.tsx`, and `client/app/[locale]/users/[userId]/loading.tsx`
- [ ] T006 [P] Review profile-related OpenAPI placeholders and schema gaps in `backend/openapi/openapi.yaml` against `specs/007-profiles-stats-progress/contracts/profiles-openapi.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared profile domain, DTO, validation, privacy, routing, and contract primitives that every user story depends on.

**Critical**: No user story work should begin until this phase is complete.

- [ ] T007 [P] Define registered profile, public profile summary, stats summary, progress summary, and game history domain models in `backend/internal/profiles/model.go`
- [ ] T008 [P] Define profile request/response DTOs with private-field-safe JSON shapes in `backend/internal/profiles/dto.go`
- [ ] T009 [P] Define stable profile service errors and HTTP error mapping helpers in `backend/internal/profiles/errors.go`
- [ ] T010 [P] Define profile metrics recorders for profile reads, updates, validation failures, stats reads, and history reads in `backend/internal/profiles/metrics.go`
- [ ] T011 [P] Add validation helper tests for display name, locale, country code, timezone, avatar URL, and preferences in `backend/internal/profiles/service_test.go`
- [ ] T012 Implement profile validation helpers and safe preference allowlist in `backend/internal/profiles/service.go`
- [ ] T013 Implement base profile repository queries for current profile reads, profile updates, public profile summaries, stats aggregation, and bounded history reads in `backend/internal/profiles/repository.go`
- [ ] T014 Implement base profile service constructor, dependencies, registered-session guard, profile DTO mapping, public DTO mapping, and progress summary builder in `backend/internal/profiles/service.go`
- [ ] T015 Implement base profile HTTP handler constructor, request parsing, response helpers, and route registration in `backend/internal/profiles/handler.go`
- [ ] T016 Wire `profiles.Repository`, `profiles.Service`, and `profiles.Handler` into application startup in `backend/cmd/api/main.go`
- [ ] T017 Update server and router signatures to accept `profiles.Handler` in `backend/internal/app/server.go` and `backend/internal/app/routes.go`
- [ ] T018 Register `/api/v1/profile` routes with CSRF protection and profile-update rate limiting in `backend/internal/app/routes.go`
- [ ] T019 Expand OpenAPI profile schemas, security, error responses, rate-limit response, and history pagination documentation in `backend/openapi/openapi.yaml`
- [ ] T020 [P] Add frontend profile API types and server-only REST helpers in `client/features/profile/types.ts` and `client/lib/api/profile.ts`
- [ ] T021 [P] Add English and Arabic base profile namespace keys for profile, stats, history, validation, loading, empty, error, disabled, success, unauthorized, not-found, and rate-limited states in `client/messages/en.json` and `client/messages/ar.json`

**Checkpoint**: Foundation ready. User story phases can now proceed in priority order or in parallel by separate implementers.

---

## Phase 3: User Story 1 - Manage Registered Profile (Priority: P1) MVP

**Goal**: A registered player can load their profile, update editable profile fields, and receive clear feedback while private account fields remain hidden.

**Independent Test**: Sign in as a registered player, load `/profile`, submit a valid update, reload the profile, and confirm updated fields persist while private account fields are not returned; repeat as a guest and confirm access is denied.

### Tests for User Story 1

- [ ] T022 [P] [US1] Add profile repository tests for current profile load, missing profile, partial update preservation, explicit optional-field clearing, and updated timestamp behavior in `backend/internal/profiles/repository_test.go`
- [ ] T023 [P] [US1] Add profile service tests for registered-session guard, guest denial, update validation, partial update behavior, private-field redaction, and profile metrics in `backend/internal/profiles/service_test.go`
- [ ] T024 [P] [US1] Add profile handler tests for `GET /api/v1/profile`, `PATCH /api/v1/profile`, CSRF rejection, guest rejection, invalid body handling, and stable error envelopes in `backend/internal/profiles/handler_test.go`
- [ ] T025 [P] [US1] Add route wiring and rate-limit tests for `/api/v1/profile` in `backend/internal/app/routes_test.go`
- [ ] T026 [P] [US1] Add frontend tests for profile load, edit form validation, save success, disabled/pending state, unauthorized state, and private-field absence in `client/features/profile/profile-form.test.tsx` and `client/features/profile/profile-summary.test.tsx`

### Implementation for User Story 1

- [ ] T027 [US1] Implement current-profile load repository method with joined registered profile data in `backend/internal/profiles/repository.go`
- [ ] T028 [US1] Implement profile update repository method with partial update preservation and explicit optional-field clearing in `backend/internal/profiles/repository.go`
- [ ] T029 [US1] Implement `GetProfile` service flow with registered-session authorization, stats/progress summary composition, and safe DTO mapping in `backend/internal/profiles/service.go`
- [ ] T030 [US1] Implement `UpdateProfile` service flow with validation, CSRF-compatible mutation behavior, metrics, and privacy-safe structured logs in `backend/internal/profiles/service.go`
- [ ] T031 [US1] Implement `GET /profile` and `PATCH /profile` handlers with stable error responses in `backend/internal/profiles/handler.go`
- [ ] T032 [US1] Complete `ProfileResponse`, `UpdateProfileRequest`, validation error, unauthorized, forbidden, and rate-limited OpenAPI details in `backend/openapi/openapi.yaml`
- [ ] T033 [US1] Implement server-only profile fetch and update helpers in `client/lib/api/profile.ts`
- [ ] T034 [US1] Implement profile update action wrappers with server-side authorization-safe request handling in `client/features/profile/actions.ts`
- [ ] T035 [US1] Implement localized profile page with server-loaded current profile state in `client/app/[locale]/profile/page.tsx`
- [ ] T036 [US1] Implement profile summary and editable profile form components in `client/features/profile/profile-summary.tsx` and `client/features/profile/profile-form.tsx`
- [ ] T037 [US1] Implement profile loading, validation, disabled, unauthorized, saved, and unexpected-error states in `client/features/profile/profile-states.tsx` and `client/app/[locale]/profile/loading.tsx`
- [ ] T038 [US1] Add English and Arabic copy for profile fields, validation messages, unauthorized access, save success, rate limits, and error recovery in `client/messages/en.json` and `client/messages/ar.json`

**Checkpoint**: User Story 1 is independently functional and testable as the registered-profile MVP.

---

## Phase 4: User Story 2 - View Public-Safe Player Stats (Priority: P1)

**Goal**: Any viewer can load a registered user's public stats without receiving private account fields or hidden gameplay details.

**Independent Test**: Complete games as a registered player, request public stats from another session, and confirm aggregate stats are correct, zero states work, and responses exclude private account and hidden gameplay data.

### Tests for User Story 2

- [ ] T039 [P] [US2] Add public stats repository tests for completed-game aggregation, zero-state users, best/average score math, last-played timestamp, disabled/missing user behavior, and no double counting in `backend/internal/profiles/repository_test.go`
- [ ] T040 [P] [US2] Add public stats service privacy tests for profile summary shaping, email/session/private preference exclusion, hidden location exclusion, and missing-user normalization in `backend/internal/profiles/service_test.go`
- [ ] T041 [P] [US2] Add public stats handler tests for `GET /api/v1/users/{userId}/stats`, invalid IDs, missing users, zero states, and stable not-found responses in `backend/internal/profiles/handler_test.go`
- [ ] T042 [P] [US2] Add frontend public stats tests for loaded, empty, missing-user, error, English, and Arabic RTL states in `client/features/profile/public-stats.test.tsx`

### Implementation for User Story 2

- [ ] T043 [US2] Implement public profile summary and public stats aggregate repository methods in `backend/internal/profiles/repository.go`
- [ ] T044 [US2] Implement `GetPublicStats` service flow with aggregate-only DTO mapping and missing/unavailable-user normalization in `backend/internal/profiles/service.go`
- [ ] T045 [US2] Implement public stats handler and route registration for `GET /users/{userId}/stats` in `backend/internal/profiles/handler.go` and `backend/internal/app/routes.go`
- [ ] T046 [US2] Migrate or delegate existing `GET /users/{userId}/stats` behavior from `backend/internal/users/handler.go`, `backend/internal/users/service.go`, `backend/internal/users/repository.go`, and `backend/internal/users/dto.go` so the public contract is single-owner and privacy-safe
- [ ] T047 [US2] Complete `PublicProfileSummary` and `UserStatsResponse` schemas with privacy notes in `backend/openapi/openapi.yaml`
- [ ] T048 [US2] Implement public stats API helper in `client/lib/api/profile.ts`
- [ ] T049 [US2] Implement localized public user stats page in `client/app/[locale]/users/[userId]/page.tsx`
- [ ] T050 [US2] Implement public stats component and empty/not-found/error states in `client/features/profile/public-stats.tsx` and `client/features/profile/profile-states.tsx`
- [ ] T051 [US2] Add English and Arabic copy for public stats labels, zero state, missing user, unavailable user, and error recovery in `client/messages/en.json` and `client/messages/ar.json`

**Checkpoint**: User Stories 1 and 2 are independently functional and testable.

---

## Phase 5: User Story 3 - Preserve Account Progress Across Sessions (Priority: P2)

**Goal**: A registered player can return after gameplay and see stable account progress and public-safe saved game history without double-counting completed games or exposing hidden answers.

**Independent Test**: Complete and reload gameplay as a registered player, sign out and sign back in, then confirm profile progress, public-safe game history, pagination, and stats remain stable.

### Tests for User Story 3

- [ ] T052 [P] [US3] Add game history repository tests for cursor pagination, deterministic ordering, limit bounds, active/completed/abandoned status handling, and invalid cursor handling in `backend/internal/profiles/repository_test.go`
- [ ] T053 [P] [US3] Add saved progress service tests for no answer spoilers, no location IDs, no raw guess coordinates, no duplicate completion contribution, and registered-only progress summaries in `backend/internal/profiles/service_test.go`
- [ ] T054 [P] [US3] Add handler tests for `GET /api/v1/users/{userId}/games`, pagination parameters, empty history, missing users, and public-safe response shape in `backend/internal/profiles/handler_test.go`
- [ ] T055 [P] [US3] Add frontend game history tests for ordering, pagination, empty state, in-progress summaries, completed summaries, and spoiler-safe rendering in `client/features/profile/game-history.test.tsx`

### Implementation for User Story 3

- [ ] T056 [US3] Implement bounded saved-progress and game-history repository query with cursor parsing and stable sort keys in `backend/internal/profiles/repository.go`
- [ ] T057 [US3] Implement progress/history DTO mapping that excludes hidden round answers, location IDs, answer coordinates, raw guess coordinates, and provider metadata in `backend/internal/profiles/dto.go`
- [ ] T058 [US3] Implement `GetPublicGameHistory` service flow with registered user lookup, pagination bounds, invalid cursor handling, zero-state behavior, and privacy-safe logs in `backend/internal/profiles/service.go`
- [ ] T059 [US3] Implement `GET /users/{userId}/games` handler and route registration in `backend/internal/profiles/handler.go` and `backend/internal/app/routes.go`
- [ ] T060 [US3] Migrate or delegate existing game-history behavior from `backend/internal/users/handler.go`, `backend/internal/users/service.go`, `backend/internal/users/repository.go`, and `backend/internal/users/dto.go` to the Phase 07 safe contract
- [ ] T061 [US3] Add Goose migration for any missing profile/history query indexes discovered during implementation in `backend/migrations/00011_profile_progress_indexes.sql`
- [ ] T062 [US3] Complete `UserGameHistoryResponse`, `UserGameHistoryItem`, cursor, limit, and bad-request OpenAPI details in `backend/openapi/openapi.yaml`
- [ ] T063 [US3] Implement public game-history API helper in `client/lib/api/profile.ts`
- [ ] T064 [US3] Implement game history component with pagination, empty, in-progress, completed, and spoiler-safe states in `client/features/profile/game-history.tsx`
- [ ] T065 [US3] Integrate game history into current profile and public user pages in `client/app/[locale]/profile/page.tsx` and `client/app/[locale]/users/[userId]/page.tsx`
- [ ] T066 [US3] Add English and Arabic copy for game history, progress, pagination, empty states, in-progress labels, completed labels, and unavailable states in `client/messages/en.json` and `client/messages/ar.json`

**Checkpoint**: User Story 3 account progress and saved-history behavior is independently functional and testable.

---

## Phase 6: User Story 4 - Handle Profile Privacy And Boundary States (Priority: P3)

**Goal**: Profile and stats flows handle invalid inputs, unsupported values, excessive update attempts, empty states, disabled accounts, localization, RTL, and accessibility without leaking private data.

**Independent Test**: Exercise profile validation failures, guest access attempts, excessive updates, empty stats/history, missing/disabled users, English and Arabic pages, keyboard focus, accessible names, and non-color-only indicators.

### Tests for User Story 4

- [ ] T067 [P] [US4] Add boundary service tests for malformed profile values, concurrent update preservation, disabled/unavailable accounts, missing profile repair behavior, and privacy-safe error logs in `backend/internal/profiles/service_test.go`
- [ ] T068 [P] [US4] Add handler tests for rate-limited profile updates, CSRF failures, unsupported locale/country/timezone/avatar/preference errors, disabled users, and stable response codes in `backend/internal/profiles/handler_test.go`
- [ ] T069 [P] [US4] Add route-level rate-limit tests for profile update attempts in `backend/internal/app/routes_test.go`
- [ ] T070 [P] [US4] Add frontend profile state tests for loading, empty, validation, disabled, success, unauthorized, not-found, rate-limited, unexpected-error, Arabic RTL, and accessible status states in `client/features/profile/profile-states.test.tsx`

### Implementation for User Story 4

- [ ] T071 [US4] Harden profile update conflict handling, optional field clearing, malformed values, missing profile handling, and disabled account behavior in `backend/internal/profiles/service.go` and `backend/internal/profiles/repository.go`
- [ ] T072 [US4] Enforce profile update rate limits and stable rate-limit error responses in `backend/internal/app/routes.go` and `backend/internal/profiles/handler.go`
- [ ] T073 [US4] Add privacy-safe structured logs for profile reads, profile updates, validation failures, public stats reads, history reads, denied guest access, and rate-limit rejections in `backend/internal/profiles/service.go` and `backend/internal/profiles/handler.go`
- [ ] T074 [US4] Add Prometheus metrics for profile reads, profile updates, validation failures, public stats reads, history reads, and rate-limited updates in `backend/internal/profiles/metrics.go` and `backend/internal/platform/observability/observability.go`
- [ ] T075 [US4] Implement shared frontend profile state surfaces for loading, empty, validation, disabled, success, unauthorized, not-found, rate-limited, and unexpected-error states in `client/features/profile/profile-states.tsx`
- [ ] T076 [US4] Ensure profile form, public stats, and game history controls expose accessible names, keyboard focus behavior, disabled semantics, live status messages, and non-color-only status indicators in `client/features/profile/profile-form.tsx`, `client/features/profile/public-stats.tsx`, and `client/features/profile/game-history.tsx`
- [ ] T077 [US4] Complete English and Arabic translations for all profile privacy, validation, boundary, empty, disabled, rate-limited, and recovery states in `client/messages/en.json` and `client/messages/ar.json`

**Checkpoint**: All user stories are independently functional and testable.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final contract, security, performance, accessibility, localization, operations, and release-readiness checks across all stories.

- [ ] T078 [P] Update Phase 07 quickstart evidence placeholders after implementation validation in `specs/007-profiles-stats-progress/quickstart.md`
- [ ] T079 [P] Update Phase 07 contract notes to match final OpenAPI details in `specs/007-profiles-stats-progress/contracts/profiles-openapi.md`
- [ ] T080 [P] Verify `.env.example` files need no new keys or update `backend/.env.example` and `client/.env.example` for any profile-specific configuration
- [ ] T081 Run backend formatting and full test gate with `gofmt` and `go test ./...` for files under `backend/`
- [ ] T082 Run targeted backend profile/user/game tests with `go test ./internal/profiles/... ./internal/users/... ./internal/games/...` from `backend/`
- [ ] T083 Run OpenAPI validation with `npx pnpm@10.24.0 check:openapi` from repository root
- [ ] T084 Run frontend lint gate with `npx pnpm@10.24.0 --dir client lint`
- [ ] T085 Run frontend typecheck gate with `npx pnpm@10.24.0 --dir client typecheck`
- [ ] T086 Run frontend production build gate with `npx pnpm@10.24.0 --dir client build`
- [ ] T087 Run frontend profile component tests with `npx pnpm@10.24.0 --dir client test` for `client/features/profile/`
- [ ] T088 Run backend migrations against configured PostgreSQL and verify profile/history index readiness using `backend/migrations/`
- [ ] T089 Record privacy evidence that profile, stats, and history responses redact email, auth/session data, private preferences, hidden locations, answer coordinates, raw guess coordinates, and provider metadata in `specs/007-profiles-stats-progress/quickstart.md`
- [ ] T090 Record English profile update browser validation evidence in `specs/007-profiles-stats-progress/quickstart.md`
- [ ] T091 Record Arabic RTL profile/stats/history browser validation evidence in `specs/007-profiles-stats-progress/quickstart.md`
- [ ] T092 Record accessibility validation for keyboard focus, accessible names, disabled states, live status messages, and non-color-only indicators in `specs/007-profiles-stats-progress/quickstart.md`
- [ ] T093 Verify profile load/update and public stats/history read performance budgets using test or log evidence in `specs/007-profiles-stats-progress/quickstart.md`
- [ ] T094 Verify backend logs and metrics avoid raw emails, tokens, private preferences, precise private location data, and hidden gameplay details in `backend/internal/profiles/service.go`

---

## Dependencies & Execution Order

### Phase Dependencies

- Phase 1 Setup has no dependencies and can start immediately.
- Phase 2 Foundational depends on Phase 1 and blocks every user story.
- Phase 3 US1 depends on Phase 2 and is the first MVP increment.
- Phase 4 US2 depends on Phase 2 and can proceed alongside US1 once shared profile DTOs/contracts are stable.
- Phase 5 US3 depends on Phase 2 and benefits from US2 public stats/history contract alignment.
- Phase 6 US4 depends on Phase 2 and hardens every completed story increment.
- Phase 7 Polish depends on the selected story set being complete.

### User Story Dependencies

- US1: Can start after Foundation and delivers registered current-profile management MVP.
- US2: Can start after Foundation and delivers public-safe stats independently of profile editing.
- US3: Can start after Foundation, but should reuse the public profile summary and pagination conventions from US2 where possible.
- US4: Can be developed alongside later stories, but final acceptance needs all error-producing paths.

### Within Each User Story

- Write failing tests first when behavior is locally reproducible.
- Model/DTO/error primitives before repository/service/handler implementation.
- Repository queries before service orchestration.
- Service logic before route wiring and frontend integration.
- Backend contracts before frontend integration.
- Localized copy and accessibility states before story checkpoint validation.

---

## Parallel Opportunities

- Setup tasks T003, T004, T005, and T006 can run in parallel after T001 and T002 are understood.
- Foundational tasks T007 through T011, T020, and T021 can run in parallel.
- Tests within each user story marked [P] can be written in parallel.
- Backend and frontend implementation tasks in a story can proceed in parallel once shared contracts are stable.
- US2 public stats and US3 saved progress can overlap after shared DTOs and route ownership decisions are completed.
- US4 boundary hardening can overlap with US1-US3 implementation after the relevant paths exist.

### Parallel Example: User Story 1

```text
Task: T022 [P] [US1] Add profile repository tests in backend/internal/profiles/repository_test.go
Task: T023 [P] [US1] Add profile service tests in backend/internal/profiles/service_test.go
Task: T024 [P] [US1] Add profile handler tests in backend/internal/profiles/handler_test.go
Task: T026 [P] [US1] Add frontend profile form/summary tests in client/features/profile/
```

### Parallel Example: User Story 2

```text
Task: T039 [P] [US2] Add public stats repository tests in backend/internal/profiles/repository_test.go
Task: T040 [P] [US2] Add public stats service privacy tests in backend/internal/profiles/service_test.go
Task: T041 [P] [US2] Add public stats handler tests in backend/internal/profiles/handler_test.go
Task: T042 [P] [US2] Add frontend public stats tests in client/features/profile/public-stats.test.tsx
```

### Parallel Example: User Story 3

```text
Task: T052 [P] [US3] Add game history repository tests in backend/internal/profiles/repository_test.go
Task: T053 [P] [US3] Add saved progress service tests in backend/internal/profiles/service_test.go
Task: T054 [P] [US3] Add handler tests for GET /api/v1/users/{userId}/games in backend/internal/profiles/handler_test.go
Task: T055 [P] [US3] Add frontend game history tests in client/features/profile/game-history.test.tsx
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1 Setup.
2. Complete Phase 2 Foundational.
3. Complete Phase 3 User Story 1.
4. Stop and validate: a registered user can load and update their current profile, and a guest is denied.

### Public Progress Increment

1. Add Phase 4 User Story 2 public-safe stats.
2. Add Phase 5 User Story 3 saved progress and game history.
3. Validate completed games contribute once, public responses remain privacy-safe, and history pagination is bounded.

### Recovery And Release Readiness

1. Add Phase 6 User Story 4 boundary, privacy, localization, RTL, and accessibility hardening.
2. Complete Phase 7 gates and record evidence in `quickstart.md`.

---

## Notes

- Keep current-profile mutation in `backend/internal/profiles`; keep existing public `users` behavior only if it delegates to the new profile-safe contract.
- Do not expose email, auth tokens, session identifiers, private preferences, hidden locations, answer coordinates, raw guess coordinates, or provider metadata in public stats/history DTOs or logs.
- Do not use GORM AutoMigrate; all schema/index changes belong in Goose migrations under `backend/migrations/`.
- Use `client/node_modules/next/dist/docs/` as the source of truth before changing Next.js App Router APIs.
- Keep canonical server state out of Zustand; profile and stats data should flow through server-only helpers in `client/lib/api/profile.ts`.
- Preserve English and Arabic copy parity for every user-facing state.
