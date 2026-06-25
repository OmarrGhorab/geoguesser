# Phase 2 - Identity And Player Sessions

Goal: support guest and registered player identity with secure backend-controlled sessions.

## Scope

- `internal/auth`
- `internal/users`
- guest session identity support for gameplay
- auth cookies, token rotation, and CSRF protection
- rate limiting for auth-sensitive endpoints
- session storage and revocation

## APIs

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `POST /api/v1/auth/refresh`
- `GET /api/v1/auth/me`

## Durable Data

- `users`
- `user_profiles`
- `auth_sessions`

## Rules

- Guest sessions can play solo and join rooms.
- Registered sessions unlock persistent profile and competitive features later.
- Refresh tokens are rotated and stored only as hashes.
- Unsafe cookie-authenticated requests require CSRF.

## Design Sources

- `docs/phase-3-database-design.md`
- `docs/phase-4-api-design.md`
- `docs/phase-5-backend-architecture.md`
- `docs/phase-8-technical-specifications.md` feature 11

## Done When

- Registered login, logout, refresh, and `me` work end to end.
- Guest identity can be resolved safely for gameplay flows.
- Auth cookies and CSRF behavior match the documented security model.
- Auth rate limits and regression tests exist.

## Dependencies

- Phase 1
