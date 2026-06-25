# Phase 1 - Platform Foundation

Goal: make the backend operationally real before gameplay features build on it.

## Scope

- `cmd/api` runtime wiring.
- `internal/app` router and dependency assembly.
- `internal/config` typed config loading and validation.
- `internal/http` request, response, validation, and error helpers.
- Core middleware:
  - request ID
  - logging
  - recovery
  - timeout
  - CORS
  - security headers
- Platform adapters:
  - PostgreSQL
  - Redis
  - clock
  - ID generation
  - baseline observability hooks
- Health and readiness endpoints.
- Goose migration setup and migration execution path.
- OpenAPI validation workflow.

## Deliverables

- Runnable API service with dependency checks.
- Initial migration chain for MVP foundation tables or migration scaffolding if schema rollout is staged.
- Shared API error envelope and request helpers.
- Baseline CI gates for backend build and tests.

## Design Sources

- `docs/phase-5-backend-architecture.md`
- `docs/phase-9-project-setup.md`
- `docs/phase-4-api-design.md`
- `docs/phase-8-technical-specifications.md` sections for observability, health, and security

## Done When

- `GET /health`, `GET /ready`, `GET /api/v1/health`, and `GET /api/v1/ready` are stable.
- PostgreSQL and Redis are wired through platform packages, not handlers.
- OpenAPI is treated as a maintained contract artifact.
- No feature code depends on package globals.

## Dependencies

None. This phase unlocks every later backend phase.
