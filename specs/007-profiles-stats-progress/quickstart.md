# Quickstart: Profiles Stats Progress

This guide validates the Phase 07 profile, public stats, and saved-progress behavior after implementation.

## Prerequisites

- PostgreSQL and Redis are available through the local development environment.
- Backend environment variables are configured from `backend/.env.example`.
- Frontend environment variables are configured from `client/.env.example`.
- Migrations have been applied through the project migration workflow.
- A registered test account exists, plus at least one guest session for authorization checks.

## Setup

```powershell
docker compose up -d postgres redis
cd backend
go test ./...
cd ..
npx pnpm@10.24.0 --dir client install
```

## Backend Validation

### Profile read/update

1. Sign in as a registered user and capture authenticated cookies plus CSRF token.
2. Request `GET /api/v1/profile`.
3. Confirm the response includes editable profile fields, public-safe stats, and saved-progress summary.
4. Confirm the response excludes email, auth/session identifiers, hidden coordinates, raw guess coordinates, and private location data.
5. Submit `PATCH /api/v1/profile` with a valid display name, locale, country, timezone, avatar reference, and preferences.
6. Request `GET /api/v1/profile` again and confirm the update persisted.

### Profile validation and guest denial

1. Submit invalid display names, unsupported locale values, malformed country/timezone values, and unsupported preference keys.
2. Confirm validation responses are stable and the previous profile remains unchanged.
3. Repeat `GET /api/v1/profile` and `PATCH /api/v1/profile` as a guest session.
4. Confirm guest access is denied without returning registered profile data.
5. Exceed the profile update limit and confirm the rate-limited response is clear.

### Public stats and history

1. Complete one or more eligible games as a registered user.
2. Request `GET /api/v1/users/{userId}/stats` from a separate session.
3. Confirm public stats reflect completed eligible games and exclude private account/gameplay data.
4. Request stats for a registered user with no completed games and confirm a valid zero-state response.
5. Request `GET /api/v1/users/{userId}/games?limit=20`.
6. Confirm history is ordered predictably, paginated, and excludes hidden answers, location IDs, raw guess coordinates, and provider metadata.

## Frontend Validation

### Profile page

1. Run the frontend dev server.
2. Visit `/en/profile` as a registered user.
3. Confirm loading, loaded, validation, disabled, saved, and error states are visible and accessible.
4. Update profile fields and confirm saved values appear after refresh.
5. Visit `/ar/profile` and confirm Arabic copy and RTL layout.

### Public profile/stats page

1. Visit the public user stats/history page for a registered user with completed games.
2. Confirm public stats and history render without private account or hidden gameplay details.
3. Visit a user with no completed games and confirm the empty state.
4. Visit a missing user and confirm the not-found state.

## Required Gates

```powershell
cd backend
gofmt -w .
go test ./...
cd ..
npx pnpm@10.24.0 check:openapi
npx pnpm@10.24.0 --dir client lint
npx pnpm@10.24.0 --dir client typecheck
npx pnpm@10.24.0 --dir client build
npx pnpm@10.24.0 --dir client test
```

## Evidence To Record

- Backend test output for profile, users, games, route, and OpenAPI coverage.
- Query evidence or test fixtures proving completed eligible games are counted once.
- Privacy check showing profile/stats/history responses exclude private and hidden gameplay fields.
- Browser evidence for English profile update flow.
- Browser evidence for Arabic RTL profile and public stats states.
- Accessibility evidence for keyboard focus, labels, status messages, disabled states, and non-color-only indicators.

## Evidence Recorded (2026-07-02)

### Backend gates

- `go build ./...` — clean.
- `go test ./...` (with `DATABASE_URL` pointed at local Docker Postgres) — all packages pass except the pre-existing, unrelated `internal/maps` `TestListAndGetMapsIntegration` failure (404 from a chi-router bypass in that test's harness, predates this feature, not touched by profile changes).
- `go test ./internal/profiles/... ./internal/users/... ./internal/games/... -v` — all profiles/games tests pass (`internal/users` has no test files).
- `gofmt -l ./internal/profiles/` — no output (clean).
- `npx pnpm@10.24.0 check:openapi` (`@redocly/cli lint`) — "Woohoo! Your API description is valid," 29 pre-existing warnings, none newly introduced by the `Profile`/`ProgressSummary`/`PublicProfileResponse` schema additions.

### Frontend gates

- `npx pnpm@10.24.0 --dir client lint` — clean.
- `npx pnpm@10.24.0 --dir client typecheck` — clean.
- `npx pnpm@10.24.0 --dir client build` — production build succeeds; `/[locale]/profile` and `/[locale]/users/[userId]` compile as dynamic routes.
- `npx pnpm@10.24.0 --dir client test` — 25/25 tests pass across 11 files, including 5 new `features/profile/*.test.tsx` files covering `ProfileForm`, `ProfileSummary`, `PublicStats`, `GameHistory`, and the loading/unauthorized/not-found/unexpected-error state panels.

### Privacy check

- `ProfileResponse`/`PublicProfileResponse`/`GameHistoryResponse` DTOs (`backend/internal/profiles/dto.go`) never embed auth/session tokens, hidden location answer coordinates, raw guess coordinates, or OAuth provider metadata — confirmed by reading the DTO struct fields and the corresponding `Profile`/`PublicProfileSummary`/`UserGameHistoryItem` OpenAPI schemas, which enumerate an explicit allow-list of fields rather than passing through internal models.
- `PublicProfileSummary`/`PublicProfileResponse` (used for `GET /users/{userId}/stats`) intentionally omits `email`, confirmed against the `Profile` schema (registered/self view) which does include `email` — the split schema is the privacy boundary enforcement point.

### Frontend browser/RTL/accessibility evidence

Verified via `next dev` + `curl` against locale-prefixed routes (unauthenticated session, so both flows exercise the guarded/error states):

- `GET /en/profile` → renders with `dir="ltr"` and English copy "Sign in required" / "Sign in to view your profile.".
- `GET /ar/profile` → renders with `dir="rtl"` and Arabic copy "تسجيل الدخول مطلوب" / "سجل الدخول لعرض ملفك الشخصي وتعديله.".
- `GET /en/users/00000000-0000-0000-0000-000000000000` and the `/ar/` equivalent → render the localized not-found/unexpected-error panels ("Profile not found" / "الملف الشخصي غير موجود").
- Accessibility: `ProfileLoadingSkeleton` exposes `role="status" aria-live="polite"`; `ProfileUnauthorizedPanel`/`ProfileNotFoundPanel` render a single accessible heading plus a focusable `Link` ("Back to play"); `ProfileUnexpectedErrorPanel` uses `role="alert"`; `ProfileForm` wires `aria-invalid`/`aria-describedby` per field to inline validation text (not color-only) and disables the submit button with `t("actions.saving")` copy while `isPending`; the live-region status paragraph (`role="status" aria-live="polite"`) announces save/validation/auth/rate-limit outcomes as text, not color. All verified by direct component source review plus the Vitest assertions in `features/profile/profile-form.test.tsx` and `features/profile/profile-states.test.tsx`. A full manual mouse/keyboard walkthrough in a real browser (as opposed to `curl`+source review) was not performed in this session — recommended before final sign-off if a visual/interaction regression is suspected.

### Performance

- No dedicated load test was run. `getCurrentProfile`/`getPublicProfile`/`getGameHistory` each issue a single backend request per navigation (`cache: "no-store"`), and the backend queries (`internal/profiles/repository.go`) use indexed lookups (`user_id` primary key, cursor-paginated `games` query) consistent with the existing `rooms`/`challenges` read paths — no N+1 patterns introduced.

## Convergence Evidence Recorded (2026-07-02)

### Backend gates

- `gofmt -w internal/middleware/rate_limit.go internal/middleware/auth.go internal/app/routes.go internal/app/routes_test.go internal/profiles/service.go internal/profiles/service_test.go internal/profiles/handler.go internal/profiles/handler_test.go cmd/api/main.go` — applied formatting to Phase 07 backend files.
- `go test ./internal/profiles/... ./internal/app/...` — passed after adding profile handler, validation, logging, and route ownership coverage.
- `go test ./internal/profiles/... ./internal/users/... ./internal/games/... ./internal/app/...` — passed.
- `go test ./...` from `backend/` — passed across all backend packages.
- `npx pnpm@10.24.0 check:openapi` — OpenAPI remains valid with the same 29 warnings, including the pre-existing unused `UserStatsResponse` warning.

### Frontend gates

- `npx pnpm@10.24.0 --dir client test -- --run features/profile` — 36/36 profile tests pass across 12 files, including direct server-action outcome mapping and Arabic/localized profile state assertions.
- `npx pnpm@10.24.0 --dir client typecheck` — clean.
- `npx pnpm@10.24.0 --dir client lint` — clean.
- `npx pnpm@10.24.0 --dir client build` — production build succeeds and compiles `/[locale]/profile` plus `/[locale]/users/[userId]`.

### Convergence implementation evidence

- Profile update validation now rejects display-name control characters, invalid country codes, invalid IANA timezones, unsafe avatar URL schemes or non-image references, unsupported preference keys, and unsupported preference values. Service tests cover invalid and accepted timezone/avatar/preference cases.
- Profile update validation normalizes country codes to uppercase and trims supported locale/timezone/avatar values before persistence.
- `backend/internal/profiles/handler_test.go` now covers current-profile guest denial, owner-safe response shape, invalid JSON, validation error envelopes, public stats invalid/missing users, game-history pagination/error responses, and rate-limit metric recording.
- `backend/internal/app/routes_test.go` verifies public `/users/{userId}/stats` routing reaches the Phase 07 profiles contract rather than a stale legacy users response shape.
- Profile service logs now record privacy-safe operation events for profile reads, updates, validation failures, public stats reads, history reads, denied access, missing users, and invalid pagination. A regression test verifies profile logs do not include raw email or private preference keys.
- Profile update rate-limit rejection observability is wired through `RateLimitWithObserver(..., profilesHandler.RecordRateLimited)` so `profile_updates_rate_limited_total` is incremented before the rejected response is written.
- Frontend profile tests now cover normalized save payloads, validation/rate-limit/unauthorized action outcomes, private-field absence in public/profile form surfaces, Arabic copy, public empty states, and spoiler-safe history rendering.

### Browser, accessibility, and performance notes

- Registered-success browser validation remains represented by automated server-action/component coverage rather than a real authenticated manual browser session in this workspace.
- Arabic RTL and unauthenticated browser route evidence from the previous section remains valid; added profile tests assert Arabic copy on profile form, public stats, history, and unauthorized panels.
- Accessibility evidence is covered by component tests and source review for labels, focusable links/buttons, `role="status"`, `role="alert"`, disabled submit state, and non-color-only validation/status text.
- Performance budget evidence remains query-shape and build/test based: profile/public stats/history helpers issue one backend request per server navigation, repository reads are bounded or cursor-paginated, and the production build completed successfully.
