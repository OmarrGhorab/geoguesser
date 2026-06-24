# Phase 9 Project Setup

## Purpose

This phase creates the actual project foundations after product, system, database, API, backend, frontend, and technical specifications are documented.

The repository now uses:

```text
client/   Next.js frontend
backend/  Go API backend
docs/     planning and architecture documentation
```

## Backend Setup

Backend root:

```text
backend/
```

Configured stack:

- Go 1.24+ module, currently built with Go 1.25 locally.
- Chi Router.
- PostgreSQL.
- GORM with PostgreSQL driver.
- Redis.
- Docker multi-stage build.
- Docker Compose services for PostgreSQL, Redis, API, and client.
- GitHub Actions CI.

Initial backend structure:

```text
backend/
  cmd/api/main.go
  internal/app/
  internal/config/
  internal/health/
  internal/middleware/
  internal/platform/postgres/
  internal/platform/redis/
  migrations/
  openapi/
  Dockerfile
  Makefile
```

Current backend endpoints:

```text
GET /health
GET /ready
GET /api/v1/health
GET /api/v1/ready
```

Why this setup:

- `cmd/api` keeps the executable entrypoint thin.
- `internal/*` prevents accidental public package APIs.
- `internal/platform/*` keeps infrastructure adapters separate from domain features.
- Health and readiness are available before product endpoints so Docker, Compose, and future deployment checks can verify runtime dependencies.
- GORM is connected explicitly, but migrations remain Goose-owned. Do not use GORM AutoMigrate in production.

## Frontend Setup

Frontend root:

```text
client/
```

Configured stack:

- Next.js 16.2.9 App Router.
- TypeScript.
- Tailwind CSS v4.
- shadcn/ui with Radix base and RTL support.
- next-intl.
- Zustand.
- React Hook Form.
- Zod.
- Motion.
- pnpm lockfile and package manager metadata.

Initial frontend additions:

```text
client/
  app/[locale]/
  components/ui/button.tsx
  features/game/
  lib/api/
  lib/i18n/
  messages/en.json
  messages/ar.json
  stores/preferences-store.ts
  proxy.ts
  components.json
```

Why this setup:

- The existing prototype is preserved and moved behind the localized App Router path.
- `/` redirects to the default locale.
- `next-intl` routing creates the foundation for `en` and `ar`.
- RTL is enabled early so layout decisions stay honest.
- `lib/api/client.ts` is server-only and uses native `fetch()`.
- Zustand is introduced only for UI preferences, not server state.
- Zod is introduced at trust boundaries through env and schema modules.
- shadcn/ui is initialized through the CLI so generated components match the installed Tailwind v4 and Radix conventions.

## Local Development

Start infrastructure:

```powershell
docker compose up -d postgres redis
```

Run backend:

```powershell
cd backend
go run ./cmd/api
```

Run frontend:

```powershell
cd client
npx pnpm@10.24.0 dev
```

Use `npx pnpm@10.24.0` on this Windows machine because global `pnpm` is not installed and Corepack could not create shims under `C:\Program Files\nodejs`. Docker Compose and CI install the pinned pnpm version explicitly for the same reason.

The old npm lockfile was removed so the frontend has one package-manager source of truth.

## Verification Commands

Backend:

```powershell
cd backend
go test ./...
```

Frontend:

```powershell
cd client
npx pnpm@10.24.0 lint
npx pnpm@10.24.0 typecheck
npx pnpm@10.24.0 build
```

Docker:

```powershell
docker compose build api
```

## CI

GitHub Actions now includes:

- Backend module download, formatting check, tests, and build.
- Frontend pnpm install, lint, typecheck, and build.
- Backend Docker image build.
- Dependabot for Go modules, npm packages, and GitHub Actions.
- CodeQL scanning.
- Trivy image scanning.

## Deferred Setup

These are intentionally not implemented yet:

- Goose migration SQL files from the Phase 3 schema.
- Product feature handlers beyond health/readiness.
- Authentication cookies and CSRF middleware.
- OpenTelemetry, Prometheus, Grafana, and Sentry wiring.
- Testcontainers integration tests.
- Nginx runtime config.
- Production secrets and deployment manifests.

## Phase 9 Exit Criteria

Phase 9 is complete when:

- Backend has a runnable Go API foundation.
- Frontend has the required Next.js stack installed and configured.
- PostgreSQL and Redis can run through Docker Compose.
- CI exists for backend, frontend, Docker, dependency updates, and security scanning.
- Baseline backend and frontend checks pass.
