# Implementation Plan: Profiles Stats Progress

**Branch**: `007-profiles-stats-progress` | **Date**: 2026-07-01 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/007-profiles-stats-progress/spec.md`

## Summary

Implement registered profile management, public-safe player stats, and saved progress foundations for account history surfaces. Backend work adds a dedicated `internal/profiles` package for current-profile read/update flows, hardens `internal/users` public stats and game-history reads, reuses existing `users`, `user_profiles`, `games`, `game_players`, and `guesses` data, and extends OpenAPI plus rate limits around profile updates. Frontend work adds localized profile pages, server-only profile API helpers, profile edit forms, public stats/history surfaces, and clear loading, empty, error, disabled, success, and rate-limited states.

## Technical Context

**Language/Version**: Go 1.25 backend module; Next.js 16.2.9 App Router, React 19.2.4, TypeScript frontend.

**Primary Dependencies**: Chi router, GORM, PostgreSQL, Goose, Redis-backed rate limiting, Prometheus, `log/slog`; Next App Router, native `fetch`, Server Actions for form mutation wrappers where appropriate, next-intl, Tailwind CSS v4, shadcn/Radix patterns, Zustand only for local UI preferences.

**Storage**: PostgreSQL for durable user profile, game participation, guess, score, and history facts. Redis remains limited to rate-limit/idempotency-style ephemeral coordination if needed; profile and progress truth remains PostgreSQL.

**Testing**: Backend `go test ./...`, targeted `profiles`, `users`, `games`, repository, handler, and route tests, OpenAPI validation. Frontend `npx pnpm@10.24.0 --dir client lint`, `typecheck`, `build`, profile component tests, and browser validation for profile view/update, public stats, game history, validation, English localization, and Arabic RTL.

**Target Platform**: Dockerized Go HTTP API plus localized Next.js App Router web client behind same-origin production proxy; local development uses the configured API URL from existing client helpers.

**Project Type**: Full-stack GeoGuess web application.

**Performance Goals**: Current-profile load and public stats/history reads return within 1 second p95 under normal local/test data; profile update returns within 1 second p95; profile page first meaningful content renders within 2 seconds p95; history/stats queries remain bounded and cursor-friendly; profile bundle impact stays small by keeping data loading server-side and only form interactivity client-side where required.

**Constraints**: Registered-only profile management; guests cannot access current-profile reads or updates. Public stats and history must exclude email, auth/session data, private preferences, hidden locations, guess coordinates, and answer-revealing metadata before authorized reveal. Profile updates require validation, CSRF protection for unsafe cookie-authenticated requests, and rate limiting. Frontend visible copy must be localized in `en` and `ar` with RTL support.

**Scale/Scope**: Phase 07 covers current profile read/update, public user stats, public-safe game history/progress summaries, and frontend profile/account-progress surfaces. It does not include guest-to-user progress linking, account deletion, email/password management, OAuth provider management, full avatar upload/moderation, friends, achievements, general leaderboards, or ranked progression.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Architecture boundaries**: PASS. Backend work stays in `backend/internal/profiles`, targeted hardening in `backend/internal/users`, route wiring in `backend/internal/app`, rate-limit wiring in `backend/internal/middleware` or existing route setup, and schema/index changes in Goose migrations if required. Frontend work stays in localized `client/app/[locale]/profile`, `client/app/[locale]/users/[userId]`, `client/features/profile`, `client/lib/api`, and `client/messages`. No GORM AutoMigrate.
- **Framework guidance**: PASS. Relevant installed Next.js 16 docs were read: `01-app/01-getting-started/05-server-and-client-components.md`, `01-app/01-getting-started/06-fetching-data.md`, `01-app/02-guides/data-security.md`, `01-app/02-guides/forms.md`, and `01-app/02-guides/internationalization.md`. Plan keeps profile data fetching in Server Components/server-only helpers and uses Client Components only for form interactivity and browser-only UI state.
- **Testing gates**: PASS. Plan requires backend unit tests for validation/stat aggregation/cursor logic, handler tests for auth/validation/rate-limit/error mappings, repository tests for profile updates and bounded history queries, OpenAPI validation, frontend lint/typecheck/build, profile component tests, and browser validation for profile update/public stats/localization/RTL flows.
- **UX consistency**: PASS. Plan covers loading, empty, error, disabled, success, validation, unauthorized, not-found, and rate-limited states; keyboard focus, semantic markup, accessible names, non-color-only indicators, and reuse of existing Tailwind/shadcn/Radix patterns.
- **Localization and RTL**: PASS. All profile, stats, history, validation, empty, unavailable, rate-limited, and success copy is planned in `client/messages/en.json` and `client/messages/ar.json`; Arabic layout uses existing locale direction handling.
- **Performance budgets**: PASS. Budgets are stated for profile load/update, public stats/history reads, page render, query bounds, and frontend bundle impact. Measurement comes from backend tests/logs and frontend build/browser validation.
- **Contracts and data**: PASS. Existing profile/stats paths in `backend/openapi/openapi.yaml` will be completed/hardened; any saved-progress/history schema/index gaps will use Goose migrations under `backend/migrations/`.
- **Operational readiness**: PASS. Plan covers rate limits, CSRF, validation, logging without email/tokens/private location data, metrics for profile reads/updates/stats/history, `.env.example` review, and no new readiness dependency beyond PostgreSQL and existing Redis rate limiting.

## Project Structure

### Documentation (this feature)

```text
specs/007-profiles-stats-progress/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── profiles-openapi.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── cmd/api/
├── internal/
│   ├── app/
│   ├── auth/
│   ├── config/
│   ├── games/
│   ├── middleware/
│   ├── platform/
│   ├── profiles/
│   └── users/
├── migrations/
└── openapi/

client/
├── app/
│   └── [locale]/
│       ├── profile/
│       └── users/
│           └── [userId]/
├── features/
│   └── profile/
├── lib/
│   └── api/
├── messages/
└── stores/

specs/007-profiles-stats-progress/
└── [planning and validation artifacts]
```

**Structure Decision**: Add `profiles` as the backend owner for current registered-profile reads and updates because the architecture docs define profile ownership separately from `users`. Keep public stats and public-safe game history in `users`, with shared DTOs/contracts aligned through OpenAPI. Frontend profile routes live under localized App Router paths, with server-only API helpers and narrow client/form components for editable controls.

## Complexity Tracking

No constitution violations.

## Phase 0 Research

See [research.md](./research.md).

## Phase 1 Design

See [data-model.md](./data-model.md), [contracts/profiles-openapi.md](./contracts/profiles-openapi.md), and [quickstart.md](./quickstart.md).

## Post-Design Constitution Check

- **Architecture boundaries**: PASS. Design artifacts keep profile management in `internal/profiles`, public account aggregates in `internal/users`, gameplay facts in existing `games`/`game_players`/`guesses` data, frontend account surfaces in localized app/features/lib/message paths, and any index or schema deltas in Goose migrations.
- **Framework guidance**: PASS. Design follows the installed Server/Client Component, data fetching, data security, forms, and internationalization guidance read during planning.
- **Testing gates**: PASS. Design maps current-profile access, profile validation, public stats privacy, history pagination, guest denial, rate limiting, and UI/localization states to automated or recorded validation.
- **UX consistency**: PASS. Design identifies profile view/edit, public profile/stats, history, loading, empty, error, validation, disabled, success, unauthorized, not-found, and rate-limited states with accessibility expectations.
- **Localization and RTL**: PASS. Design keeps visible copy in `en`/`ar` message catalogs and uses existing direction handling for Arabic.
- **Performance budgets**: PASS. Data model and contracts require bounded profile/stats/history reads, cursor pagination, and server-side data loading to avoid avoidable client work.
- **Contracts and data**: PASS. Contract artifact lists REST profile, stats, and history expectations, privacy guarantees, error codes, and OpenAPI updates. Durable schema reuse is explicit; migration needs are bounded to profile/history indexes or future profile fields discovered during implementation.
- **Operational readiness**: PASS. Research, data model, and quickstart cover validation, CSRF/rate-limit/security behavior, logs/metrics/redaction, and readiness impact from PostgreSQL plus existing Redis rate limiting only.
