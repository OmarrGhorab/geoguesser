# Data Model: Matchmaking And Ranked Foundations

## Overview

Backend Phase 9 keeps active search coordination in Redis and writes PostgreSQL only after two compatible players are atomically claimed. PostgreSQL becomes authoritative once a complete ranked match and playable game destination commit. Redis state is disposable and recoverable from durable assignments.

## Durable Entities

### Ranked Match

Backed by new `matches` table.

Fields:

- `id`: UUID primary identity.
- `formation_key`: unique opaque identifier copied from the Redis claim; makes formation replay idempotent.
- `game_id`: required unique foreign key to `games.id`.
- `mode`: `ranked_standard` in this phase.
- `status`: `matched`, `active`, `completed`, `cancelled`, or `failed_to_start`.
- `matched_at`: time the complete assignment became durable.
- `started_at`: nullable competitive game start.
- `completed_at`: nullable successful terminal time.
- `closed_at`: nullable cancellation/failure terminal time.
- `failure_code`: nullable bounded internal reason code; never arbitrary user or provider text.
- `created_at`, `updated_at`: audit timestamps.

Validation and constraints:

- `formation_key` is unique.
- `game_id` is unique and required; a durable match never exists without a recoverable destination.
- `mode` is restricted to approved competitive modes.
- `matched` has no terminal timestamp.
- `active` requires `started_at`.
- `completed` requires `started_at` and `completed_at`, with no `closed_at`.
- `cancelled` and `failed_to_start` require `closed_at`, with no `completed_at`.
- Deleting the linked game is restricted so competitive history cannot silently disappear.

State transitions:

```text
matched -> active -> completed
matched -> cancelled
matched -> failed_to_start
active -> cancelled
active -> failed_to_start  (launch/setup failure before meaningful gameplay)
```

Terminal states are immutable except through an explicit future administrative repair workflow.

Indexes:

- Unique `formation_key`.
- Unique `game_id`.
- `(mode, status, matched_at DESC, id DESC)` for bounded operational/history reads.
- `(status, updated_at)` for future cleanup/reconciliation.

### Match Participant

Backed by new `match_players` table.

Fields:

- `match_id`: foreign key to `matches.id`.
- `user_id`: required foreign key to `users.id` because ranked play is registered-only.
- `game_player_id`: required unique foreign key to `game_players.id`.
- `status`: `assigned`, `active`, `completed`, `cancelled`, or `failed`.
- `assigned_at`: durable assignment time.
- `completed_at`: nullable successful completion time.
- `closed_at`: nullable cancellation/failure time.

Validation and constraints:

- Primary key `(match_id, user_id)`.
- Unique `(match_id, game_player_id)` and globally unique `game_player_id`.
- Partial unique `user_id` while status is `assigned` or `active`; this is the final guard against concurrent duplicate ranked assignments.
- `completed` requires `completed_at`.
- `cancelled` or `failed` requires `closed_at`.
- Each match is created with exactly two participants in the same transaction; service and integration tests enforce the cardinality because a simple row check cannot count sibling rows.
- User/game-player deletion is restricted for competitive history. Existing account status supports disabled/deleted users without hard-deleting the identity.

State transitions:

```text
assigned -> active -> completed
assigned -> cancelled
assigned -> failed
active -> cancelled
active -> failed
```

Indexes:

- Partial unique `(user_id)` for `assigned`, `active`.
- `(user_id, match_id)` for current-status recovery.
- Unique `game_player_id`.

### Ranked Game

Backed by existing `games` with related `game_players`, `rounds`, and `guesses`.

Fields and rules:

- `games.mode`: `ranked`.
- `games.map_id`: configured active standard competitive map.
- `games.round_count`: configured standard count, default 5.
- `games.timer_seconds`: configured competitive round timer, default 60.
- `games.scoring_version`: existing server scoring version.
- `game_players`: exactly two registered player snapshots with unique `(game_id, user_id)`.
- `rounds`: distinct server-selected active locations; first round begins after a short configured assignment countdown.
- `guesses`: existing one-guess-per-player-per-round server-scored records.

Validation:

- A client cannot create a ranked game through the general solo-game creation operation.
- Formation validates an active map and enough distinct active locations before committing.
- Both player rows, all planned rounds, match, and participant links commit atomically.
- Guessing before the scheduled round start or after its deadline is rejected by server time.
- Ranked mode uses multiplayer completion: a round advances after both eligible players submit or its deadline passes.
- Match/participant lifecycle changes occur in the same transaction as ranked game lifecycle changes.
- Existing hidden-coordinate and participant authorization rules apply.

## Ephemeral Entities

### Queue Entry

Suggested Redis keys:

- `matchmaking:v1:queue:{mode}`: sorted set ordered by original enqueue epoch milliseconds.
- `matchmaking:v1:player:{user_id}`: leased hash containing the player's single active queue/claim state.

Player hash fields:

- `entry_id`: opaque unique queue-entry identity.
- `user_id`: registered player identity, represented by the key and validated payload.
- `mode`: `ranked_standard`.
- `state`: `searching` or `claimed`.
- `enqueued_at_ms`: original queue time; never changed by retries or lease renewal.
- `lease_expires_at_ms`: current activity lease.
- `claim_id`: present only while claimed.

Rules:

- One player pointer across all modes.
- Join returns an existing valid searching entry unchanged.
- Status renewal extends only the lease, not queue priority.
- Missing/expired/mismatched player hashes make sorted-set members stale and ineligible.
- Queue payloads contain no email, token, profile preference, coordinates, guesses, or opponent details.

State transitions:

```text
absent -> searching -> claimed -> absent
searching -> absent          (leave or lease expiry)
claimed -> searching         (safe recovery/requeue)
claimed -> absent            (durable formation committed)
```

### Pair Claim

Suggested Redis keys:

- `matchmaking:v1:claim:{claim_id}`: short-lived pair/formation record.
- `matchmaking:v1:claims`: sorted set ordered by recovery deadline.

Fields:

- `claim_id` / `formation_key`: shared idempotency identity.
- `mode`.
- two `entry_id`, `user_id`, and original `enqueued_at_ms` values.
- `claimed_at_ms`.
- `recover_after_ms`.

Rules:

- Created atomically while both player pointers move from `searching` to `claimed` and both members leave the queue sorted set.
- Claim script validates that both entries are current, leased, compatible, and distinct.
- Only exact matching `entry_id` and `claim_id` values may finalize or release a claim.
- On recovery, PostgreSQL is checked by `formation_key` first.
- If durable match exists, Redis is finalized/cleaned.
- If no match exists and both players remain eligible, they are requeued with original timestamps.
- If one player is no longer eligible, only eligible players may return to searching.

### Matchmaking Status

Public-safe computed state, not independently stored.

States:

- `not_queued`: no durable active match and no valid searching/claim state.
- `searching`: valid leased queue entry.
- `matched`: durable assigned/active match exists; PostgreSQL wins even if Redis is stale or unavailable.
- `temporarily_unavailable`: no durable assignment can answer the request and ephemeral state cannot be determined safely.

Status details:

- Searching: mode, search start, and lease expiry only.
- Matched: match ID, game ID, mode, formed time, and destination path only.
- Never exposes the opponent, queue position, exact queue population, hidden locations, guesses, or private account fields.

## Formation Transaction

Before the transaction:

1. Atomically claim the oldest valid compatible pair in Redis.
2. Select the configured number of distinct active locations for the configured map.

Inside one PostgreSQL transaction:

1. Lock both `users` rows in stable UUID order.
2. Revalidate both accounts are active.
3. Recheck no active `match_players` assignment and no conflicting active game exists.
4. Create one ranked game and its two registered game-player snapshots.
5. Create all rounds and schedule the first server-controlled countdown/deadline.
6. Create one match using the claim ID as `formation_key`.
7. Create exactly two match-player rows.
8. Commit.

After commit:

1. Best-effort finalize the exact Redis claim.
2. Return/recover the durable match for both players.

On failure before commit, all PostgreSQL writes roll back and the exact Redis claim is released or recovered without losing original priority.

## Concurrency Invariants

- Atomic Redis scripts prevent two workers from claiming the same live queue entry.
- A unique formation key makes transaction replay converge on one match.
- Stable user-row lock ordering prevents formation deadlocks.
- Partial unique active participant index prevents one user from committing to two active matches even if Redis correctness is bypassed.
- Unique game and game-player links prevent duplicate destinations or participant snapshots.
- Leave wins only while an entry is still `searching`; after claim or durable commit it cannot destroy assignment state.
- Durable match lookup always precedes presenting `not_queued`.

## Configuration

- `MATCHMAKING_DEFAULT_MAP_ID`: required active map UUID for `ranked_standard`.
- `MATCHMAKING_QUEUE_LEASE_SECONDS`: default 30.
- `MATCHMAKING_CLAIM_TTL_SECONDS`: default 15.
- `MATCHMAKING_START_DELAY_SECONDS`: default 5.
- `MATCHMAKING_ROUND_COUNT`: default 5, valid 1 to 10.
- `MATCHMAKING_TIMER_SECONDS`: default 60, valid 10 to 600.
- `MATCHMAKING_CANDIDATE_SCAN_LIMIT`: default 20, bounded to protect Redis work.

## Privacy And Retention

- Redis state is short-lived and contains only identifiers and coordination timestamps.
- Durable match rows retain competitive integrity facts but no raw tokens, IP addresses, hidden location answers, or guess coordinates beyond existing authorized gameplay records.
- Logs and metric labels must not include user IDs, access/CSRF tokens, raw Redis keys, email, opponent details, hidden locations, or guesses.
- Future hard-delete/privacy requirements need a separate design for pseudonymized competitive history; this phase follows existing account-status retention.
