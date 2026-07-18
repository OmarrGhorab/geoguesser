# Implementation Plan: Casual And Ranked Team Modes

**Branch**: `codex/casual-ranked-backend-fixes` | **Date**: 2026-07-18 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/012-casual-ranked-modes/spec.md`

**Tasking scope override (2026-07-18)**: The current `tasks.md` is backend-only by explicit user direction. It covers Go services, PostgreSQL/Redis/storage, HTTP/WebSocket contracts, migrations, workers, observability, and backend verification. All `client/` implementation, localization/RTL UI, browser-only accessibility, frontend tests, and frontend build gates are deferred to a later frontend task set. This backend milestone is not the release-complete full-stack feature and must not be presented as user-shippable until that follow-up is implemented and its constitution gates pass.

## Summary

Deliver signed-in Casual and Ranked matchmaking for 1v1, 2v2, and 4v4 with complete premade parties for team queues, server-authoritative five-round scoring, team totals, ranked speed bonuses, private team collaboration, post-guess spectating, seasonal rating progression, and an exclusive global World Legend top 500. The implementation extends the existing `matchmaking`, `games`, `realtime`, `uploads`, `friends`, and `leaderboards` foundations; adds cohesive `parties`, `matchplay`, and `competitive` backend packages; uses PostgreSQL for durable business facts and Redis for disposable queue/live state; and adds localized Next.js play, match, and competitive routes with narrow realtime Client Components.

## Technical Context

**Language/Version**: Go 1.25 backend module; Next.js 16.2.10 App Router, React 19.2.7, TypeScript 5.9, Tailwind CSS 4.3.

**Primary Dependencies**: Chi Router, GORM, PostgreSQL, Goose, `go-redis/v9`, `coder/websocket`, AWS S3-compatible R2 client, Prometheus, `log/slog`; Next App Router, native `fetch`, Server Actions and Route Handlers, next-intl, Zod, shadcn/ui/Radix, Google Maps JavaScript API, Vitest, Testing Library, and Playwright. Add `golang.org/x/image` only for WebP decoding and high-quality image resizing.

**Storage**: PostgreSQL is authoritative for parties/invites, match rosters/results, guesses, chat/moderation, seasons, competitive standings, and rating history. Redis holds queue tickets and claims, one-time realtime tickets, connection presence/reconnect windows, monotonic channel versions, current proposed markers/view states, and cross-instance Pub/Sub fanout. R2/local object storage holds raw uploads briefly and sanitized private chat derivatives for their retention window.

**Testing**: Go unit, handler, repository, PostgreSQL integration, Redis Lua/concurrency, WebSocket multi-client, image-sanitization, migration, OpenAPI, race, and benchmark tests; frontend Vitest/Testing Library plus Playwright multi-context flows; mandatory backend `go test ./...` and `go test -race ./...` in Linux CI, frontend `npx pnpm@10.24.0 --dir client test`, `lint`, `typecheck`, and `build`.

**Target Platform**: Dockerized Go API on Linux with PostgreSQL, Redis, R2-compatible storage, and a localized Next.js web client. Browsers use same-origin HTTP BFF routes and short-lived one-time tickets for direct `wss` connections to the backend.

**Project Type**: Full-stack realtime web game.

**Performance Goals**: Party commands p95 ≤500 ms; queue/status p95 ≤1 s; compatible ticket formation visible to all members p95 ≤3 s; guess submission p95 ≤500 ms; authorized realtime fanout p95 ≤250 ms and UI-visible p95 ≤1 s for up to eight players; text chat p95 ≤500 ms; 5 MB image validation/sanitization p95 ≤3 s; competitive profile/top-500 reads p95 ≤300 ms. Match snapshots use bounded joins with no N+1 reads; top-500 queries scan at most 501 ordered standings; matchmaking Lua inspects at most 20 candidate tickets. The new play shell adds at most 70 KB gzip initial JavaScript excluding the already-required Google Maps SDK, with chat, results, and competitive panels route- or interaction-lazy.

**Constraints**: Signed-in active accounts only; complete premade Duo/Squad parties; exactly two equal teams; five rounds; Casual has no scoring timer or rating; Ranked uses 60 seconds and the specified bonus; one visible cross-format rating; at most one active party queue/match assignment per user; Redis loss cannot erase durable match/rating/chat facts; hidden answers and opponent-private state never enter unauthorized payloads, logs, or metrics; normal chat access ends 15 minutes after match completion; text chat remains usable if image storage is unavailable; no arbitrary remote images, voice/video, random teammate fill, or automated visual-content moderation in v1.

**Scale/Scope**: Design target of 10,000 simultaneously searching users, 2,000 concurrent matches/16,000 sockets, parties of 1/2/4, five active rounds per match, 60 messages per user per match minute, and one global active-season leaderboard. Feature flags allow staged activation without removing the additive schema.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

- **Architecture boundaries**: PASS. New business ownership is isolated in `internal/parties`, `internal/matchplay`, and `internal/competitive`; `matchmaking` owns ticket coordination and atomic formation, `games` owns round/guess calculations, `realtime` remains transport, and infrastructure adapters remain under `internal/platform`. Schema changes use Goose migration `00019`; no AutoMigrate.
- **Framework guidance**: PASS. Installed Next.js 16.2.10 guidance was read for Server/Client Components, data fetching, caching, Route Handlers, forms/Server Actions, data security, and internationalization. Initial reads remain Server Components/server-only `fetch`; live map, WebSocket, timer, uploads, and chat are narrow Client Components. Mutations re-authorize server-side and use same-origin Route Handlers or Server Actions.
- **Testing gates**: PASS. The design requires pure scoring/rating/audience tests, database and Redis concurrency tests, contract/handler tests, multi-client WebSocket tests, image sanitization tests, localized UI tests, Playwright 1v1/2v2/4v4 flows, and all mandatory project gates.
- **UX consistency**: PASS. Loading, empty, party-not-ready, searching, match-found, countdown, active, submitted, spectating, reconnecting, timed-out, abandoned, result, placement, promoted/demoted, rate-limited, unsafe-image, muted, and service-unavailable states are specified. Existing Tailwind/shadcn/Radix patterns and shared mission map primitives are reused; keyboard focus and live announcements are required.
- **Localization and RTL**: PASS. All visible mode, party, match, chat, result, season, rank, and error copy is added to `en` and `ar` catalogs; direction-sensitive layouts, marker labels, chat order, rank progress, and spectator controls receive Arabic RTL browser coverage.
- **Performance budgets**: PASS. API, fanout, image, query, queue scan, socket scale, and bundle budgets are explicit and measurable through Prometheus histograms, benchmarks, query assertions, build output, and browser traces.
- **Contracts and data**: PASS. `backend/openapi/openapi.yaml` will be updated from the contract artifact, and one additive Goose migration creates/extends all durable entities with legacy `ranked_standard` compatibility.
- **Operational readiness**: PASS. The plan adds validated matchmaking/season/realtime/chat configuration, graceful background workers, storage-degraded text-chat behavior, existing PostgreSQL/Redis readiness dependencies, redacted metrics/logging, abuse rate limits, retention cleanup, feature flags, and rollback-by-disable guidance.

## Project Structure

### Documentation (this feature)

```text
specs/012-casual-ranked-modes/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── casual-ranked-openapi.md
├── checklists/
│   └── requirements.md
└── tasks.md                 # generated later by /speckit-tasks
```

### Source Code (repository root)

```text
backend/
├── cmd/api/main.go
├── internal/
│   ├── app/
│   ├── config/
│   ├── parties/             # durable party/invite/readiness lifecycle
│   ├── matchmaking/         # generalized queue tickets, claims, formation
│   ├── games/               # team round state, speed scoring, safe reveal
│   ├── matchplay/           # match snapshot, chat, markers, spectating policy
│   ├── competitive/         # seasons, Elo changes, divisions, top 500
│   ├── realtime/            # authenticated party/match WebSocket transport
│   ├── uploads/             # purpose-bound sanitized chat attachments
│   └── platform/{redis,storage}/
├── migrations/00019_casual_ranked_team_modes.sql
└── openapi/openapi.yaml

client/
├── app/
│   ├── api/{parties,matchmaking,matches,realtime,uploads}/
│   └── [locale]/{play,matches/[matchId],competitive}/
├── features/{play,party,matchplay,competitive}/
├── features/mission/        # source of map/panorama primitives extracted for reuse
├── lib/api/
└── messages/{en,ar}.json
```

**Structure Decision**: Keep authoritative business facts in focused backend feature packages and use interfaces for cross-package policy hooks: parties consumes a narrow friendship/block policy, matchmaking consumes party and competitive snapshots, games invokes match/competitive transaction hooks, and realtime delegates command authorization to matchplay. The client receives server-rendered initial state and hydrates only the live party/match surfaces; canonical party, match, and rating state is never placed in Zustand.

## Complexity Tracking

No constitution violations. Three new backend packages are justified by distinct lifecycles and security boundaries; combining parties, realtime collaboration, and seasonal rating into `matchmaking` would create a single package that owns unrelated durable and ephemeral concerns.

## Phase 0 Research

See [research.md](./research.md). All technical and product-policy unknowns are resolved there, including queue ticket atomicity, rating formula, season reset, top-500 ordering, realtime authentication/fanout, image sanitization, retention, and frontend data flow.

## Phase 1 Design

See [data-model.md](./data-model.md), [contracts/casual-ranked-openapi.md](./contracts/casual-ranked-openapi.md), and [quickstart.md](./quickstart.md).

### Delivery Slices

1. **Foundations and compatibility**: additive migration, configuration/feature flags, package seams, legacy `ranked_standard` alias, shared client API schemas, and observability.
2. **Parties and generalized matchmaking**: complete premade party lifecycle, readiness, ticket-based Solo/Duo/Squad queues, atomic equal-team formation, and recoverable assignment.
3. **Authoritative match gameplay**: team slots/totals, Casual no-timer progression, Ranked timer/speed bonus, safe delayed reveal, forfeits, results, and exact-once lifecycle hooks.
4. **Realtime collaboration**: one-time socket tickets, targeted multi-instance fanout, presence/reconnect, proposed markers, view-only spectating, team text chat, image sanitization, mute/report, and retention cleanup.
5. **Competitive progression**: Elo finalization, placements, 21 divisions, seasonal reset/rollover, indexed global top 500, World Legend display, profile/history, and leaderboard.
6. **Localized frontend and rollout**: play/party/queue, responsive match shell, chat/spectator/results, competitive screens, home links, en/ar/RTL/a11y, staged flags, browser evidence, and full gates.

## Post-Design Constitution Check

- **Architecture boundaries**: PASS. The data model and contracts preserve package ownership, transactional hooks, infrastructure adapters, and a single additive Goose migration.
- **Framework guidance**: PASS. Server-rendered initial pages, `server-only` fetch modules, uncached user state, same-origin mutation proxies, serializable props, and narrow browser-only clients match the installed Next.js guidance.
- **Testing gates**: PASS. The quickstart maps each independently testable story to unit, contract, integration, realtime, browser, localization, and mandatory build commands.
- **UX consistency**: PASS. Every interactive journey includes disabled/pending/error/recovery states, accessible status announcements, focus restoration, and reusable visual primitives.
- **Localization and RTL**: PASS. Contracts expose stable codes while client catalogs own visible wording; RTL-sensitive map overlays and chat/spectator ordering are explicitly tested.
- **Performance budgets**: PASS. Bounded Redis scans, indexed standings, paginated chat, compact targeted events, throttled view state, bounded socket queues, sanitized asset caps, and lazy client panels satisfy stated budgets.
- **Contracts and data**: PASS. The contract defines additive/changed HTTP and WebSocket shapes, while the data model specifies migration backfills, constraints, indexes, retention, and rollback behavior.
- **Operational readiness**: PASS. Startup validation, graceful workers, reconnect/version recovery, Redis/PostgreSQL failure precedence, R2 degradation, redacted telemetry, abuse limits, flags, and season rollover are defined.

## Backend-Only Milestone Confirmation (Phase 8, 2026-07-18)

**Status**: Backend milestone polish gates T120–T129 completed for planning/CI/docs and local unit verification. This is **not** release-complete full-stack delivery.

### What this milestone owns

- Go packages: `parties`, `matchmaking`, `games`, `matchplay`, `competitive`, `realtime`, `uploads`, platform redis/storage/workers, app wiring
- Goose migration `00019_casual_ranked_team_modes.sql` (additive; guarded irreversible down)
- OpenAPI HTTP/WS contracts in `backend/openapi/openapi.yaml`
- Feature flags, background workers, retention cleanup, competitive season ops
- CI backend integration/race package expansion
- Evidence recorded in [quickstart.md](./quickstart.md)

### Operational notes (implemented)

| Area | Implementation summary |
| --- | --- |
| Flags | `CASUAL_MATCHMAKING_ENABLED`, `RANKED_TEAM_MODES_ENABLED`, `TEAM_CHAT_IMAGES_ENABLED` default false; parties/realtime enable when either matchmaking flag is true |
| Workers | Lifecycle sweep, ranked round-deadline sweep, claim recovery, progression retry, season rollover, chat/raw cleanup — started from `cmd/api`, stopped on shutdown |
| Retention | 30d unreported chat, 180d reported, legal-hold skip, raw upload cleanup; text chat remains when image storage is degraded |
| Rollback | Disable flags; do not down-migrate `00019` after real party/match/chat/rating data exists |
| Season ops | Exactly one active season enforced; rollover creates next season and freezes prior final/peak facts; initial rating 800, reset factor BPS 5000, abandon penalty 15, top-500 min matches 25 |
| Observability | Categorical Prometheus labels only plus committed Grafana dashboard and Prometheus alert rules under `backend/observability/` (no tickets, Redis keys, chat bodies, storage keys, answers, or precise coordinates as label values) |
| API handoff | Consumers follow OpenAPI + `contracts/casual-ranked-openapi.md`; one-time WS tickets; same-origin BFF for session/CSRF |

### Deferred frontend / full-stack release gates

No `client/` changes are required to close this backend milestone. The following remain **out of scope** until a frontend task set:

1. Next.js play / matches / competitive routes and BFF Route Handlers
2. Localized `en`/`ar` message catalogs and Arabic RTL layout verification
3. Narrow Client Components for map, WebSocket, timer, uploads, chat
4. Vitest / Testing Library unit coverage for play/party/matchplay UI
5. Playwright multi-context 1v1/2v2/4v4, chat/image, spectating, competitive flows
6. Frontend merge gates: `pnpm test`, `pnpm lint`, `pnpm typecheck`, `pnpm build`
7. Bundle budget measurement for the play shell (≤70 KB gzip excl. Maps SDK)
8. Accessibility keyboard/focus/live-region browser evidence
9. Staged flag enablement UX (disabled CTAs, service-unavailable image controls)
10. End-to-end product acceptance scenarios 1–7 in quickstart as **browser** evidence

### Residual backend verification risk (honest)

- Local Windows cannot run `-race` without cgo; race package gate is Linux CI.
- Shared Neon/Upstash used for opportunistic integration re-runs produced package-level failures; authoritative integration is CI service containers, not shared cloud DBs.
- Docker Desktop was unavailable on the polish workstation, so a clean local Postgres/Redis re-validation was not performed here.
- Live multi-client API/WebSocket soak and R2 image path were not re-run as manual ops on this pass; unit/handler coverage and CI remain the automated bar.

### Constitution note for this milestone

Local unit/handler, vet, lint, OpenAPI, and migration checks are recorded separately from the still-required disposable-service integration and Linux race gates. The unchecked tasks in `tasks.md` remain merge blockers until CI or an equivalent clean environment supplies that evidence. Full-stack constitution UX/localization/browser gates are **explicitly deferred** with the tasking scope override above and are not claimed as pass for release.
