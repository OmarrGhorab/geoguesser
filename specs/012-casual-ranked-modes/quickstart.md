# Quickstart: Casual And Ranked Team Modes

This guide validates the finished feature. Use disposable local/test data; the feature migration and seasonal results become intentionally irreversible after real play data exists.

## Prerequisites

- Go 1.25+, Node 22+, pnpm 10.24.0, Docker Compose.
- PostgreSQL and Redis reachable through `backend/.env`.
- An active public map with at least five eligible locations in `MATCHMAKING_DEFAULT_MAP_ID`.
- `client/.env.local` configured with `BACKEND_API_URL`, `NEXT_PUBLIC_REALTIME_URL`, and a browser-restricted `NEXT_PUBLIC_GOOGLE_MAPS_API_KEY`.
- R2 credentials for shared-environment image tests; local storage is sufficient for functional sanitization tests.
- Eight active registered test accounts that are mutual accepted friends for the 4v4 flow.

## Configuration Defaults To Add

Document and validate these exact names in `backend/.env.example`:

```dotenv
CASUAL_MATCHMAKING_ENABLED=false
RANKED_TEAM_MODES_ENABLED=false
TEAM_CHAT_IMAGES_ENABLED=false
PARTY_INVITE_TTL_SECONDS=900
MATCH_RECONNECT_GRACE_SECONDS=90
CASUAL_INACTIVITY_SECONDS=600
MATCH_SWEEP_INTERVAL_SECONDS=5
MATCH_CLAIM_SWEEP_INTERVAL_SECONDS=5
COMPETITIVE_SEASON_DURATION_DAYS=84
COMPETITIVE_INITIAL_RATING=800
COMPETITIVE_ELO_K=32
COMPETITIVE_RESET_FACTOR_BPS=5000
COMPETITIVE_ABANDON_PENALTY=15
COMPETITIVE_TOP500_MIN_MATCHES=25
REALTIME_TICKET_TTL_SECONDS=30
REALTIME_ALLOWED_ORIGINS=http://localhost:3000
REALTIME_OUTBOUND_QUEUE_SIZE=128
TEAM_CHAT_RETENTION_DAYS=30
TEAM_CHAT_REPORT_RETENTION_DAYS=180
TEAM_CHAT_IMAGE_MAX_BYTES=5242880
TEAM_CHAT_IMAGE_MAX_PIXELS=20000000
TEAM_CHAT_IMAGE_MAX_DIMENSION=2048
TEAM_CHAT_CLEANUP_INTERVAL_SECONDS=900
```

Keep the existing five rounds, 60-second Ranked timer, 30-second queue lease, 15-second claim TTL, five-second start delay, and candidate scan limit 20. Enable all three feature flags only after migration succeeds.

## Start And Migrate

```powershell
docker compose up --build
```

Or run services separately:

```powershell
Set-Location backend
go run github.com/pressly/goose/v3/cmd/goose@v3.27.2 -dir ./migrations postgres $env:DATABASE_URL up
go run ./cmd/api
```

```powershell
npx pnpm@10.24.0 --dir client install
npx pnpm@10.24.0 --dir client dev
```

Expected startup evidence:

- Migration `00019` follows `00018`.
- Startup validates competitive/chat/realtime constants and loads exactly one active season.
- `/api/v1/ready` is healthy with PostgreSQL and Redis.
- With object storage unavailable, readiness and text chat remain healthy while image controls show a localized unavailable state.

## Scenario 1: Casual Solo, Duo, And Squad

1. Open independent authenticated contexts for 2, 4, then 8 players.
2. Queue Casual Solo. For Duo/Squad, create complete accepted-friend parties, accept invites, and ready every member.
3. Confirm no player can enter another party, queue, or game; assignments never mix playlist or format.
4. Play five rounds. Confirm no gameplay countdown/speed bonus and that rounds wait for all active players.
5. Verify team totals equal member accuracy sums and results explicitly show no rating change.
6. Test explicit leave and the ten-minute no-interaction safeguard; both close without competitive changes.

## Scenario 2: Ranked Timer, Bonus, And Safe Reveal

1. Queue equal-rating Ranked Solo, Duo, and Squad sides.
2. Confirm every client receives the same 60-second server start/deadline.
3. Submit a 4,000-point accuracy guess halfway through: expected bonus `floor(4000 × 0.05 × 0.5) = 100`, total 4,100.
4. Confirm a zero-score early guess gets zero bonus and a late/missing guess gets zero total.
5. Confirm an early submitter never receives answer coordinates until the shared round closes.
6. Retry with the same idempotency key and receive the stored result; conflicting submissions cannot replace it.
7. Confirm exact team sums, draw behavior, and one stored rating change per participant.

## Scenario 3: Collaboration, Spectating, And Privacy

1. In Duo/Squad, send text/image messages and proposed markers from each teammate.
2. Confirm team-only delivery within one second, stable message order, accessible marker identity, and marker recovery after refresh.
3. Lock a guess; that marker locks and the player can spectate/select teammates while continuing chat.
4. Attempt opponent/non-participant HTTP reads, socket tickets, attachment URLs, marker/view commands, and team spectating; all must deny without private payloads.
5. In Solo, submit early and spectate the opponent's scene. Confirm no map, cursor, marker, coordinates, score, answer, or private UI is shared.
6. Reconnect within 90 seconds and recover via snapshot; exceed grace separately and confirm Ranked forfeit plus abandon penalty.

## Scenario 4: Images, Mute, Report, And Retention

1. Upload valid JPEG/PNG/WebP files ≤5 MB as `team_chat`, complete sanitization, and attach each ready file once.
2. Verify teammates receive only a short-lived sanitized-JPEG URL and never raw storage keys/objects.
3. Reject SVG, GIF, spoofed MIME, corrupt bytes, >5 MB, >20 MP, unauthorized context, reused attachment, and remote URL; verify raw cleanup.
4. Mute/unmute a teammate and verify per-viewer filtering. Report a message/image and confirm idempotent acceptance with no target notification.
5. With a controlled clock, verify unreported cleanup after 30 days, reported retention for 180 days, legal-hold protection, and failed raw cleanup within 24 hours.

## Scenario 5: Placements, Rating, And Divisions

1. Start hidden at rating 800; five placement matches show progress without rating/rank.
2. Verify documented Elo probability, equal-rated `+16/-16`, equal base delta for non-abandoning teammates, and quitter loss delta plus `-15`.
3. Seed every 100-point boundary from 0 through 2,000; verify all 21 mappings, promotion/demotion, floor zero, and Geo Master I's open upper range.
4. Replay finalization concurrently; standings and unique `(match_id,user_id)` history must remain unchanged.

## Scenario 6: World Legend And Rollover

1. Seed 502 eligible standings plus controls with unfinished placements, 24 matches, or bad standing.
2. Verify rating/wins/rating-time ordering and exactly positions 1–500 receive World Legend.
3. Move players across 500/501; membership refreshes within five minutes (target ≤60 seconds).
4. Race season rollover; exactly one next season is created and old final/ending/peak facts freeze.
5. Verify next rating `round(800 + 0.5 × (old - 800))`, floor zero, and placements reset to 0/5.

## Scenario 7: Localization, Accessibility, And Recovery

- Exercise every journey in `en` and `ar`; verify RTL without reversing numeric semantics, keyboard controls, focus restoration, accessible names, live announcements, alert errors, and non-color-only markers.
- Cover no map key, map/socket/Redis/PostgreSQL/R2 failure, unsafe image, stale party, expired invite, queue delay, progression pending, empty leaderboard, and slow-consumer reconnect states.
- Refresh forming/searching/active/submitted/spectating/result states and recover within three seconds.

## Automated Gates

```powershell
Set-Location backend
go test ./internal/parties/... ./internal/matchmaking/... ./internal/games/... ./internal/matchplay/... ./internal/competitive/... ./internal/realtime/... ./internal/uploads/... ./internal/platform/redis/... ./internal/platform/storage/... ./internal/app/...
go test ./...
go run github.com/pressly/goose/v3/cmd/goose@v3.27.2 -dir ./migrations postgres $env:DATABASE_URL status
npx -y @redocly/cli@2.38.0 lint ./openapi/openapi.yaml
```

Linux CI additionally runs the targeted packages with `go test -p 1 -race`. Frontend gates:

```powershell
npx pnpm@10.24.0 --dir client test:unit
npx pnpm@10.24.0 --dir client test:e2e
npx pnpm@10.24.0 --dir client lint
npx pnpm@10.24.0 --dir client typecheck
npx pnpm@10.24.0 --dir client build
```

Performance evidence includes a 10,000-user ticket benchmark, PostgreSQL plans for snapshot/top 500, eight-client WebSocket and slow-consumer timing, 5 MB sanitization timing, and Next.js route bundle output.

## Rollout And Rollback

1. Deploy migration/code with all flags false; validate season setup and legacy Ranked Solo reads.
2. Enable Casual internally and validate all sizes/recovery.
3. Enable Ranked; monitor formation/finalization retries, rating distribution, abandons, and top-500 latency.
4. Enable images after sanitization/cleanup dashboards are healthy.

Rollback disables flags, stops new joins, and preserves/reconciles active durable matches. Never down-migrate `00019` after new party, match, chat, or rating data exists.

## Phase 8 Backend Polish Evidence (2026-07-18)

Backend-only milestone gates T120–T129. Frontend browser/unit gates remain deferred (see plan.md).

### T120 — CI package lists

`.github/workflows/ci.yml` integration and race jobs now include:

`./internal/friends/` `./internal/leaderboards/` `./internal/parties/` `./internal/matchmaking/` `./internal/games/` `./internal/matchplay/` `./internal/competitive/` `./internal/realtime/` `./internal/uploads/` `./internal/platform/redis/` `./internal/platform/storage/` `./internal/platform/workers/` `./internal/app/`

Both steps still use `go test -p 1` (integration) and `go test -p 1 -race` (race) with `DATABASE_URL` + `REDIS_URL` against CI Postgres 16 and Redis 7 services.

### T121 — Migration validation

Commands:

```powershell
Set-Location backend
go run github.com/pressly/goose/v3/cmd/goose@v3.27.2 -dir ./migrations postgres $env:DATABASE_URL status
go run github.com/pressly/goose/v3/cmd/goose@v3.27.2 -dir ./migrations postgres $env:DATABASE_URL up
```

**Evidence (2026-07-18, ~16:48 UTC)**: goose status against the configured `DATABASE_URL` showed migrations `00001`–`00019` applied; `00019_casual_ranked_team_modes.sql` applied at `Sat Jul 18 14:47:53 2026`.

Guarded down behavior (source review of `00019`):

- `+goose Down` raises if any of: parties/members/invites, competitive standings/rating changes, team messages/mutes/reports, or non-`ranked_standard` matches exist.
- Exception text: `migration 00019 is irreversible: party/competitive/team_message rows or non-legacy match modes exist`.
- Season 1 seed is part of Up (`competitive_seasons` sequence 1 / slug `season-1`, active window).

**Not re-run here**: full fresh-DB upgrade cycle from empty schema (requires disposable local Postgres; Docker daemon unavailable on the polish workstation). CI still applies goose `up` on a clean test DB before unit/integration jobs.

Automated coverage: `internal/app/casual_ranked_migration_test.go` (requires `DATABASE_URL`).

### T122 — OpenAPI / Redocly

```powershell
Set-Location backend
npx -y @redocly/cli@2.38.0 lint ./openapi/openapi.yaml
```

**Result (2026-07-18)**: `validated in ~814ms` — **valid**. Exit success with **28 warnings** (pre-existing style/rules: missing `info.license`, localhost/example servers, missing tag descriptions, challenges path ambiguity, a few 2xx/4xx operation rules, one example null vs enum for `placements_required`, unused `UserStatsResponse`). **No critical errors** requiring OpenAPI schema fixes for this gate.

### T123 — gofmt / golangci-lint

```powershell
Set-Location backend
gofmt -w .
# expect: gofmt -l . empty
golangci-lint run ./...
go build ./...
```

**Result (2026-07-18, ~16:49 UTC)**: `gofmt` clean; `golangci-lint run ./...` **0 issues**; `go build ./...` **green**. Easy findings fixed during polish (ineffectual assigns, unused helpers, errcheck, staticcheck, test ChatStore stub completeness).

### T124–T125 — Targeted suites and `go test ./...`

**Unit / memory gate (matches CI "Unit test suite" — no `DATABASE_URL` / `REDIS_URL` override):**

```powershell
Set-Location backend
# Ensure DATABASE_URL and REDIS_URL are unset for this gate
go test ./...
```

| Field | Value |
| --- | --- |
| Started (UTC) | 2026-07-18T16:49:06Z |
| Finished (UTC) | 2026-07-18T16:49:10Z |
| Exit code | **0** |
| Result | All packages with tests **ok** (or cached ok); packages without tests listed as `?` |

Earlier full cold run without env overrides (2026-07-18T16:44:43Z–16:46:46Z, ~123s) also exited **0**.

**Targeted story packages with env-backed Neon/Upstash (`DATABASE_URL` + `REDIS_URL` loaded from `backend/.env`):**

```powershell
go test -count=1 ./internal/parties/... ./internal/matchmaking/... ./internal/games/... ./internal/matchplay/... ./internal/competitive/... ./internal/realtime/... ./internal/uploads/... ./internal/platform/redis/... ./internal/platform/storage/... ./internal/platform/workers/... ./internal/app/...
```

| Field | Value |
| --- | --- |
| Started (UTC) | 2026-07-18T16:47:10Z |
| Finished (UTC) | 2026-07-18T16:48:29Z |
| Exit code | **1** (shared remote DB/Redis residuals — see below) |
| Passed packages | `competitive`, `uploads`, `platform/storage`, `platform/workers`, `app` |
| Failed packages | `parties`, `matchmaking`, `games`, `matchplay`, `realtime`, `platform/redis` (subset of tests) |

### T126 — Race gate

CI job: `go test -p 1 -race` over the T120 package list on **Linux** (`ubuntu-latest`) with Postgres/Redis services.

**Local Windows residual**: `go test -race` failed immediately with `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`. Full race evidence is deferred to Linux CI (expected path for this gate).

### T127 — Backend API/WebSocket validation residuals

Covered by unit/handler suites without remote deps (PASS under unit gate):

- Parties lifecycle, readiness, invite policy stubs
- Matchmaking ticket/formation pure + memory paths
- Games Casual/Ranked team scoring, speed bonus, delayed reveal policy
- Matchplay audience/spectating/snapshot privacy unit tests
- Competitive rating/ranks/season/leaderboard unit tests
- Uploads image sanitizer unit tests
- Realtime hub/handler unit tests

**Not fully re-executed as multi-client live API on this workstation:**

- End-to-end Casual/Ranked 1v1/2v2/4v4 HTTP+WS against a local API process
- Live chat image upload against R2
- Placement/division/top-500 browser or multi-session soak

**Shared remote integration residuals** (when pointing tests at Neon + Upstash from `.env`):

- UUID `Scan` errors (`driver.Value type string ... to a uint8`) on some party/matchmaking formation repository paths — shared DB / driver scanning vs local CI Postgres behavior not re-proven here
- `matches_format_check` / `matches_lifecycle_check` failures when older test seeds insert incomplete playlist/format/lifecycle columns against 00019 constraints
- Redis matchmaking exclusivity + marker throttle failures; some tests still dialed `localhost:6379` despite `REDIS_URL`
- Realtime Pub/Sub extra-event flake on shared Redis
- Docker Desktop unavailable → no clean local Postgres/Redis for isolated integration re-run

**Honest residual risk**: Linux CI with service containers is the authoritative integration/race proof. Shared cloud DBs are not disposable test harnesses and must not be treated as green evidence.

### T128 — Ops notes (flags, workers, retention, rollback, handoff)

#### Feature flags (default **false**)

| Env | Effect when false |
| --- | --- |
| `CASUAL_MATCHMAKING_ENABLED` | Casual queues and party feature path off (parties also require ranked flag for enablement OR) |
| `RANKED_TEAM_MODES_ENABLED` | Ranked team modes + competitive handler feature gate off |
| `TEAM_CHAT_IMAGES_ENABLED` | Image attach/sanitize path returns unavailable; text chat remains |

Party/realtime feature enablement uses `CASUAL_MATCHMAKING_ENABLED \|\| RANKED_TEAM_MODES_ENABLED`.

#### Background workers (started in `cmd/api/main.go`)

| Worker | Config knobs | Role |
| --- | --- | --- |
| Match lifecycle sweep | `MATCH_SWEEP_INTERVAL_SECONDS`, `MATCH_RECONNECT_GRACE_SECONDS`, `CASUAL_INACTIVITY_SECONDS` | Disconnect grace, inactivity abandon, round/match closure |
| Ranked deadline sweep | `MATCH_SWEEP_INTERVAL_SECONDS` | Close expired Ranked rounds without requiring client polling/submission |
| Match claim sweep | `MATCH_CLAIM_SWEEP_INTERVAL_SECONDS` | Recover expired formation claims |
| Competitive progression retry | competitive worker interval | Exact-once rating finalization retries |
| Season rollover | `COMPETITIVE_SEASON_DURATION_DAYS` + rollover runner | Single next-season creation; freeze prior standings |
| Team chat cleanup | `TEAM_CHAT_CLEANUP_INTERVAL_SECONDS`, retention days | Expire unreported messages, retain reported, raw object cleanup |

Workers stop on process shutdown before HTTP drain completes.

#### Retention

| Data | Default |
| --- | --- |
| Unreported team chat | `TEAM_CHAT_RETENTION_DAYS=30` |
| Reported team chat | `TEAM_CHAT_REPORT_RETENTION_DAYS=180` |
| Failed/raw uploads | cleaner raw max-age path (24h class cleanup in cleaner config) |
| Legal hold | reports with legal hold skip delete |

#### Rollback-by-disable

1. Set all three feature flags false and redeploy config (or restart with env).
2. Stop accepting new party/queue joins; existing durable matches continue via lifecycle workers until terminal.
3. Do **not** goose `down` `00019` after party/competitive/chat/non-legacy match rows exist — migration is intentionally irreversible in that state.
4. Schema remains additive; flags only gate new entry points.

#### API consumer handoff (frontend follow-up)

- OpenAPI source of truth: `backend/openapi/openapi.yaml` (validated via Redocly; warnings only).
- Contract narrative: `specs/012-casual-ranked-modes/contracts/casual-ranked-openapi.md`.
- Consumers must use same-origin BFF for cookies/CSRF; realtime uses short-lived one-time tickets (`ticket.{opaque}`) over `Sec-WebSocket-Protocol`, never URLs/logs.
- Stable error codes and resource shapes are backend-owned; visible copy/localization is client-owned.
- Enable flags only after migration `00019` and Season 1 (or active season) are present; readiness requires Postgres + Redis.

#### Review remediation evidence (2026-07-18)

- `go test -count=1 ./...`: PASS, including WebSocket timeout isolation, multi-socket reconnect counts, durable connection transitions, deadline runner, party/chat fanout, and upload finalization regressions.
- `go vet ./...`: PASS.
- `golangci-lint run ./...`: PASS after the final staticcheck cleanup.
- Grafana dashboard: `backend/observability/grafana/dashboards/casual-ranked-backend.json` (JSON parse validated).
- Prometheus rules: `backend/observability/prometheus/casual-ranked-alerts.yml`.
- Local race remains unavailable because Windows has `CGO_ENABLED=0` and no C compiler; T126 remains unchecked pending Linux CI.
- Docker Desktop is unavailable, so T124 disposable PostgreSQL/Redis integration and T127 live multi-client validation remain unchecked rather than being represented as complete.

### T129 — Milestone scope confirmation

This polish pass changes **backend**, **CI**, and **specs/012 planning** paths only. No `client/` implementation is required to close the backend milestone. Deferred frontend gates are listed in `plan.md` (Post-Backend Milestone section).
