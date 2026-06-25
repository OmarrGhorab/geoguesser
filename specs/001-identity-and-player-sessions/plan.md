# Implementation Plan: Identity and Player Sessions

**Branch**: `001-identity-and-player-sessions` | **Date**: 2026-06-25 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-identity-and-player-sessions/spec.md`

## Summary

Implement backend identity and session management for the GeoGuess game. Add email/password registration and login, Google and Discord OAuth sign-in, rotated refresh-token sessions, HTTP-only cookie issuance, CSRF protection, guest session support, and rate limiting for auth-sensitive endpoints. Update the database schema and OpenAPI contract accordingly.

## Technical Context

**Language/Version**: Go 1.25

**Primary Dependencies**: Chi router, GORM, PostgreSQL, Redis, golang.org/x/crypto

**Storage**: PostgreSQL for users, profiles, auth sessions, and OAuth connections; Redis for rate limit counters and CSRF token state

**Testing**: `go test ./...` with unit, handler, and integration tests. Integration tests require PostgreSQL and Redis.

**Target Platform**: Linux server / Docker Compose locally

**Project Type**: web-service backend

**Performance Goals**: Auth endpoints p95 latency under 200ms excluding network and external OAuth provider calls

**Constraints**: HTTP-only Secure SameSite cookies; refresh tokens stored as hashes; no secrets in logs

**Scale/Scope**: Single modular monolith backend; supports thousands of concurrent sessions

## Constitution Check

- **Architecture boundaries**: PASS. Work stays in `backend/` under `internal/auth`, `internal/users`, `internal/middleware`, and `internal/platform`. Database changes use Goose migrations. No GORM AutoMigrate in production code.
- **Framework guidance**: N/A for backend-only change. Frontend auth screens are not in scope for this phase.
- **Testing gates**: PASS. Plan includes unit tests for token/password logic, handler tests for request/response shape, integration tests for login/refresh/logout, and security tests for CSRF and rate limits.
- **UX consistency**: N/A for backend-only change. User-facing error codes and messages are stable and safe.
- **Localization and RTL**: N/A for backend-only change. Error messages are safe to localize on the frontend.
- **Performance budgets**: PASS. Auth endpoint p95 target is 200ms; refresh token rotation uses indexed lookups.
- **Contracts and data**: PASS. OpenAPI auth paths and schemas will be updated. Goose migrations will add OAuth table and indexes.
- **Operational readiness**: PASS. New secrets and OAuth credentials will be documented in `.env.example`. Logs will redact tokens, passwords, and CSRF values.

## Project Structure

### Documentation

```text
specs/001-identity-and-player-sessions/
├── spec.md
├── plan.md
└── tasks.md
```

### Source Code

```text
backend/
├── cmd/api/main.go
├── internal/
│   ├── app/
│   │   ├── routes.go          # add auth routes and middleware
│   │   └── server.go          # unchanged
│   ├── auth/
│   │   ├── handler.go         # register, login, logout, refresh, me, oauth routes
│   │   ├── service.go         # auth business logic
│   │   ├── repository.go      # users, profiles, sessions, oauth queries
│   │   ├── model.go           # GORM models for auth domain
│   │   ├── dto.go             # request/response DTOs
│   │   ├── tokens.go          # JWT access token and refresh token handling
│   │   ├── passwords.go       # Argon2id/bcrypt password hashing
│   │   ├── cookies.go         # HTTP-only cookie helpers
│   │   ├── oauth.go           # Google/Discord OAuth flow helpers
│   │   ├── errors.go          # auth domain errors
│   │   └── *_test.go          # tests
│   ├── users/
│   │   ├── handler.go         # public user stats, user lookup
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── model.go
│   │   ├── dto.go
│   │   ├── errors.go
│   │   └── *_test.go
│   ├── middleware/
│   │   ├── auth.go            # session loader middleware
│   │   ├── csrf.go            # CSRF token validation
│   │   └── rate_limit.go      # rate limiting middleware + stores
│   └── platform/
│       ├── postgres/
│       │   └── transaction.go # tx helper
│       └── redis/
│           └── rate_limit.go  # rate limit store
├── migrations/
│   └── 00002_oauth_accounts.sql
└── openapi/
    └── openapi.yaml           # add oauth paths and schemas
```

**Structure Decision**: The GeoGuess backend is a modular monolith. Auth and users are separate feature packages as defined in the architecture docs. OAuth connections get their own table and are owned by the auth package.

## Complexity Tracking

No constitution violations. OAuth adds complexity compared to email-only auth, but it is explicitly required by the user. A simpler email-only alternative was rejected because the user requires Google and Discord sign-in.
