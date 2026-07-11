# Research: Matchmaking And Ranked Foundations

## Redis Queue Representation

**Decision**: Use a versioned sorted set per competitive mode for FIFO ordering, a leased per-player hash as the cross-mode uniqueness pointer, and short-lived claim records for pairs being finalized. Suggested keys are `matchmaking:v1:queue:{mode}`, `matchmaking:v1:player:{user_id}`, `matchmaking:v1:claim:{claim_id}`, and `matchmaking:v1:claims`.

**Rationale**: Sorted sets provide ordered candidate selection without durable queue writes. The per-player pointer prevents duplicate entries across modes. Separate claims preserve enough state to reconcile a crash between Redis selection and PostgreSQL commit.

**Alternatives considered**:

- PostgreSQL queue rows: rejected because approved architecture assigns ephemeral matchmaking coordination to Redis and durable writes only after pairing.
- Redis Streams: rejected because consumer delivery does not solve per-player uniqueness, FIFO compatibility, or leave/claim races.
- One `SETNX` pointer followed by a separate sorted-set write: rejected because partial failures can leave contradictory state.

## Atomic Queue Commands

**Decision**: Implement join, leave, lease renewal/status, pair claim, claim finalization, and claim release as small atomic Redis scripts with strict key/version checks.

**Rationale**: Queue correctness spans multiple keys. Atomic scripts give one linearization point for duplicate joins, leave-versus-claim, stale-member removal, and two-worker competition. Existing repository code already uses Redis scripting for atomic rate-limit behavior.

**Alternatives considered**:

- `WATCH` plus pipelines: rejected because retry behavior and multi-key state-machine reasoning are more complex.
- Distributed lock alone: rejected because lock expiry or incorrect release can permit duplicate claims; locks may reduce contention but cannot be the correctness boundary.
- Copying the current room lock: rejected because its constant value and unconditional delete can release a newer owner's lock after TTL rollover.

## Lease And Expiry Semantics

**Decision**: A searching entry uses a renewable 30-second lease. Status polling renews the lease while preserving the original enqueue timestamp and FIFO score. The client polls every 2 seconds while searching; Redis scripts prune missing, expired, or mismatched candidates lazily.

**Rationale**: Active players can wait indefinitely without abandoned tabs staying matchable. Keeping the original score prevents polling or retrying from changing priority.

**Alternatives considered**:

- Fixed maximum wait TTL: rejected because a legitimate long wait would unexpectedly remove an active player.
- No TTL: rejected because abandoned entries would require a mandatory worker before Phase 12.
- Heartbeat that rewrites queue score: rejected because it allows priority manipulation.

## Match Formation Trigger

**Decision**: For this phase, run one bounded formation/reconciliation attempt after successful join and during searching status checks. The second compatible join can therefore synchronously claim the oldest pair. A Phase 12 worker may later call the same service operation for proactive sweeping and throughput.

**Rationale**: The phase depends on workers conceptually but the documented delivery order places the full worker process in Phase 12. Request-driven triggering meets the three-second assignment goal with the active polling flow and avoids an unmanaged API-process goroutine.

**Alternatives considered**:

- New `cmd/worker` now: rejected as premature expansion into Phase 12; the service boundary remains worker-ready.
- Permanent API-process ticker: rejected because multiple replicas complicate ownership, shutdown, and operational expectations.
- Match only on join: rejected because a claim interrupted by a transient failure would not recover unless another player joined.

## Redis/PostgreSQL Consistency

**Decision**: Treat Redis as authoritative only while searching/claimed and PostgreSQL as authoritative immediately after a durable match commits. Every join/status operation first checks for an active durable assignment. Formation uses a unique claim/formation identifier, and recovery asks PostgreSQL whether that identifier committed before finalizing, requeueing with original timestamps, or discarding the claim.

**Rationale**: Redis and PostgreSQL cannot share a transaction. Durable-first status precedence makes a crash after database commit safe even if Redis finalization fails. Unique formation identity makes retries idempotent.

**Alternatives considered**:

- Best-effort dual writes with compensating deletes: rejected because crash windows can lose a committed match or duplicate participants.
- Redis-only match result: rejected because reconnects, history, future ratings, and dispute diagnosis require durable facts.
- Durable `forming` row before a complete destination exists: rejected because transaction rollback can represent formation until the complete match is ready.

## Durable Match Model

**Decision**: Add `matches` and `match_players`. A match links one ranked game and holds lifecycle timestamps. Each participant row stores both `user_id` and `game_player_id`, plus participant lifecycle status. Add a partial unique index on `match_players.user_id` for `assigned` or `active` states.

**Rationale**: The older ERD's `game_player_id` alone cannot enforce one active ranked assignment per registered user. The intentional `user_id` duplication enables a database-level concurrency guard while `game_player_id` remains the immutable gameplay snapshot/outcome link.

**Alternatives considered**:

- Only `(match_id, game_player_id)`: rejected because active uniqueness would require a cross-table join that cannot be expressed as a normal unique index.
- Player-one/player-two columns on `matches`: rejected because it is less extensible and duplicates participant lifecycle fields.
- Deferred trigger enforcing exactly two rows: rejected because atomic service creation plus integration tests are simpler; the schema still enforces uniqueness and active assignment.

## Ranked Game Destination

**Decision**: Create a direct `ranked` game, two game-player snapshots, selected rounds, the match, and participant links as one PostgreSQL transaction after locations are successfully selected. Start the first round with a short server-controlled countdown, and generalize existing multiplayer game behavior from `private_room` to both `private_room` and `ranked` modes.

**Rationale**: A match is final only when both players can recover one playable destination. Direct games avoid fake room codes, hosts, expiry, and lobby semantics. Reusing the multiplayer guess/deadline path preserves server-authoritative scoring and waits for both guesses or the deadline.

**Alternatives considered**:

- Create a hidden private room: rejected because host/code/visibility concepts are misleading for ranked pairing and add authorization edge cases.
- Create a pending game and let the first player start it: rejected because arrival timing would control both players' competitive start.
- Defer gameplay integration entirely: rejected because durable outcomes and future rating foundations need a real linked game lifecycle.

## Transaction And Lifecycle Boundaries

**Decision**: Sort user IDs and lock both active user rows in stable order during formation; recheck account and active-assignment eligibility; then create the complete bundle. Match and participant transitions to `active`, `completed`, `cancelled`, or `failed_to_start` occur in the same transaction as their corresponding game lifecycle change through a narrow transaction-aware games seam.

**Rationale**: Stable lock ordering avoids deadlocks, the partial unique index is the final concurrency guard, and atomic lifecycle updates prevent future rating readers from seeing a completed game with a nonterminal match.

**Alternatives considered**:

- Post-commit lifecycle hook only: rejected because it introduces an observable consistency gap and retry backlog.
- Database trigger coupling games to matches: rejected because hidden cross-feature behavior is harder to test and maintain.
- Redis locks as final eligibility guard: rejected because Redis state is ephemeral and cannot protect durable invariants.

## Public HTTP Contract

**Decision**: Keep `POST /matchmaking/queue`, `DELETE /matchmaking/queue`, and `GET /matchmaking/status`, but replace stale guest/quick-play schemas. All endpoints require a registered access cookie; unsafe methods require CSRF. Public states are `not_queued`, `searching`, `matched`, and `temporarily_unavailable`. POST returns `202`, DELETE is idempotent with `204`, and GET returns `200`.

**Rationale**: Existing route names are stable, while current OpenAPI security and payloads conflict with the new specification. The small state union lets clients recover after races without exposing other queued players.

**Alternatives considered**:

- Separate 201/200 for first/repeated joins: rejected because clients do not need to distinguish and both requests mean search accepted.
- Return room objects/opponent details: rejected because ranked uses a direct game and status must be privacy-safe.
- Add WebSocket matchmaking events: rejected because bounded polling meets the two-second freshness target with much less lifecycle complexity.

## Authentication, Eligibility, CSRF, And Rate Limits

**Decision**: Use `RequireAuth` on all matchmaking routes and existing global CSRF enforcement for POST/DELETE. Join and formation additionally query `users.status = active` because token resolution alone does not recheck disabled state. Add per-user rate-limit keys: a low command/churn budget and a polling-safe status budget.

**Rationale**: Ranked identity must be durable. Middleware currently proves a registered session but not current account eligibility. Per-user limits avoid shared-IP penalties and token-rotation bypasses.

**Alternatives considered**:

- Guest access from the old placeholder: rejected by the specification.
- IP-only limits: rejected because homes, schools, and mobile gateways can share addresses.
- Raw access-cookie rate-limit keys: rejected because they store sensitive token material in key names and vary across sessions.

## Frontend Data Flow

**Decision**: Render initial status in a localized Server Component through `client/lib/api/matchmaking.ts`. Use Server Actions for join and leave so existing cookie and CSRF forwarding is reused. While searching, a narrow Client Component polls a same-origin Route Handler every two seconds with no overlapping requests, stops on terminal states, and backs off on rate limits or failures.

**Rationale**: Canonical state stays on the backend, server-only credentials remain protected, and only time-sensitive interaction enters the client bundle. The Route Handler is a better semantic fit for repeated GET polling than repeatedly invoking mutation-oriented Server Actions or refreshing the entire RSC tree.

**Alternatives considered**:

- Zustand server-state store: rejected by project constitution and because it would create a second authority.
- Browser-direct backend requests: rejected because it needs a new public base URL/proxy and duplicates cookie/CORS/error behavior.
- Full-page `router.refresh()` polling: rejected because it rerenders more of the route than needed.

## Observability And Failure Behavior

**Decision**: Add bounded-label metrics for commands, status states, queue depth by approved mode, claim/formation/recovery outcomes, formation latency, stale cleanup, rate limiting, Redis failures, and database failures. Logs include safe claim/match identifiers and outcomes but omit raw player IDs, tokens, Redis keys, opponent identity, hidden locations, and guesses. Redis outage fails queue commands closed; status still returns a durable match when present, otherwise a recoverable unavailable outcome.

**Rationale**: Match formation races require diagnosis, but high-cardinality or sensitive labels would make telemetry unsafe and expensive. PostgreSQL/Redis already participate in readiness, so no new dependency is introduced.

**Alternatives considered**:

- Claiming distributed tracing/Sentry coverage now: rejected because current observability adapters do not provide production tracing/error capture; Phase 12 owns that hardening.
- Returning `not_queued` on Redis failure: rejected because it can cause duplicate join attempts and misrepresent uncertain state.
