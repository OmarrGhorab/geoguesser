# Implementation Plan: Authenticated Home Read Model

## Summary

Add a backend-only `internal/home` composition package. It consumes narrow
interfaces implemented by the existing profiles repository, challenges service,
and maps service. The handler stays transport-only; the service validates the
registered session and active viewer, starts three bounded read branches with
shared cancellation, and returns one typed snapshot.

## Constitution Check

- **Architecture**: PASS. Feature package under `backend/internal`; explicit
  constructor wiring in `cmd/api`; no new persistence or migration.
- **Testing**: PASS. Unit, handler, route/auth, metrics, OpenAPI, and full backend
  gates are required.
- **UX/localization**: N/A. Backend-only data contract; no visible copy added.
- **Performance**: PASS. Target p95 is under 300 ms on local fixtures. Exactly
  three bounded branches are permitted and the map list is capped at four.
- **Operations/security**: PASS. Hashed registered-user rate limit, bounded-label
  metrics, safe logs, stable 401/429/503 errors, and no private profile fields.

## Contract

`GET /api/v1/home` returns:

- `viewer`: user ID, display name, optional avatar URL, optional country code;
- `stats`: existing completed-game aggregates;
- `daily_challenge`: existing daily challenge metadata;
- `recommended_maps`: zero to four existing public map DTOs.

All sections are required for a coherent initial snapshot. Dependency failure
returns `503`; the frontend may retain its previous snapshot and retry.

## Complexity Tracking

Fixed concurrency is justified because these three independent reads are always
needed together and directly address the request-round-trip concern. A generic
workflow engine, unbounded goroutines, cache, new SQL read model, and economy or
billing placeholders are rejected as premature.

## Verification

- `go test ./...`: passed.
- `go vet ./...`: passed.
- `go build ./cmd/api`: passed.
- `golangci-lint run ./internal/home ./internal/app ./cmd/api`: passed.
- Full `golangci-lint run`: blocked by pre-existing `gofmt` findings in
  `internal/friends/cursor.go`, `dto.go`, and `errors.go`; none are modified by
  this feature.
- Redocly 2.38.0 validation: passed with repository-wide pre-existing warnings.
- Targeted `go test -race`: unavailable because the Windows toolchain has CGO
  disabled.
- Frontend integration verification: home unit tests passed (53 tests), and
  frontend typecheck, lint, and production build passed.
- Live Docker verification: the rebuilt API became healthy, readiness returned
  `200`, and daily challenge materialization returned `200` with five rounds
  after skipping newer maps with insufficient active locations. A subsequent
  authenticated `GET /api/v1/home` returned `200` with the composed snapshot.
