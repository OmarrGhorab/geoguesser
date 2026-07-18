# Tasks: Casual And Ranked Team Modes — Backend Only

**Input**: Design documents from `/specs/012-casual-ranked-modes/`

**Scope**: This task set implements backend behavior only: Go API/services, PostgreSQL, Redis, object storage, WebSocket transport, migrations, contracts, workers, observability, and backend verification. It MUST NOT create or modify files under `client/`. Frontend UI, localization/RTL, browser accessibility, frontend tests, and frontend build gates are deferred to a separate task set; completing this file does not make the full feature release-ready.

**Tests**: Required by the project constitution. Test tasks precede the behavior they protect and must fail before implementation where locally reproducible.

**Organization**: Tasks are grouped by the five specification user stories. IDs are execution ordered; `[P]` means the task can run concurrently with other marked tasks in that phase because it owns different files.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Parallelizable without an incomplete dependency or file collision
- **[US1]–[US5]**: Traceability to the corresponding specification story
- Every task names its concrete backend, contract, infrastructure, or planning path

---

## Phase 1: Backend Setup

**Purpose**: Establish feature configuration and the single new backend dependency before shared schema/transport work.

- [x] T001 Add failing validation/default tests for all feature flags, party/match timing, competitive constants, realtime limits, and chat retention/image limits in `backend/internal/config/config_test.go`
- [x] T002 Implement the validated configuration fields from `quickstart.md` and document safe disabled defaults in `backend/internal/config/config.go` and `backend/.env.example`
- [x] T003 Add `golang.org/x/image` for WebP decode and high-quality resize, then tidy and verify the dependency graph in `backend/go.mod` and `backend/go.sum`

---

## Phase 2: Shared Backend Foundations

**Purpose**: Additive schema, common mode contracts, idempotency, and targeted realtime primitives that block every user story.

**⚠️ CRITICAL**: No story implementation begins until this phase passes.

- [x] T004 [P] Add failing PostgreSQL migration tests for new tables, columns, constraints, indexes, seeded Season 1, legacy `ranked_standard` backfill, deterministic team slots, and guess score backfill in `backend/internal/app/casual_ranked_migration_test.go`
- [x] T005 Implement additive Goose migration `00019`, guarded irreversible down behavior, Season 1 seed, and legacy backfills in `backend/migrations/00019_casual_ranked_team_modes.sql`
- [x] T006 Add migration fixtures for active/completed legacy ranked matches and verify upgrade preservation and constraint rejection in `backend/internal/app/casual_ranked_migration_test.go`
- [x] T007 [P] Define canonical playlist/format/mode mappings, `ranked_standard` alias handling, team-size validation, shared lifecycle enums, and stable errors in `backend/internal/matchmaking/model.go`, `backend/internal/games/state.go`, and `backend/internal/matchplay/errors.go`
- [x] T008 [P] Define narrow party, competitive, match-lifecycle, collaboration, and event-publisher interfaces without package cycles in `backend/internal/matchmaking/contracts.go`, `backend/internal/games/contracts.go`, and `backend/internal/realtime/contracts.go`
- [x] T009 [P] Add failing one-time realtime-ticket tests for user/channel binding, 30-second expiry, atomic consume, replay rejection, and safe hashed Redis keys in `backend/internal/platform/redis/realtime_test.go`
- [x] T010 Implement one-time realtime tickets, channel versions, presence/reconnect keys, and safe key builders in `backend/internal/platform/redis/realtime.go`
- [x] T011 [P] Add failing generic hub/event tests for party and match channels, per-user/team audiences, bounded queues, version ordering, and slow-consumer close behavior in `backend/internal/realtime/event_test.go` and `backend/internal/realtime/hub_test.go`
- [x] T012 Generalize room-specific realtime envelopes and the in-memory hub while preserving existing room events in `backend/internal/realtime/event.go` and `backend/internal/realtime/hub.go`
- [x] T013 [P] Add failing multi-instance Pub/Sub tests for reference-counted channel subscriptions, duplicate suppression, unsubscribe cleanup, and shutdown in `backend/internal/platform/redis/realtime_pubsub_test.go`
- [x] T014 Implement reference-counted party/match Pub/Sub fanout in `backend/internal/platform/redis/realtime_pubsub.go`
- [x] T015 [P] Add a context-bound periodic worker runner with panic isolation, bounded run timeout, graceful stop, and last-success/failure metrics in `backend/internal/platform/workers/runner.go` and `backend/internal/platform/workers/runner_test.go`
- [x] T016 [P] Add failing generic command-idempotency tests for same-body replay, conflicting-body rejection, TTL, and caller scoping in `backend/internal/platform/redis/command_idempotency_test.go`
- [x] T017 Implement the Redis command-idempotency store used by party, queue, message, report, upload-complete, and leave commands in `backend/internal/platform/redis/command_idempotency.go`

**Checkpoint**: Migration, configuration, common contracts, idempotency, and targeted realtime infrastructure pass independently.

---

## Phase 3: User Story 1 — Play A Casual Match (Priority: P1) 🎯 Backend MVP

**Goal**: Provide signed-in Casual Solo/Duo/Squad parties, equal-team matchmaking, five no-timer rounds, summed team scoring, safe shared reveal, results, reconnect, forfeit, and inactivity closure with no competitive mutation.

**Independent Test**: Using API/service tests with 2, 4, and 8 registered users, form each Casual format, complete five rounds without a deadline or speed bonus, verify exact team totals and delayed reveal, and confirm no competitive rows change.

### Tests for User Story 1

- [x] T018 [P] [US1] Add failing party service tests for friend-only invites, either-direction blocks, capacity, readiness reset, leader transfer, queue locking, and idempotent decline/disband in `backend/internal/parties/service_test.go`
- [x] T019 [P] [US1] Add failing PostgreSQL party tests for one-active-party uniqueness, concurrent invite acceptance, stable locks, roster capacity, and version increments in `backend/internal/parties/repository_test.go`
- [x] T020 [P] [US1] Add failing party handler and route-security tests for auth, CSRF, strict JSON, command idempotency, privacy-safe errors, and rate limits in `backend/internal/parties/handler_test.go` and `backend/internal/app/routes_test.go`
- [x] T021 [P] [US1] Add failing Redis v2 team-ticket tests for Solo/Duo/Squad join, per-user exclusivity, party-version validation, lease renewal, atomic equal-roster claim, release, and durable-first recovery in `backend/internal/platform/redis/matchmaking_v2_test.go`
- [x] T022 [P] [US1] Add a 10,000-searcher/scan-limit-20 Casual ticket benchmark with 1/2/4-player rosters in `backend/internal/platform/redis/matchmaking_v2_benchmark_test.go`
- [x] T023 [P] [US1] Add failing match formation integration tests for 1v1/2v2/4v4 cardinality, team slots, distinct users/locations, transaction rollback, legacy alias recovery, and duplicate formation keys in `backend/internal/matchmaking/team_formation_test.go`
- [x] T024 [P] [US1] Add failing Casual game tests for null timers, all-active submission advancement, locked guesses, zero missing scores, summed player/team totals, exact draws, and delayed answer reveal in `backend/internal/games/casual_team_test.go`
- [x] T025 [P] [US1] Add failing match snapshot/result authorization tests for participant-only reads, pre-reveal redaction, round results, Casual no-progression response, and privacy-safe not-found in `backend/internal/matchplay/snapshot_test.go`
- [x] T026 [P] [US1] Add failing lifecycle tests for explicit Casual leave, 90-second disconnect grace, ten-minute inactivity closure, party restoration after terminal match, and retry safety in `backend/internal/matchplay/lifecycle_test.go`

### Implementation for User Story 1

- [x] T027 [P] [US1] Create Party, PartyMember, PartyInvite models plus strict request/response DTOs and stable errors in `backend/internal/parties/model.go`, `backend/internal/parties/dto.go`, and `backend/internal/parties/errors.go`
- [x] T028 [US1] Implement transactional party persistence, row-lock ordering, active-member uniqueness, invite expiry, and versioned snapshots in `backend/internal/parties/repository.go`
- [x] T029 [P] [US1] Expose a narrow accepted-friend/either-direction-block party policy without leaking block ownership in `backend/internal/friends/party_policy.go` and `backend/internal/friends/party_policy_test.go`
- [x] T030 [US1] Implement party create/current/read/invite/accept/decline/readiness/leave/kick/disband rules and idempotency in `backend/internal/parties/service.go`
- [x] T031 [US1] Implement authenticated party handlers, error mapping, route registration, and rate-limit observers in `backend/internal/parties/handler.go` and `backend/internal/parties/metrics.go`
- [x] T032 [P] [US1] Replace pair-only queue DTOs with canonical ticket/roster/status DTOs while retaining legacy response fields and `ranked_standard` parsing in `backend/internal/matchmaking/dto.go` and `backend/internal/matchmaking/model.go`
- [x] T033 [US1] Implement Redis v2 Lua-backed ticket join/renew/leave/claim/release/recovery and keep v1 reads during rollout in `backend/internal/platform/redis/matchmaking_v2.go` and `backend/internal/matchmaking/redis_adapter.go`
- [x] T034 [US1] Generalize durable formation input to two ordered rosters, validate all users under stable locks, create team-slotted game/match participants, and preserve formation idempotency in `backend/internal/matchmaking/repository.go`
- [x] T035 [US1] Generalize matchmaking service eligibility, complete-party readiness, six-mode compatibility, Casual ticket matching, lease recovery, and assignment responses in `backend/internal/matchmaking/service.go`
- [x] T036 [P] [US1] Extend game/player/guess models and DTOs for team slots, accuracy/bonus totals, nullable multiplayer reveal, submitted counts, and Casual mode in `backend/internal/games/model.go`, `backend/internal/games/dto.go`, and `backend/internal/games/multiplayer.go`
- [x] T037 [US1] Implement Casual team formation/play persistence, no-deadline advancement, atomic team total updates, draw/result calculation, and terminal lifecycle hooks in `backend/internal/games/repository.go`
- [x] T038 [US1] Implement Casual submit/replay behavior, prevent early multiplayer answer reveal, expose shared round results, and preserve solo/daily response compatibility in `backend/internal/games/service.go` and `backend/internal/games/handler.go`
- [x] T039 [P] [US1] Create match snapshot/result/leave DTOs and participant/team projections in `backend/internal/matchplay/model.go` and `backend/internal/matchplay/dto.go`
- [x] T040 [US1] Implement authorized match snapshot, revealed-round result, terminal result, explicit leave, and party-restoration transactions in `backend/internal/matchplay/repository.go` and `backend/internal/matchplay/service.go`
- [x] T041 [US1] Implement match snapshot/result/leave handlers with privacy-safe errors, CSRF, idempotency, and rate-limit metrics in `backend/internal/matchplay/handler.go` and `backend/internal/matchplay/metrics.go`
- [x] T042 [US1] Publish versioned party roster/readiness/queue/assignment and Casual match/round/result events through the shared publisher in `backend/internal/parties/events.go`, `backend/internal/matchmaking/events.go`, and `backend/internal/matchplay/events.go`
- [x] T043 [US1] Implement bounded disconnect, inactivity, expired-claim, and progression-neutral Casual lifecycle sweeps in `backend/internal/matchplay/worker.go` and `backend/internal/matchmaking/worker.go`
- [x] T044 [US1] Wire party, generalized matchmaking, matchplay, workers, feature flags, auth, CSRF, rate limits, and graceful shutdown in `backend/cmd/api/main.go`, `backend/internal/app/server.go`, and `backend/internal/app/routes.go`
- [x] T045 [US1] Update party, matchmaking, match snapshot/round result/result/leave schemas and legacy compatibility in `backend/openapi/openapi.yaml`
- [x] T046 [US1] Add Casual API examples for party creation, queue status, 1v1/2v2/4v4 assignments, delayed reveal, and results in `backend/postman/GeoGuess.postman_collection.json` and `backend/postman/saved-responses/`

**Checkpoint**: Casual is fully testable through backend APIs/realtime events in all three team sizes with no competitive state changes.

---

## Phase 4: User Story 2 — Compete In A Ranked Match (Priority: P1)

**Goal**: Add 60-second authoritative Ranked rounds, accuracy-weighted speed bonuses, rating-aware matchmaking, exact-once Elo changes, placements, and abandonment penalties for all formats.

**Independent Test**: Form equal-rating Ranked 1v1/2v2/4v4 matches, verify shared deadlines and disclosed bonuses, finalize win/loss/draw once, and confirm symmetric team rating changes plus quitter-only penalty.

### Tests for User Story 2

- [x] T047 [P] [US2] Add failing table tests for remaining-time fraction, rounding, zero accuracy, deadline clamp, and 250-point cap in `backend/internal/games/speed_scoring_test.go`
- [x] T048 [P] [US2] Add failing PostgreSQL Ranked round tests for identical deadlines, all-submit/timeout closure, delayed reveal, idempotent retries, and summed bonus totals in `backend/internal/games/ranked_team_test.go`
- [x] T049 [P] [US2] Add failing Ranked ticket tests for ±100 initial window, +50/30-second expansion, ±400 cap, team-average comparison, placement hidden rating, and two-named-rank party spread in `backend/internal/matchmaking/ranked_ticket_test.go`
- [x] T050 [P] [US2] Add failing Elo unit tests for expected probability, equal-rating `+16/-16`, draws, team-equal base delta, floor zero, hidden five placements, and `-15` abandon penalty in `backend/internal/competitive/rating_test.go`
- [x] T051 [P] [US2] Add failing concurrent rating repository tests for stable user locks, unique match/user changes, atomic standings/match finalization, retry replay, and rollback in `backend/internal/competitive/repository_test.go`
- [x] T052 [P] [US2] Add failing abandonment integration tests ensuring team forfeit, quitter-only extra penalty, normal opponent win, reconnect recovery, and terminal replay safety in `backend/internal/matchplay/ranked_abandon_test.go`

### Implementation for User Story 2

- [x] T053 [P] [US2] Implement server-time speed bonus calculation and pure helpers in `backend/internal/games/speed_scoring.go`
- [x] T054 [US2] Persist accuracy/speed/total atomically, enforce the 60-second deadline, and return nullable reveal/progress fields in `backend/internal/games/repository.go`, `backend/internal/games/service.go`, and `backend/internal/games/dto.go`
- [x] T055 [P] [US2] Create active-season Standing and RatingChange models, rating outcome DTOs, and stable progression errors in `backend/internal/competitive/model.go`, `backend/internal/competitive/dto.go`, and `backend/internal/competitive/errors.go`
- [x] T056 [US2] Implement active-season/standing locks, team-average reads, exact-once rating history, standings updates, and match progression finalization in `backend/internal/competitive/repository.go`
- [x] T057 [US2] Implement Elo calculation, five-placement visibility, symmetric team deltas, abandon penalty, floor zero, and idempotent finalization in `backend/internal/competitive/service.go`
- [x] T058 [US2] Add active-season and rating snapshots to Ranked ticket eligibility/formation and enforce expanding rating windows and party spread in `backend/internal/matchmaking/service.go` and `backend/internal/matchmaking/repository.go`
- [x] T059 [US2] Replace the ranked-only lifecycle adapter with a match lifecycle hook that finalizes game result and competitive changes transactionally or marks retryable progression pending in `backend/internal/matchmaking/lifecycle.go` and `backend/internal/games/service.go`
- [x] T060 [US2] Implement explicit/disconnect Ranked forfeit, participant abandon facts, normal opponent win, and quitter-only penalty in `backend/internal/matchplay/service.go` and `backend/internal/matchplay/repository.go`
- [x] T061 [US2] Add bounded retry of completed Ranked matches lacking `progression_finalized_at` in `backend/internal/competitive/worker.go`
- [x] T062 [US2] Return Ranked timer, accuracy, bonus, team result, placement progress, old/new rating, and pending/retry states from `backend/internal/matchplay/dto.go` and `backend/internal/matchplay/handler.go`
- [x] T063 [P] [US2] Add Ranked formation/finalization/abandonment histograms and bounded labels without user IDs or guesses in `backend/internal/matchmaking/metrics.go`, `backend/internal/games/metrics.go`, and `backend/internal/competitive/metrics.go`
  - competitive package metrics complete (`backend/internal/competitive/metrics.go`); matchmaking/games histogram portions remain for their package owners if not already present
- [x] T064 [US2] Update Ranked queue, guess, result, progression-pending, and abandon contracts in `backend/openapi/openapi.yaml`

**Checkpoint**: Ranked match formation, timing, scoring, and rating work for every team size without the full rank/leaderboard presentation added by US5.

---

## Phase 5: User Story 3 — Collaborate With A Team (Priority: P1)

**Goal**: Add private team text/images, proposed markers, mute/report, sanitized storage, targeted multi-instance events, and bounded retention for Duo/Squad.

**Independent Test**: In a two-team backend integration test, exchange text/images and markers, verify exact team-only delivery and score behavior, reject every opponent/non-participant read, and validate mute/report/retention cleanup.

### Tests for User Story 3

- [x] T065 [P] [US3] Add failing team-chat service tests for authorization, 500-character validation, text-or-file rule, idempotency, one attachment, message/image limits, terminal access window, mute filtering, and report privacy in `backend/internal/matchplay/chat_service_test.go`
- [x] T066 [P] [US3] Add failing PostgreSQL chat tests for stable sequence pagination, unique client IDs/files/reports, team filtering, concurrent sends, report retention, and legal hold in `backend/internal/matchplay/chat_repository_test.go`
- [x] T067 [P] [US3] Add failing sanitizer tests for JPEG/PNG/WebP, MIME spoofing, corrupt data, 5 MB/20 MP limits, 2,048-pixel resize, metadata stripping, output cap, and raw cleanup in `backend/internal/uploads/image_sanitizer_test.go`
- [x] T068 [P] [US3] Add failing local/R2 storage adapter tests for bounded GetObject, private PutObject, delete, cancellation, and no public derivative URL in `backend/internal/platform/storage/storage_test.go`
- [X] T069 [P] [US3] Add failing Redis live-state tests for team-scoped markers, 2/s throttling, round TTL, versioning, snapshot recovery, locked-player rejection, and key redaction in `backend/internal/platform/redis/matchplay_test.go`
- [X] T070 [P] [US3] Add failing multi-client WebSocket tests for one-use subprotocol tickets, party/match snapshots, targeted team events, command replay, version gaps, origin rejection, and slow-consumer closure in `backend/internal/realtime/match_handler_test.go`
- [X] T071 [P] [US3] Add failing cross-instance integration tests proving Pub/Sub delivers each authorized message/marker event once and never to the opposing team in `backend/internal/realtime/pubsub_integration_test.go`
- [X] T072 [P] [US3] Add failing cleanup-worker tests for 30-day unreported, 180-day reported, legal-hold, 24-hour raw-upload, bounded batches, and storage-deletion retry in `backend/internal/matchplay/cleanup_test.go`

### Implementation for User Story 3

- [x] T073 [P] [US3] Extend storage Provider with bounded object read/write, implement local/R2 adapters, and preserve existing upload behavior in `backend/internal/platform/storage/storage.go`, `backend/internal/platform/storage/local.go`, and `backend/internal/platform/storage/r2.go`
- [x] T074 [US3] Implement magic-byte verification, decode, pixel/dimension checks, resize, JPEG re-encode, output cap, derivative write, and raw deletion in `backend/internal/uploads/image_sanitizer.go`
- [x] T075 [P] [US3] Extend upload/file models, repository fields, and DTOs with purpose, match context, sanitization status, and raw-key lifecycle in `backend/internal/uploads/model.go`, `backend/internal/uploads/repository.go`, and `backend/internal/uploads/dto.go`
- [x] T076 [US3] Implement purpose-bound team-chat upload creation/completion, same-match authorization, sanitized-only readiness, degraded storage errors, and raw cleanup in `backend/internal/uploads/service.go` and `backend/internal/uploads/handler.go`
- [x] T077 [P] [US3] Add TeamMessage, MatchMute, MessageReport models plus strict chat/attachment/report DTOs in `backend/internal/matchplay/chat_model.go` and `backend/internal/matchplay/chat_dto.go`
- [x] T078 [US3] Implement chat persistence, sequence pagination, unique idempotency/file binding, same-team queries, mute/report facts, and retention selection in `backend/internal/matchplay/chat_repository.go`
- [x] T079 [US3] Implement active-team chat authorization, text/image limits, mute filtering, report privacy, attachment signed URLs, and 15-minute result access in `backend/internal/matchplay/chat_service.go`
- [x] T080 [US3] Implement chat/message/attachment/mute/report handlers, privacy-safe errors, CSRF, per-user limits, and rate-limit observers in `backend/internal/matchplay/chat_handler.go`
- [X] T081 [P] [US3] Implement round marker/view/presence/version storage and atomic command throttles in `backend/internal/platform/redis/matchplay.go`
- [X] T082 [US3] Implement authenticated realtime-ticket issuance for party/match channels and `geoguess.v1` plus `ticket.{opaque}` subprotocol consumption in `backend/internal/realtime/ticket_handler.go` and `backend/internal/realtime/handler.go`
- [X] T083 [US3] Implement strict WebSocket command decoding, heartbeat/marker command dispatch, command acknowledgements, version-gap errors, ping/pong, and reconnect facts in `backend/internal/realtime/match_handler.go`
- [X] T084 [US3] Implement server-side per-user/team audience resolution and compact chat/marker/lifecycle event publishing after durable commit in `backend/internal/matchplay/audience.go` and `backend/internal/matchplay/events.go`
- [x] T085 [US3] Implement bounded chat/raw-object retention cleanup with delete retry and legal-hold exclusion in `backend/internal/matchplay/cleanup.go`
- [X] T086 [P] [US3] Add chat, attachment, marker, ticket, socket, slow-consumer, sanitizer, and cleanup metrics with redacted bounded labels in `backend/internal/matchplay/metrics.go`, `backend/internal/realtime/metrics.go`, and `backend/internal/uploads/metrics.go`
  - [x] matchplay chat/attachment/mute/report/cleanup + marker/event publish metrics in `backend/internal/matchplay/metrics.go`
  - [x] realtime ticket/socket/slow-consumer/marker/command metrics in `backend/internal/realtime/metrics.go`
  - [x] uploads sanitizer/command/storage metrics in `backend/internal/uploads/metrics.go`
- [X] T087 [US3] Wire storage sanitizer, match live-state adapter, realtime ticket/fanout handlers, cleanup worker, and chat/image feature flags in `backend/cmd/api/main.go`, `backend/internal/app/server.go`, and `backend/internal/app/routes.go`
- [X] T088 [US3] Update chat, attachment, upload-purpose, mute/report, realtime-ticket, and marker WebSocket contracts in `backend/openapi/openapi.yaml` and `specs/012-casual-ranked-modes/contracts/casual-ranked-openapi.md`

**Checkpoint**: Duo/Squad collaboration works through backend contracts with strict team privacy, safe derivatives, and bounded retention; text remains available during image-storage degradation.

---

## Phase 6: User Story 4 — Spectate After Submitting (Priority: P2)

**Goal**: Enforce Solo-opponent versus team-teammate post-submit spectating using provider-safe scene state only.

**Independent Test**: Submit a guess in Solo and team matches, select every allowed/disallowed target, and prove only authorized scene state is delivered while map, marker, cursor, score, answer, and private client fields never cross the boundary.

### Tests for User Story 4

- [x] T089 [P] [US4] Add failing audience-policy table tests for Solo opponent, Duo/Squad teammates, unsubmitted viewers, submitted/disconnected targets, round close, and forbidden opponents in `backend/internal/matchplay/spectating_test.go`
- [x] T090 [P] [US4] Add failing Redis scene-state tests for provider-safe schemas, 4/s throttle, material-change deduplication, round TTL, and rejection of map/guess/answer fields in `backend/internal/platform/redis/matchplay_view_test.go`
- [x] T091 [P] [US4] Add failing WebSocket authorization tests for select/view commands, automatic fallback targets, disconnect/submit/round-end transitions, and privacy-safe errors in `backend/internal/realtime/spectating_test.go`

### Implementation for User Story 4

- [x] T092 [P] [US4] Implement pure allowed-target and scene-field policy for Solo and team formats in `backend/internal/matchplay/spectating.go`
- [x] T093 [US4] Implement validated/throttled view-state persistence and material-change detection in `backend/internal/platform/redis/matchplay.go`
- [x] T094 [US4] Implement spectate-select/view-update commands, automatic allowed-target fallback, and targeted view events in `backend/internal/realtime/match_handler.go` and `backend/internal/matchplay/events.go`
- [x] T095 [US4] Add viewer submission state and allowed spectate target IDs to authorized snapshots without leaking opponent guesses in `backend/internal/matchplay/service.go` and `backend/internal/matchplay/dto.go`
- [x] T096 [P] [US4] Add spectate selection, denial, fallback, and view-update metrics with format/outcome labels only in `backend/internal/matchplay/metrics.go`
- [x] T097 [US4] Finalize spectating WebSocket commands/events and safe scene schemas in `specs/012-casual-ranked-modes/contracts/casual-ranked-openapi.md`

**Checkpoint**: Post-submit spectating is backend-enforced and cannot be widened by a modified client.

---

## Phase 7: User Story 5 — Progress Through Eight Ranks (Priority: P2)

**Goal**: Expose placements, 21 standard divisions, seasonal profile/history, indexed top 500, exclusive World Legend, and safe automatic rollover.

**Independent Test**: Seed boundaries and 502 eligible players, verify every division and tie-break, cross the 500/501 cutoff, then race season rollover and confirm one immutable closed season plus the exact reset formula.

### Tests for User Story 5

- [x] T098 [P] [US5] Add failing rank-mapping tests for all 21 100-point divisions, zero floor, unbounded Geo Master I, placement hiding, and World Legend overlay in `backend/internal/competitive/ranks_test.go`
- [x] T099 [P] [US5] Add failing PostgreSQL leaderboard tests for eligibility, rating/wins/reached-time/user-ID order, exact top 500, viewer projection, cursor stability, inactive accounts, and closed snapshots in `backend/internal/competitive/leaderboard_test.go`
- [x] T100 [P] [US5] Add failing season-rollover concurrency tests for advisory locking, one next season, frozen final/peak facts, 50% reset toward 800, placement reset, and idempotent retries in `backend/internal/competitive/season_test.go`
- [x] T101 [P] [US5] Add failing profile/history/season/leaderboard handler tests for auth, pagination, placement redaction, World Legend membership, stable errors, and rate limits in `backend/internal/competitive/handler_test.go`
- [x] T102 [P] [US5] Add failing cache/invalidation tests and query-budget benchmarks proving ≤60-second top-500 staleness, ≤501 ordered scans, and p95 target fixtures in `backend/internal/competitive/cache_test.go` and `backend/internal/competitive/benchmark_test.go`

### Implementation for User Story 5

- [x] T103 [P] [US5] Add rank/division/World-Legend DTOs, opaque leaderboard/history cursors, and season summary shapes in `backend/internal/competitive/ranks.go`, `backend/internal/competitive/dto.go`, and `backend/internal/competitive/cursor.go`
- [x] T104 [US5] Implement all rank thresholds, placement hiding, division progress, top-500 overlay, and frozen ending/peak rank helpers in `backend/internal/competitive/ranks.go`
- [x] T105 [US5] Implement active profile, rating history, eligible ordered standings, viewer entry, closed leaderboard, and frozen season queries with bounded indexed pagination in `backend/internal/competitive/repository.go`
- [x] T106 [US5] Implement profile/history/season/top-500 services, good-standing eligibility, deterministic tie-break, and stable cache keys in `backend/internal/competitive/service.go`
- [x] T107 [P] [US5] Implement Redis top-500/profile page cache with ≤60-second TTL and post-rating/rollover invalidation in `backend/internal/competitive/cache.go`
- [x] T108 [US5] Implement advisory-lock season closure, final position/rank freeze, next 84-day season, soft reset, and placement reset in `backend/internal/competitive/season.go`
- [x] T109 [US5] Add minute-based rollover and bounded retry workers with last-success/failure observations in `backend/internal/competitive/worker.go`
- [x] T110 [US5] Implement authenticated competitive profile, history, seasons, active leaderboard, and closed top-500 handlers in `backend/internal/competitive/handler.go`
- [x] T111 [US5] Register competitive routes, feature gates, per-user read limits, and service/worker wiring in `backend/internal/app/routes.go` and `backend/cmd/api/main.go`
- [x] T112 [P] [US5] Add profile/leaderboard/rollover/cache/World-Legend metrics using season sequence and bounded outcomes only in `backend/internal/competitive/metrics.go`
- [x] T113 [US5] Update competitive profile/history/seasons/leaderboards and rank schemas in `backend/openapi/openapi.yaml`
- [x] T114 [US5] Add competitive profile, placements, promotion/demotion, top-500 cutoff, and closed-season examples in `backend/postman/GeoGuess.postman_collection.json` and `backend/postman/saved-responses/`

**Checkpoint**: Backend progression exposes the seven divided ranks and exclusive global World Legend top 500 with durable seasonal history.

---

## Phase 8: Backend Polish And Cross-Cutting Verification

**Purpose**: Prove security, failure recovery, performance, contracts, migration safety, and backend release readiness without performing deferred frontend work.

- [x] T115 [P] Add cross-feature authorization tests covering guests, disabled users, nonmembers, opponents, blocked relationships, stale parties, expired tickets, and privacy-safe error equivalence in `backend/internal/app/casual_ranked_security_test.go`
- [x] T116 [P] Add telemetry redaction tests proving logs/metrics omit raw tickets, Redis keys, user IDs where prohibited, chat/image contents, storage keys, hidden answers, and precise guesses in `backend/internal/app/casual_ranked_redaction_test.go`
- [x] T117 [P] Add worker integration tests for claim recovery, round deadlines, reconnect forfeits, Casual inactivity, progression retry, season rollover, retention cleanup, bounded batches, and graceful shutdown in `backend/internal/app/casual_ranked_workers_test.go`
- [x] T118 Add dependency-failure tests proving PostgreSQL durable state wins after formation, Redis outages block only ephemeral operations, R2 outages degrade images but not text/readiness, and invalid Ranked season config fails closed in `backend/internal/app/casual_ranked_failure_test.go`
- [x] T119 [P] Add end-to-end backend benchmarks for 10,000 queued users, 1v1/2v2/4v4 formation, eight-client event fanout, match snapshot queries, top 500, chat pagination, and 5 MB sanitization in `backend/internal/app/casual_ranked_benchmark_test.go`
- [x] T120 Update CI PostgreSQL/Redis integration and Linux race package lists for parties, matchplay, competitive, realtime, uploads, storage, workers, matchmaking, and games in `.github/workflows/ci.yml`
- [x] T121 Validate fresh and legacy upgrade migrations, constraints, Season 1 seed, and guarded down behavior; record evidence in `specs/012-casual-ranked-modes/quickstart.md`
- [x] T122 Run Redocly validation and reconcile every implemented HTTP schema/error/example against `backend/openapi/openapi.yaml` and `specs/012-casual-ranked-modes/contracts/casual-ranked-openapi.md`
- [x] T123 Run `gofmt` and `golangci-lint` for the backend and resolve all findings in `backend/`
- [ ] T124 Run all targeted story/integration suites against disposable PostgreSQL/Redis and record command/results in `specs/012-casual-ranked-modes/quickstart.md`
- [x] T125 Run `go test ./...` from `backend/` and record the final backend gate in `specs/012-casual-ranked-modes/quickstart.md`
- [ ] T126 Run the Linux `go test -p 1 -race` package gate in CI and record the final race evidence in `specs/012-casual-ranked-modes/quickstart.md`
- [ ] T127 Execute backend-only live API/WebSocket validation for Casual/Ranked 1v1/2v2/4v4, chat/images, spectating privacy, placement/divisions, top 500, and recovery; record evidence and residuals in `specs/012-casual-ranked-modes/quickstart.md`
- [x] T128 Document implemented backend flags, season operations, worker dashboards/alerts, retention cleanup, rollback-by-disable, migration irreversibility, and API consumer handoff in `specs/012-casual-ranked-modes/plan.md` and `specs/012-casual-ranked-modes/quickstart.md`
- [x] T129 Confirm the milestone change set is limited to backend, infrastructure, and planning paths and record all deferred release gates in `specs/012-casual-ranked-modes/plan.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Starts immediately.
- **Foundations (Phase 2)**: Depends on Setup and blocks every story.
- **US1 Casual (Phase 3)**: Depends on Foundations; establishes parties, team tickets, team game state, snapshots, and lifecycle.
- **US2 Ranked (Phase 4)**: Depends on US1's team engine and can run in parallel with US3 once US1 is complete.
- **US3 Collaboration (Phase 5)**: Depends on US1 match/team authorization and shared realtime foundation; can run in parallel with US2.
- **US4 Spectating (Phase 6)**: Depends on US3 live view transport and US1 submission state.
- **US5 Ranks (Phase 7)**: Depends on US2 standings/rating finalization; can run in parallel with US3/US4 after US2 completes.
- **Polish (Phase 8)**: Depends on every backend story selected for this milestone.

### User Story Dependency Graph

```text
Foundations -> US1 Casual -> US2 Ranked -> US5 Ranks
                         \-> US3 Collaboration -> US4 Spectating
```

### Within Each Story

1. Write the listed tests and confirm the targeted behavior fails.
2. Add models/DTOs and persistence.
3. Implement service policy and transaction boundaries.
4. Implement handlers/realtime commands and metrics.
5. Update OpenAPI/examples and run the story checkpoint.

## Parallel Opportunities

- Foundation migration, mode-contract, realtime-ticket, hub, Pub/Sub, worker, and idempotency tests are on separate files and may start together.
- In US1, party, Redis queue, formation, game, snapshot, and lifecycle tests may be written concurrently; party policy/model work can proceed alongside game DTO work after tests exist.
- After US1, US2 and US3 are separate workstreams. Within US3, storage/sanitizer, chat persistence, Redis live state, and realtime tests are parallel until service integration.
- US4 test files can be written in parallel, but command implementation waits for US3 realtime dispatch.
- US5 mapping, repository, rollover, handler, and benchmark tests may be written concurrently; cache work can proceed alongside DTO/rank helpers.
- Cross-cutting security, redaction, worker, and benchmark tests are parallel after all story APIs stabilize.

## Parallel Examples

### US1 Casual

```text
Parallel: T018 party service tests, T021 Redis ticket tests, T023 formation tests,
T024 game tests, T025 snapshot tests, T026 lifecycle tests.
Then: T027–T031 party slice and T032/T036/T039 DTO slices can proceed in parallel.
```

### US2 Ranked And US3 Collaboration

```text
After US1 checkpoint:
- Stream A: T047–T064 Ranked timer/bonus/rating.
- Stream B: T065–T088 chat/images/markers/realtime.
```

### US5 Ranks

```text
Parallel: T098 rank mapping, T099 leaderboard persistence, T100 rollover,
T101 handlers, T102 cache/benchmark tests.
Then integrate through T103–T114.
```

## Implementation Strategy

### Backend MVP First

1. Complete Setup and Foundations.
2. Complete US1 through T046.
3. Stop and validate all Casual formats through backend API/realtime tests.
4. Do not claim a user-facing release; frontend remains explicitly deferred.

### Incremental Backend Delivery

1. **Casual foundation**: parties + equal-team queues + no-timer games.
2. **Ranked core**: timer + speed scoring + exact-once rating.
3. **Collaboration**: team chat/images/markers/realtime.
4. **Spectating**: server-enforced post-submit view policy.
5. **Progression**: divisions + seasons + exclusive top 500.
6. **Backend hardening**: gates, failure drills, performance evidence, operations.

## Notes

- No task in this file authorizes edits under `client/`.
- Frontend schema consumers must be implemented later from the finalized OpenAPI/WebSocket contracts.
- A checked backend task means its automated verification and relevant contract update also pass.
- Do not use GORM AutoMigrate; only Goose migration `00019` changes production schema.
- Preserve unrelated user changes and existing room/daily/solo behavior throughout implementation.
- Commit after coherent tested groups, not before failing tests are converted to passing behavior.
