# Data Model: Casual And Ranked Team Modes

## Ownership And Authority

- PostgreSQL is authoritative for party membership, match/game facts, accepted guesses, messages/reports, seasons, standings, and rating changes.
- Redis is authoritative only for searching/claimed queue tickets and disposable live state. Once match formation commits, PostgreSQL always wins recovery.
- R2/local storage contains private sanitized chat derivatives; PostgreSQL file/message rows control access.
- All timestamps are UTC. UUID primary keys use the repository's existing UUID conventions. User/profile deletions remain restricted or soft-deleted where competitive/moderation history requires identity continuity.

## Migration Strategy

Use one additive Goose migration: `backend/migrations/00019_casual_ranked_team_modes.sql`.

1. Create party, season/standing/rating, chat/moderation tables and seed one active Season 1 beginning at the migration transaction time with an 84-day end.
2. Add nullable/backfilled match/game/guess/upload columns and new constraints/indexes.
3. Backfill existing `ranked_standard` matches as `playlist='ranked'`, `format='solo'`, `team_size=1`, linked to seeded Season 1; assign deterministic team slots 1 and 2 by `(assigned_at, user_id)` within each legacy match; copy guess `score` to `accuracy_score` with zero bonus.
4. Make required backfilled columns non-null where safe. Preserve `matches.mode='ranked_standard'` as a readable legacy value; new writes use canonical six-mode values.
5. Expand the guess total-score constraint from 5,000 to 5,250 while preserving the 0–5,000 accuracy constraint.
6. Down migration is supported only before new rows exist; otherwise it raises an explicit irreversible-data exception rather than deleting match, rating, or moderation history.

## Durable Entities

### Party

Table: `parties`

| Field | Rules |
| --- | --- |
| `id` | UUID primary key |
| `format` | `duo` or `squad`; Solo has no durable party |
| `capacity` | 2 for Duo, 4 for Squad; check constraint agrees with format |
| `leader_user_id` | Active member and registered user |
| `status` | `forming`, `queued`, `in_match`, `closed` |
| `version` | Non-negative monotonic value incremented by every membership/readiness/queue transition |
| `active_match_id` | Nullable match reference; required only for `in_match` |
| `created_at`, `updated_at`, `closed_at` | Lifecycle timestamps |

State transitions:

```text
forming -> queued -> in_match -> forming
forming -> closed
queued -> forming          (leader leaves queue or ticket recovery fails)
queued -> closed           (party disbanded)
in_match -> closed         (all members leave/party disbands after terminal match)
```

Only the leader can invite, kick, join/leave queue, or disband. If the leader voluntarily leaves while `forming`, leadership transfers to the earliest `joined_at`, then lowest user ID. Any roster change clears all remaining readiness. A party cannot change while `queued`; the leader must leave the queue first. `in_match` departures go through match abandonment, not party mutation.

Indexes: `(leader_user_id, status)`, `(status, updated_at)`, partial index on `active_match_id IS NOT NULL`.

### Party Member

Table: `party_members`

| Field | Rules |
| --- | --- |
| `party_id`, `user_id` | Composite primary key; registered user |
| `status` | `active`, `left`, `kicked` |
| `ready` | Only meaningful for active membership |
| `joined_at`, `left_at` | Lifecycle timestamps |

Constraints/indexes:

- Partial unique `user_id` where `status='active'`; one active party per user.
- Active member count never exceeds party capacity; service transaction locks the party row before changes.
- Leader must have an active membership, enforced by transactional service checks.
- `(party_id, status, joined_at, user_id)` supports roster/leadership reads.

### Party Invite

Table: `party_invites`

| Field | Rules |
| --- | --- |
| `id` | UUID primary key |
| `party_id`, `inviter_user_id`, `invitee_user_id` | Inviter must be current leader; invitee must be active accepted friend with no either-direction block |
| `status` | `pending`, `accepted`, `declined`, `expired`, `revoked` |
| `expires_at` | 15 minutes after creation |
| `created_at`, `responded_at` | Lifecycle timestamps |

Partial unique `(party_id, invitee_user_id)` for pending invites. Acceptance locks invite, party, and user rows in stable UUID order; it rechecks friendship/block, capacity, account eligibility, and active-party uniqueness.

### Match Extensions

Existing table: `matches`

Add:

| Field | Rules |
| --- | --- |
| `playlist` | `casual` or `ranked` |
| `format` | `solo`, `duo`, `squad` |
| `team_size` | 1, 2, or 4 and must agree with format |
| `season_id` | Required for Ranked, null for Casual |
| `team_one_score`, `team_two_score` | Non-negative final/running totals, default 0 |
| `winner_team_slot` | Null for active/draw/cancelled; 1 or 2 for decisive completion |
| `result` | Nullable until terminal: `team_one_win`, `team_two_win`, `draw`, `forfeit`, `abandoned`, `cancelled` |
| `last_activity_at` | Updated by accepted gameplay/collaboration activity; drives Casual 10-minute safeguard |
| `chat_access_until` | Terminal time plus 15 minutes; null before terminal; bounds normal participant chat/history/attachment access |
| `progression_finalized_at` | Required for completed Ranked once rating transaction commits; null for Casual |

Canonical new `mode` values are `casual_solo`, `casual_duo`, `casual_squad`, `ranked_solo`, `ranked_duo`, and `ranked_squad`. `ranked_standard` is a legacy alias interpreted as Ranked Solo.

Lifecycle remains `matched -> active -> completed/cancelled/failed_to_start`; abandonment/forfeit is represented by terminal result plus match/player facts, not a new lifecycle state.

New indexes: `(playlist, format, status, matched_at DESC, id)`, `(season_id, status, completed_at)`, `(status, last_activity_at)` for bounded sweeps.

### Match Participant Extensions

Existing table: `match_players`

Add:

| Field | Rules |
| --- | --- |
| `team_slot` | 1 or 2; each completed formation has exactly `team_size` participants per slot |
| `party_id` | Nullable for Solo; both team members reference their formation party snapshot |
| `abandoned_at` | Set only for a participant who exhausts reconnect grace or explicitly leaves |
| `abandon_reason` | Bounded enum: `explicit_leave`, `disconnect_timeout`, `account_ineligible` |

Existing partial unique active assignment by user remains the final cross-format concurrency guard. Add `(match_id, team_slot, status, user_id)`.

### Game Player Extension

Existing table: `game_players`

Add nullable `team_slot SMALLINT`. It is 1 or 2 for matchmade Casual/Ranked players and remains null for existing solo/daily/private-room rows. Add `(game_id, team_slot, status, id)`.

### Guess Extensions

Existing table: `guesses`

Add:

| Field | Rules |
| --- | --- |
| `accuracy_score` | 0–5,000; existing geography score |
| `speed_bonus` | 0–250; zero outside Ranked |
| existing `score` | Must equal `accuracy_score + speed_bonus`; range 0–5,250 |

Ranked bonus calculation inside the locked round transaction:

```text
duration_ms  = max(1, ends_at - starts_at)
remaining_ms = clamp(ends_at - submitted_at, 0, duration_ms)
speed_bonus  = min(250, floor(accuracy_score * 0.05 * remaining_ms / duration_ms))
score        = accuracy_score + speed_bonus
```

At or after the server deadline, normal submission is rejected; deadline closure inserts/records zero-point timed-out guesses according to existing multiplayer behavior. Idempotent replay returns the originally stored fields.

### Competitive Season

Table: `competitive_seasons`

| Field | Rules |
| --- | --- |
| `id` | UUID primary key |
| `sequence` | Positive unique increasing integer |
| `slug`, `name` | Unique stable slug; localized client renders generic season label plus number/name |
| `status` | `scheduled`, `active`, `closed` |
| `starts_at`, `ends_at`, `closed_at` | `ends_at > starts_at`; closed requires `closed_at` |
| `initial_rating` | 800 for v1 |
| `elo_k` | 32 for v1 |
| `reset_factor_bps` | 5,000 (50%) |
| `top500_min_matches` | 25 |
| `created_at`, `updated_at` | Audit timestamps |

Only one active season via a partial unique index. Rollover holds a PostgreSQL advisory transaction lock, freezes the old standings, and creates the next 84-day season exactly once.

### Competitive Standing

Table: `competitive_standings`

Primary key: `(season_id, user_id)`.

| Field | Rules |
| --- | --- |
| `rating` | Non-negative; starts at season-reset value or 800 |
| `placements_completed` | 0–5; rank hidden until 5 |
| `matches_played`, `wins`, `losses`, `draws`, `abandons` | Non-negative and internally consistent |
| `peak_rating` | At least current season's minimum historical rating |
| `rating_reached_at` | Updated only when rating changes; top-500 third tie-break |
| `last_match_id` | Latest processed match |
| `final_position` | Null while active; frozen position on close for eligible ordered players |
| `ending_rank_code`, `peak_rank_code` | Frozen on season close; World Legend only when final position ≤500 |
| `created_at`, `updated_at` | Audit timestamps |

Active top-500 index:

```text
(season_id, rating DESC, wins DESC, rating_reached_at ASC, user_id ASC)
INCLUDE (matches_played, placements_completed, peak_rating)
```

Eligibility: placements completed, `matches_played >= season.top500_min_matches`, user active/in good standing, rating ≥2,000. Read-time `ROW_NUMBER()` over the eligible ordered set grants World Legend to positions 1–500. Cache TTL is ≤60 seconds and invalidated after any Ranked result.

### Division Mapping

| Rating | Code | Label |
| --- | --- | --- |
| 0–99 | `scout_3` | Scout III |
| 100–199 | `scout_2` | Scout II |
| 200–299 | `scout_1` | Scout I |
| 300–399 | `pathfinder_3` | Pathfinder III |
| 400–499 | `pathfinder_2` | Pathfinder II |
| 500–599 | `pathfinder_1` | Pathfinder I |
| 600–699 | `trailblazer_3` | Trailblazer III |
| 700–799 | `trailblazer_2` | Trailblazer II |
| 800–899 | `trailblazer_1` | Trailblazer I |
| 900–999 | `navigator_3` | Navigator III |
| 1,000–1,099 | `navigator_2` | Navigator II |
| 1,100–1,199 | `navigator_1` | Navigator I |
| 1,200–1,299 | `cartographer_3` | Cartographer III |
| 1,300–1,399 | `cartographer_2` | Cartographer II |
| 1,400–1,499 | `cartographer_1` | Cartographer I |
| 1,500–1,599 | `explorer_3` | Explorer III |
| 1,600–1,699 | `explorer_2` | Explorer II |
| 1,700–1,799 | `explorer_1` | Explorer I |
| 1,800–1,899 | `geo_master_3` | Geo Master III |
| 1,900–1,999 | `geo_master_2` | Geo Master II |
| ≥2,000 | `geo_master_1` | Geo Master I unless current top-500 eligibility grants World Legend |

Placement profiles expose progress only. World Legend is presentation derived from active/final position, never a rating threshold stored in place of the standard code.

### Rating Change

Table: `competitive_rating_changes`

| Field | Rules |
| --- | --- |
| `id` | UUID primary key |
| `season_id`, `match_id`, `user_id` | Unique `(match_id, user_id)` ensures exact-once application |
| `team_slot`, `outcome` | Participant's team and `win`, `loss`, or `draw` |
| `own_team_average`, `opponent_team_average` | Pre-match integer averages |
| `expected_bps` | Expected result in basis points for explainability |
| `base_delta` | Rounded Elo team delta |
| `abandon_penalty` | 0 or -15 |
| `old_rating`, `new_rating`, `total_delta` | `new = max(0, old + base + penalty)`; total reflects floor |
| `old_rank_code`, `new_rank_code` | Null while placements incomplete; World Legend evaluated separately |
| `created_at` | Finalization time |

Rating finalization locks match, active season, and all participant standings in sorted user order; verifies the game/match terminal result; inserts all changes and updates all standings in the same transaction; sets `matches.progression_finalized_at`; then invalidates leaderboard cache/publishes result events after commit. A unique conflict is treated as an idempotent replay and returns stored changes.

### Team Message

Table: `team_messages`

| Field | Rules |
| --- | --- |
| `id` | UUID primary key |
| `match_id`, `team_slot`, `author_user_id` | Author must be an active participant on that team at creation |
| `client_message_id` | UUID supplied by client; unique `(match_id, author_user_id, client_message_id)` |
| `sequence` | Database sequence used for stable pagination/event recovery |
| `text` | Trimmed, 0–500 characters; text or attachment must be present |
| `file_id` | Nullable, unique when present, purpose/context/owner must match |
| `status` | `active`, `reported`, `removed`, `expired` |
| `created_at`, `expires_at` | Normal expiry is match completion +30 days |

Indexes: `(match_id, team_slot, sequence DESC)`, `(expires_at, status)`, unique file attachment, unique idempotency key. Normal reads always filter the caller's team and active match access; muted-author filtering is applied per viewer without changing storage.

### Match Mute

Table: `match_mutes`

Primary key `(match_id, muter_user_id, muted_user_id)`. Both users must share the same match team; self-mute is rejected. Mute can be toggled during active/result-session access and expires with chat cleanup.

### Message Report

Table: `team_message_reports`

| Field | Rules |
| --- | --- |
| `id` | UUID primary key |
| `message_id`, `match_id`, `reporter_user_id` | Reporter must have been on message team; unique reporter/message |
| `reason` | `harassment`, `hate`, `sexual`, `violent`, `spam`, `other` |
| `status` | `pending`, `reviewed`, `dismissed`, `actioned` |
| `created_at`, `resolved_at`, `retention_until` | Retain referenced facts 180 days minimum from report |
| `legal_hold` | Default false; prevents cleanup while true |

No player-facing moderation queue is added in v1; operational access remains privileged and audited.

### Upload/File Extensions

Existing `uploads` and `files` add:

- `purpose`: `general` or `team_chat` (existing rows backfill `general`).
- `context_id`: match ID for team-chat assets, null for general.
- `sanitization_status`: `not_required`, `pending`, `ready`, `rejected`.
- `raw_storage_key`: retained only until sanitized/rejected cleanup; the normal file storage key points to the sanitized derivative.

Team-chat completion checks owner/context, declared metadata, detected MIME, decoded dimensions ≤20 MP, output longest edge ≤2,048 px, and sanitized JPEG output ≤5 MB. Only `ready` files can be attached. Raw objects are deleted immediately after successful derivative storage and within 24 hours after failed/abandoned uploads.

## Ephemeral Redis Entities

### Queue Ticket

Keys:

```text
matchmaking:v2:queue:{mode}                 ZSET enqueue time -> ticket_id
matchmaking:v2:ticket:{ticket_id}           HASH immutable roster/mode/rating + lease/state
matchmaking:v2:user:{user_id}               STRING ticket_id with same lease
matchmaking:v2:claim:{claim_id}              HASH two ticket snapshots and recovery deadline
matchmaking:v2:claims                        ZSET recovery deadline -> claim_id
```

Ticket fields: `ticket_id`, optional `party_id`, canonical mode, ordered user IDs, team size, team-average rating for Ranked, enqueue time, lease expiry, state (`searching`/`claimed`), claim ID, and party version. No names, chat, guesses, coordinates, tokens, or hidden locations.

A Lua join validates all user pointers are absent/same-ticket and creates the ticket atomically. A Lua claim scans at most 20 oldest candidates, rejects stale/mismatched party versions or rating windows, selects two disjoint equal rosters, removes both tickets, and writes one claim atomically. Release restores eligible tickets with original enqueue times; durable formation lookup by formation key always precedes requeue.

### Realtime Connection Ticket

```text
realtime:v1:ticket:{sha256(token)} -> user_id, channel_kind, channel_id, expires_at
```

Created after authenticated HTTP authorization, TTL 30 seconds, consumed atomically once during WebSocket upgrade, never logged or returned after consumption.

### Match Live State

```text
matchplay:v1:{match_id}:version                          INT
matchplay:v1:{match_id}:presence:{user_id}               STRING + 90s TTL
matchplay:v1:{match_id}:reconnect:{user_id}              HASH + 90s TTL
matchplay:v1:{match_id}:round:{round_id}:markers:{team}  HASH user_id -> lat/lng/version
matchplay:v1:{match_id}:round:{round_id}:views           HASH user_id -> provider-safe scene JSON
matchplay:v1:{match_id}:command:{user_id}:{command_id}   SETNX + bounded TTL
```

Round keys expire shortly after round reveal; match keys expire after the result session. Redis snapshots never contain answer coordinates or locked guesses.

## Realtime State And Audience Rules

Channel kinds are `party` and `match`. Every event has `event_id`, `type`, channel ID, optional game/round ID, UTC occurrence time, monotonic version, and payload. Audience is resolved server-side and omitted from public payloads.

- Party roster/readiness/queue/assignment events: active party members only.
- Match lifecycle/round counts/results: all match participants, with unrevealed opponent fields removed.
- Team messages/attachments/markers: same team only, excluding muted message authors per recipient.
- View state: delivered only to currently authorized submitted spectators; never includes map, cursor, proposed marker, locked coordinates, score, or answer.
- On version gaps or reconnect, client fetches the authorized HTTP snapshot; events are not treated as complete canonical history.

## Background Workers

- **Match lifecycle sweep** every 5 seconds, bounded batch 100: close expired Ranked rounds, forfeit disconnects past 90 seconds, end Casual matches with no activity for 10 minutes, and retry terminal progression finalization.
- **Season rollover** every minute under advisory lock: close ended season, freeze final facts, create next season/reset rows lazily or on first queue/profile read.
- **Retention cleanup** every 15 minutes, bounded batch 100: expire/delete eligible chat rows and sanitized/raw objects; skip reported/legal-hold content.
- **Claim recovery** retains the existing request-driven path and adds a bounded 5-second sweep so abandoned team claims recover without user traffic.

Workers share the process shutdown context, stop accepting new work before server shutdown, finish current transactions within the shutdown deadline, and expose last-success/failure metrics. PostgreSQL/Redis remain readiness dependencies; object storage degradation affects image status/metrics but not global readiness.
