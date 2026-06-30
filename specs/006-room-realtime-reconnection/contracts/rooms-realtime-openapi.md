# Contracts: Rooms And Realtime

## Contract Scope

This artifact guides updates to `backend/openapi/openapi.yaml`, frontend API types, and realtime event documentation for Phase 06.

## REST Room Commands

The existing OpenAPI file already declares these room paths. Phase 06 implementation must complete the backend behavior and expand schemas where needed:

- `POST /api/v1/rooms`
- `POST /api/v1/rooms/join`
- `GET /api/v1/rooms/{roomCode}`
- `PATCH /api/v1/rooms/{roomCode}/settings`
- `POST /api/v1/rooms/{roomCode}/start`
- `DELETE /api/v1/rooms/{roomCode}/players/{playerId}`

### Required Schema Updates

`Room` should include:

- `id`
- `code`
- `visibility`
- `status`
- `game_id`
- `host_player_id`
- `version`
- `max_players`
- `round_count`
- `timer_seconds`
- `expires_at`
- `players`
- `ready_player_ids`
- `current_round`
- `guess_progress`

`RoomPlayer` should include:

- `id`
- `user_id`
- `display_name`
- `role`
- `membership_status`
- `presence_status`
- `is_ready`
- `total_score`
- `joined_at`
- `left_at`

`RoomCurrentRound` should include:

- `id`
- `round_number`
- `status`
- `starts_at`
- `ends_at`
- `media`
- `revealed`

It must not include hidden answer fields before reveal.

`RoomGuessProgress` should include:

- `submitted_count`
- `eligible_count`
- `submitted_player_ids`

It must not expose guess coordinates or answer data before reveal.

### Command Request Rules

`CreateRoomRequest`:

- `map_id`: required UUID.
- `visibility`: required, Phase 06 accepts `private`.
- `round_count`: required 1 to 10.
- `timer_seconds`: nullable, 10 to 600 when present.
- `max_players`: required 2 to 50.
- Optional display name can be accepted if the host is a guest and no profile name exists.

`JoinRoomRequest`:

- `code`: required uppercase 6 to 10 characters after normalization.
- `display_name`: optional 2 to 32 characters for guest/player snapshot.

`UpdateRoomSettingsRequest`:

- optional `map_id`, `round_count`, `timer_seconds`, `max_players`, and ready-reset behavior.
- allowed only in `lobby`.

`StartRoom`:

- requires `Idempotency-Key`.
- allowed only for host while room is in `lobby`.

`RemoveRoomPlayer`:

- host-only.
- cannot remove the current host unless host transfer behavior is explicitly triggered.

## Error Codes

Room handlers should map service errors into the existing error envelope with stable codes:

- `room_not_found`
- `room_full`
- `room_not_joinable`
- `room_expired`
- `room_already_started`
- `room_settings_locked`
- `room_host_required`
- `room_player_not_found`
- `room_player_removed`
- `room_reconnect_expired`
- `room_identity_mismatch`
- `room_code_rate_limited`
- `round_closed`
- `round_not_current`
- `already_guessed`
- `idempotency_conflict`
- `realtime_origin_forbidden`
- `realtime_auth_required`

## Realtime Endpoint

Endpoint:

```text
GET /realtime/rooms/{roomCode}
```

Authentication:

- Uses existing registered or guest session cookies.
- Requires same-origin or configured allowed origin.
- Rejects anonymous, non-participant, removed, or kicked clients.

Connection behavior:

- On connect, server authenticates participant and sends `room.snapshot`.
- Client sends heartbeat/liveness messages at the configured interval.
- Server publishes room events to all active connections for the room.
- On reconnect, client discards local room state and reloads authoritative room state or accepts the server snapshot.

## Event Envelope

```json
{
  "event_id": "evt_01J...",
  "type": "room.player_joined",
  "room_code": "ABCD12",
  "game_id": "0197...",
  "occurred_at": "2026-06-30T12:00:00Z",
  "version": 7,
  "payload": {}
}
```

Rules:

- `event_id` is unique.
- `version` increases monotonically per room.
- Events are hints, not authoritative state.
- Payloads must be hidden-coordinate-safe.
- Clients refetch room state after reconnect, missing versions, duplicate versions, or stale versions.

## Event Types

### `room.snapshot`

Payload:

- `room`: full safe `Room` response shape.

### `room.player_joined`

Payload:

- `player`: safe `RoomPlayer`.

### `room.player_left`

Payload:

- `player_id`
- `membership_status`

### `room.player_disconnected`

Payload:

- `player_id`
- `reconnect_expires_at`

### `room.player_reconnected`

Payload:

- `player_id`
- `presence_status`

### `room.player_removed`

Payload:

- `player_id`
- `membership_status`

### `room.settings_updated`

Payload:

- safe settings fields.
- optional `ready_reset` boolean.

### `room.ready_updated`

Payload:

- `player_id`
- `is_ready`
- `ready_count`
- `eligible_count`

### `room.ready_reset`

Payload:

- `reason`

### `room.started`

Payload:

- `room`
- `game`

### `round.started`

Payload:

- `round`: safe current-round DTO.
- `guess_progress`

### `round.guess_count_changed`

Payload:

- `round_id`
- `submitted_count`
- `eligible_count`
- `submitted_player_ids`

### `round.ended`

Payload:

- `round_id`
- `reason`: `all_submitted`, `deadline`, or `cancelled`.

### `round.results_revealed`

Payload:

- `round_id`
- per-player round score summary.
- revealed location only when reveal is allowed.

### `game.completed`

Payload:

- final safe game result summary.

### `room.error`

Payload:

- stable `code`.
- safe, non-secret message key or message.

## Hidden Coordinate Contract

Before reveal, none of the following may appear in REST or realtime payloads:

- `location_id`
- actual answer latitude
- actual answer longitude
- provider references that directly reveal answer location
- hidden admin notes or private location metadata

## Frontend Type Contract

Add frontend room types under `client/features/rooms/types.ts` and server-only calls under `client/lib/api/rooms.ts`.

Required helper operations:

- `createRoom`
- `joinRoom`
- `getRoom`
- `updateRoomSettings`
- `startRoom`
- `removeRoomPlayer`

Client realtime module:

- connects to `NEXT_PUBLIC_REALTIME_URL` plus `/rooms/{roomCode}`;
- validates event envelopes defensively;
- exposes connection states `connecting`, `connected`, `reconnecting`, `degraded`, `closed`;
- triggers authoritative refetch on version mismatch or reconnect.
