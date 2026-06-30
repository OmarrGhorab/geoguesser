# Phase 0 Research: Room Realtime Reconnection

## Decision: Use WebSocket For Room Channels

Use WebSocket for `/realtime/rooms/{roomCode}` room channels and keep create, join, settings, start, remove-player, and guess submissions as authoritative HTTP commands.

**Rationale**: Private rooms need low-latency server-to-client events plus client-to-server liveness messages for presence and reconnect behavior. WebSocket avoids noisy polling and gives one persistent channel for lobby, presence, round, guess-progress, result, and recovery events. Keeping mutations on HTTP preserves existing validation, CSRF/idempotency patterns, and clearer retry semantics.

**Alternatives considered**:

- Server-Sent Events plus HTTP heartbeats: simpler server transport, but awkward for bidirectional presence and reconnect state.
- HTTP polling only: easiest to implement, but too slow/noisy for synchronized multiplayer and does not meet the 2-second live update target reliably.
- WebSocket-only commands: lower round-trips, but weakens the existing API contract model and makes idempotent command retries harder.

## Decision: Use `github.com/coder/websocket`

Add `github.com/coder/websocket` for backend WebSocket accept/read/write behavior.

**Rationale**: The project currently has no WebSocket dependency. `github.com/coder/websocket` is actively maintained, has context-aware APIs, JSON helpers, zero dependencies, and is the maintained successor path for nhooyr/websocket. This fits the backend's context-first request handling and keeps dependency footprint small.

**Alternatives considered**:

- `gorilla/websocket`: mature and widely used, but the project history has maintenance interruptions and the API is less context-oriented.
- `golang.org/x/net/websocket`: older and not recommended for new server work.
- Build raw WebSocket handling manually: unnecessary protocol risk and not worth the maintenance cost.

## Decision: Store Durable Room Facts In Existing Tables

Use existing `rooms`, `room_players`, `games`, `game_players`, `rounds`, and `guesses` for durable facts. Add migrations only if implementation proves a missing constraint, index, or field is required.

**Rationale**: The initial schema already contains private-room durable entities, active room-code uniqueness, room membership history, private-room game mode, game participants, rounds, and guesses. Reusing these tables avoids unnecessary data churn and keeps Phase 06 focused on room orchestration, realtime, and multiplayer state transitions.

**Alternatives considered**:

- Create new room tables: duplicates existing schema and increases migration risk.
- Store active room state only in Redis: not durable enough for reloadable results and reconnect recovery.
- Add host participant FK immediately: useful but not required because host identity can be represented by the host `game_player` role plus `room_players`; registered host history can still use `rooms.host_user_id`.

## Decision: Use Redis For Active Coordination

Use Redis for active room snapshots, event version counters, presence records, reconnect windows, ready state, room locks, pub/sub fanout, idempotency claims, and rate-limit counters.

**Rationale**: Room state is frequently updated, short-lived, and needs cross-process fanout. Redis is already part of the stack and is the documented boundary for presence, active room snapshots, realtime fanout, and locks.

**Alternatives considered**:

- PostgreSQL polling for active state: durable but too chatty and slower for live room updates.
- In-memory hub only: simpler for one process, but fails horizontal scaling and loses state on restart.
- Durable event table for all room events: useful for audit/replay later, but overkill for MVP room hints because durable facts already exist in game/room tables.

## Decision: Version Events And Refetch On Mismatch

Every room event includes an event ID, room code, occurred timestamp, monotonic room version, type, and compact payload. Clients apply events only when versions are expected and refetch authoritative room state after reconnect, gaps, duplicates, or stale events.

**Rationale**: Realtime events are hints, not the source of truth. Versioning lets the UI update optimistically while preserving correctness after missed messages, reconnects, or multiple backend instances.

**Alternatives considered**:

- Full room snapshot in every event: simpler client logic but wasteful and increases accidental hidden-data exposure.
- Client trusts event order only: fragile across reconnects and pub/sub delivery.
- Durable event replay: stronger recovery, but not required when clients can refetch current authoritative state.

## Decision: Extend Game Logic For Multiplayer Round Completion

Add a multiplayer-aware game progression path for private rooms instead of trying to reuse the solo "one guess completes a round" behavior unchanged.

**Rationale**: Solo currently completes a round immediately after one accepted guess. Private rooms must wait for all active eligible players or a server deadline, assign missed-round zeroes, reveal results to all players together, and then start the next round. The existing scoring and result DTO concepts remain reusable, but progression rules need a room-aware path.

**Alternatives considered**:

- Treat each player as a separate solo game: breaks synchronized rounds and shared final results.
- Complete the room round after first guess: unfair and violates the spec.
- Defer multiplayer scoring to a future phase: impossible because two players must complete a private room game in this phase.

## Decision: Reconnect Grace Defaults To 30 Seconds

Use a configurable 30-second reconnect grace window, with heartbeat/presence TTLs derived from a small heartbeat interval plus missed-heartbeat tolerance.

**Rationale**: The spec requires 30 seconds by default. It is short enough to avoid stale lobby clutter while allowing common refresh/network hiccups. Configuration keeps production tuning possible without code changes.

**Alternatives considered**:

- No grace window: too punishing for refreshes and mobile network transitions.
- Several minutes: slows lobby cleanup and can leave active rooms feeling stuck.
- Per-room custom grace: unnecessary complexity for MVP.

## Decision: Degraded Mode Keeps HTTP Commands Authoritative

When realtime delivery is unavailable, clients show reconnecting/degraded state and refresh room/game state through existing server reads; commands still go through authoritative HTTP endpoints.

**Rationale**: This preserves gameplay correctness even when live delivery is temporarily unavailable. It also gives the frontend a clear accessibility and UX path for reconnection without trusting local state.

**Alternatives considered**:

- Block all gameplay when realtime drops: safer but too disruptive.
- Let client continue from local state only: risks stale timers, duplicate guesses, and hidden-data leaks.
- Poll continuously as primary mode: noisy and weaker UX than reconnect-driven fallback.
