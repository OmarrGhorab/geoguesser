# Data Model: Room Realtime Reconnection

## Overview

Phase 06 uses existing durable gameplay tables for facts and Redis-backed ephemeral state for active room coordination. Durable data must be enough to reload a completed or active room after browser refresh. Ephemeral data must be safe to discard and rebuild from durable facts where possible.

## Durable Entities

### Room

Backed by existing `rooms`.

Fields:

- `id`: UUID room identity.
- `game_id`: optional linked private-room game.
- `code`: uppercase join code, unique while room is joinable.
- `visibility`: `private` for this phase.
- `status`: `lobby`, `active`, `completed`, `expired`, `cancelled`.
- `host_user_id`: registered host when available; guest hosts are represented through host participant role.
- `max_players`: 2 to 50.
- `round_count`: 1 to 10.
- `timer_seconds`: null or 10 to 600.
- `expires_at`, `created_at`, `updated_at`: lifecycle timestamps.

Validation:

- Joinable room codes are unique and non-enumerable.
- Settings can change only in `lobby`.
- Completed, expired, or cancelled rooms cannot be joined.
- Active rooms can be rejoined only by an existing matching participant.

State transitions:

```text
lobby -> active -> completed
lobby -> expired
lobby -> cancelled
active -> completed
active -> cancelled
```

### Room Participant

Backed by existing `room_players` plus linked `game_players`.

Fields:

- `room_id`: owning room.
- `game_player_id`: linked game participant.
- `status`: `joined`, `left`, `kicked`, `disconnected`.
- `joined_at`, `left_at`: membership timestamps.
- `game_players.user_id` or `guest_identity_hash`: participant identity.
- `game_players.display_name`: snapshot display name.
- `game_players.role`: `host`, `player`, future `spectator`.
- `game_players.status`: active game participant state.
- `game_players.total_score`: durable total score.

Validation:

- One registered user per room game.
- One guest identity per room game.
- A kicked participant cannot rejoin the same room.
- A reconnect must match the same registered user or guest identity.
- Host-only commands require a participant with host role.

### Private Room Game

Backed by existing `games`.

Fields:

- `id`: game identity.
- `mode`: `private_room`.
- `status`: `pending`, `active`, `completed`, `abandoned`, `cancelled`.
- `map_id`, `round_count`, `timer_seconds`, `scoring_version`.
- `started_at`, `completed_at`.

Validation:

- Game settings mirror locked room settings at start.
- A room has at most one linked game.
- Round count and timer ranges match existing game constraints.

State transitions:

```text
pending -> active -> completed
pending -> cancelled
active -> abandoned
active -> cancelled
```

### Round

Backed by existing `rounds`.

Fields:

- `id`, `game_id`, `location_id`, `round_number`.
- `status`: `pending`, `active`, `completed`, `cancelled`.
- `starts_at`, `ends_at`, `revealed_at`.

Validation:

- Only one active round per game.
- `location_id` and answer coordinates are never exposed before authorized reveal.
- Timed rounds reject guesses after `ends_at`.

State transitions:

```text
pending -> active -> completed
active -> cancelled
```

### Guess

Backed by existing `guesses`.

Fields:

- `id`, `round_id`, `game_player_id`.
- submitted `latitude`, `longitude`.
- server-calculated `distance_meters`, `score`.
- optional `idempotency_key`.
- `submitted_at`, `created_at`.

Validation:

- One guess per player per round.
- Optional idempotency key is unique per player where present.
- Guess coordinates must be valid latitude/longitude.
- Late guesses and non-current-round guesses are rejected.

## Ephemeral Entities

### Room State Snapshot

Suggested Redis key: `rooms:{code}:snapshot`.

Fields:

- `room_code`, `room_id`, `game_id`.
- `status`, `version`, `updated_at`.
- safe settings summary.
- safe roster summary with participant presence.
- active round summary when started.
- aggregate guess progress.

Rules:

- Must not contain hidden coordinates, location IDs, or answer-revealing provider metadata.
- Can be rebuilt from PostgreSQL plus active presence.
- TTL extends while room is active and expires after completion/cleanup.

### Room Version

Suggested Redis key: `rooms:{code}:version`.

Fields:

- monotonic integer version.

Rules:

- Incremented on every state-changing command or presence transition that should notify clients.
- Included in every realtime event and room snapshot.
- Used by clients to detect stale, duplicate, or missing events.

### Presence Record

Suggested Redis key: `rooms:{code}:presence:{game_player_id}`.

Fields:

- `connection_id`.
- `status`: `connected`, `disconnected`.
- `last_seen_at`.
- optional user agent hash/request metadata for diagnostics only.

Rules:

- Refreshed by heartbeat/liveness messages.
- Expires after missed-heartbeat threshold.
- Expiration marks the participant disconnected and emits a presence event.
- Must not store raw cookies, tokens, IP-sensitive data, or hidden coordinates.

### Reconnect Window

Suggested Redis key: `rooms:{code}:reconnect:{game_player_id}`.

Fields:

- `game_player_id`.
- `expires_at`.
- `last_version`.

Rules:

- Default TTL is 30 seconds.
- Reconnect succeeds only for the same registered user or guest identity.
- Reconnect restores presence and returns authoritative room state.

### Ready State

Suggested Redis key: `rooms:{code}:ready`.

Fields:

- set/map of ready `game_player_id` values.

Rules:

- Stored only while room is in lobby.
- Cleared when membership or fairness-affecting settings change.
- Host can start without all players ready unless implementation tasks choose to enforce all-ready start as a product refinement.

### Room Lock

Suggested Redis key: `rooms:{code}:lock`.

Rules:

- Protects race-sensitive commands such as start room, settings update, join capacity check, remove player, and round transition.
- Has a short TTL and safe release behavior.
- Durable database constraints remain the final protection against race bugs.

### Realtime Event

Suggested Redis pub/sub channel: `rooms:{code}:events`.

Envelope:

- `event_id`.
- `type`.
- `room_code`.
- `game_id`.
- `occurred_at`.
- `version`.
- `payload`.

Event types:

- `room.snapshot`.
- `room.player_joined`.
- `room.player_left`.
- `room.player_disconnected`.
- `room.player_reconnected`.
- `room.player_removed`.
- `room.settings_updated`.
- `room.ready_updated`.
- `room.ready_reset`.
- `room.started`.
- `round.started`.
- `round.guess_count_changed`.
- `round.ended`.
- `round.results_revealed`.
- `game.completed`.
- `room.error`.

Rules:

- Payloads are compact and safe.
- Events are hints; clients refetch on mismatch or reconnect.
- `event_id` supports deduplication.

## Relationships

- A `Room` has one optional `Private Room Game`.
- A `Room` has many `Room Participants`.
- A `Room Participant` wraps one `GamePlayer`.
- A `Private Room Game` has many `Rounds`.
- A `Private Room Game` has many `GamePlayers`.
- A `Round` receives at most one `Guess` from each active eligible `GamePlayer`.
- A `Realtime Event` references a `Room` and optionally a `Game`/`Round` through safe identifiers.

## Query And Index Needs

Existing indexes cover:

- Active room lookup by code.
- Room cleanup by status/expiration.
- Host room history.
- Game player uniqueness by user or guest identity.
- Game roster lookup by game/status.
- Round lookup by game/round number.
- Guess uniqueness by round/player.
- Guess idempotency by player/key from Phase 04.

Implementation should add a migration only if tests reveal missing performance or integrity support, such as:

- faster room membership lookup by `room_id, status`;
- room-player lookup by `game_player_id`;
- additional status values required by implementation.

## Hidden Data Rules

- Pre-reveal room state and events must not include `rounds.location_id`.
- Pre-reveal room state and events must not include actual latitude/longitude.
- Pre-reveal media must not include provider metadata that trivially identifies the answer beyond what is already shown as playable media.
- Result reveal can include actual location only after a guess, round reveal, or final result authorization.

## Configuration Values

- `ROOM_RECONNECT_GRACE_SECONDS`: default `30`.
- `ROOM_HEARTBEAT_INTERVAL_SECONDS`: default selected during implementation, expected 5 to 15 seconds.
- `ROOM_PRESENCE_TTL_SECONDS`: derived from heartbeat interval and missed heartbeat tolerance.
- `NEXT_PUBLIC_REALTIME_URL`: browser-visible realtime base URL for local/dev where same-origin proxy is unavailable.
