# Quickstart: Friends And Social Graph Backend

## Prerequisites

- PostgreSQL available via `DATABASE_URL`
- Redis available via `REDIS_URL` (rate limiting / sessions)
- Goose CLI or project migration workflow
- Go toolchain for `backend/`

## Migrate

```bash
cd backend
# apply through 00015_friends_social_graph.sql
goose -dir migrations postgres "$DATABASE_URL" up
```

Validate reversible:

```bash
goose -dir migrations postgres "$DATABASE_URL" down
goose -dir migrations postgres "$DATABASE_URL" up
```

## Run API

```bash
cd backend
go run ./cmd/api
```

## Smoke scenarios (authenticated registered session + CSRF)

1. `POST /api/v1/friends/requests` with `{ "user_id": "<target>" }` → 201
2. As target: `GET /api/v1/friends/requests/incoming` → pending request
3. As target: `POST /api/v1/friends/requests/{id}/accept` → accepted
4. Both: `GET /api/v1/friends` → each other present
5. `GET /api/v1/leaderboards/friends` → self + accepted friends only
6. `POST /api/v1/friends/{userId}/block` → 204; lists conceal prior friendship
7. `DELETE /api/v1/friends/{userId}/block` as blocker → 204

## Tests

```bash
cd backend
go test ./internal/friends/...
go test ./internal/leaderboards/...
go test ./internal/app/ -run Friends
go test -race ./internal/friends/... ./internal/leaderboards/... ./internal/app/
```

PostgreSQL integration tests skip when `DATABASE_URL` is unset.

## Performance checks

- Use repository benchmarks and `EXPLAIN` notes recorded under Phase 7 polish.
- Budgets: friend list pages p95 < 200 ms local fixtures; friends leaderboard p95 < 300 ms.

## Implementation status

| Phase | Status |
| --- | --- |
| Setup / foundation | Complete |
| US1 requests | Complete |
| US2 accepted friends | Complete |
| US3 block/unblock | Complete |
| US4 friends leaderboard | Complete |
| Polish / release evidence | Complete (local gates); Linux race deferred to CI |

## Local verification performed (2026-07-11)

- Migration `00015` applied; down + up revalidated
- `gofmt` clean, `go vet ./...` pass, `golangci-lint run` 0 issues
- Unit: `go test ./...` (no `DATABASE_URL`) pass
- Integration (Docker Postgres + Redis): friends, leaderboards, matchmaking, games, platform/redis, app pass
- `go build ./cmd/api` pass
- Redocly OpenAPI: valid (pre-existing warnings only)
- Frontend: `pnpm lint`, `pnpm typecheck`, `pnpm build` (see session log)
- `go test -race` blocked on Windows without CGO — use Linux CI

## Residual risks

- Large friend graphs without covering indexes could degrade list latency — mitigated by migration indexes.
- Privacy regressions if new fields are added to DTOs without review — covered by privacy regression tests.
- Windows hosts cannot run the race detector without CGO; rely on `.github/workflows/ci.yml` ubuntu race job.
