# Quickstart: Matchmaking And Ranked Foundations

## Local Prerequisites

- Backend Go toolchain and module dependencies under `backend/`
- PostgreSQL and Redis reachable via `backend/.env`
- Frontend Node 22+ and `pnpm@10.24.0`
- Goose CLI for migration validation (`go install github.com/pressly/goose/v3/cmd/goose@latest`)

## Backend Start

```powershell
Set-Location backend
Copy-Item .env.example .env
# Set MATCHMAKING_DEFAULT_MAP_ID to an active map UUID in your database.
go run ./cmd/api
```

Health:

- `GET http://localhost:8080/health`
- `GET http://localhost:8080/ready`

## Frontend Start

```powershell
npx pnpm@10.24.0 --dir client install
npx pnpm@10.24.0 --dir client dev
```

Open:

- `http://localhost:3000/en/matchmaking`
- `http://localhost:3000/ar/matchmaking`

## Validation Commands

```powershell
Set-Location backend
go test ./...
go test -race ./internal/matchmaking/ ./internal/games/ ./internal/platform/redis/ ./internal/app/
go build ./cmd/api
```

Migration validation against a disposable/local database:

```powershell
Set-Location backend
goose -dir ./migrations postgres $env:DATABASE_URL up
goose -dir ./migrations postgres $env:DATABASE_URL status
goose -dir ./migrations postgres $env:DATABASE_URL down-by-one
goose -dir ./migrations postgres $env:DATABASE_URL up
```

OpenAPI gate from repository root:

```powershell
npx pnpm@10.24.0 check:openapi
```

Frontend gates:

```powershell
npx pnpm@10.24.0 --dir client test
npx pnpm@10.24.0 --dir client lint
npx pnpm@10.24.0 --dir client typecheck
npx pnpm@10.24.0 --dir client build
```

## Manual Two-Player Happy Path

1. Open `/en/matchmaking` as registered Player A.
2. Confirm the initial state is not queued and the ranked-search control is enabled.
3. Join `ranked_standard`.
4. Confirm the UI enters searching within one second, announces the change, disables duplicate join, and retains a leave action.
5. Refresh Player A's page and confirm the same original search start is restored.
6. Open a separate browser profile/private session, sign in as registered Player B, and visit `/en/matchmaking`.
7. Join `ranked_standard` as Player B.
8. Confirm both players receive the same match ID and game destination within three seconds and neither remains searching.
9. Refresh both sessions and confirm the matched assignment remains recoverable.
10. Follow the destination in both sessions.
11. Confirm both players see the same scheduled ranked round and cannot submit before its server-controlled start.
12. Submit one guess from each player and confirm the round advances only after both submit or the deadline expires.
13. Complete all rounds and confirm both participants can reload the same final result.
14. Confirm the match and both participant records are terminal and no rating/division change is displayed.

## Implementation Progress Log

### 2026-07-11 — Feature complete (backend + frontend)

| Gate | Result | Notes |
|------|--------|-------|
| Migration 00014 up/down/up | PASS | Disposable `postgres:16` on `:5433`; tables/indexes/checks verified |
| Formation key index plan | PASS | `Index Scan using matches_formation_key_unique` |
| Active assignment index | PASS | Partial unique `match_players_active_user_uidx` present (empty-table plan may seq-scan) |
| `go test ./internal/matchmaking/` | PASS | Formation, leave/recovery, lifecycle, privacy, metrics |
| `go test ./internal/games/` | PASS | Ranked multiplayer gates + guess lifecycle |
| `go test ./internal/app/` | PASS | Matchmaking route auth/CSRF/rate-limit tests |
| `go build ./cmd/api` | PASS | Ranked lifecycle adapter wired |
| `go test -race` (local Windows) | SKIP | Requires `CGO_ENABLED=1`; CI Ubuntu job runs race suite |
| `check:openapi` | PASS | Valid; preexisting warnings only |
| Client vitest | PASS | 60 tests (matchmaking, ranked-game, messages parity, rooms, profile) |
| Client lint | PASS | After purity/effect boundary fixes |
| Client typecheck | PASS | |
| Client build | PASS | Routes include `/[locale]/matchmaking`, `/[locale]/games/[gameId]`, status proxy |
| CI workflow | UPDATED | Frontend `pnpm test`; backend race suite for matchmaking packages |
| Automated concurrency (2 players / 8 workers) | PASS | Exactly one durable match |
| Interactive browser two-session | RESIDUAL | Scripted happy path documented above; run manually before release |
| Interactive AR/RTL browser pass | RESIDUAL | Catalog parity + RTL composition covered in unit tests; full browser QA residual |

### 2026-07-11 — Review remediation (lifecycle, reconcile, rate limit, gates)

| Gate | Result | Notes |
|------|--------|-------|
| `gofmt -l .` | PASS | Empty after `gofmt -w .` |
| `go vet ./...` | PASS | |
| `golangci-lint run ./...` | PASS | 0 issues |
| `go test ./...` | PASS | Full backend suite without shared disposable DB env |
| `go test ./internal/platform/redis -run RateLimiter -count=50` | PASS | Unique ZSET members (seq + UUID) |
| `go test ./internal/matchmaking/` | PASS | Durable-first reconcile; eligible-only requeue; formation activates match |
| `go test ./internal/games/` | PASS | T082 hook commit/rollback + cancel/abandon TX |
| `go build ./cmd/api` | PASS | Metrics observer wired on matchmaking routes |
| `go test -race` (local Windows) | SKIP | CGO disabled; CI Linux race remains authoritative |
| Benchmark fixture | UPDATED | `matchmaking_benchmark_test.go` seeds 10,000 members, scan bound 20 |

**Remediation delivered**

- Ranked formation creates **active** match + participants with the already-active ranked game (lifecycle agreement)
- `CancelMultiplayerGameTx` / `AbandonRankedGame` / `CancelRankedGame` run match cancel hooks in the same TX
- Expired-claim reconciliation is **service-owned**: PostgreSQL formation-key first, then per-player requeue
- Claim release requeues players **independently** (eligible half retained)
- Rate limiter members are guaranteed unique (`unix_nano` + atomic seq + UUID)
- Matchmaking routes use `RateLimitWithObserver` → `RecordRateLimited`; stale prune increments `ObserveStaleEntry`
- T082 tests: completion/cancel hooks commit or roll back with game state

### 2026-07-11 — Final backend hardening

| Gate | Result | Notes |
|---|---|---|
| `go test ./...` | PASS | Full backend unit suite |
| PostgreSQL/Redis integration suite | PASS | Sequential matchmaking, games, Redis, and route packages against disposable services |
| `go vet ./...`, `golangci-lint run`, `go build ./cmd/api` | PASS | 0 lint findings |
| Goose CLI migration run | PASS | Official Goose v3 applied migrations 00001–00014 to a fresh disposable database |
| OpenAPI validation | PASS | Valid with existing repository-wide warnings |

**Hardening delivered**

- Durable-state lookup failures now preserve claims and return a recoverable unavailable state rather than requeueing uncertain players.
- Candidate revalidation now excludes players who acquire a conflicting active game while queued, while preserving the valid opponent's search.
- A bounded launch/setup failure code now transitions active ranked matches and participants to `failed_to_start`/`failed` atomically with game cancellation.
- Lifecycle integration tests assert game, match, and both participant rows commit or roll back together.
- CI now provisions PostgreSQL and Redis, applies migrations with Goose, and runs the matchmaking integration and race suites serially against those services.

**Explicit exclusions (still out of scope)**

- Ratings, seasons, divisions, rewards, parties, tournaments, duel combat
- Phase 12 background matcher/sweeper
