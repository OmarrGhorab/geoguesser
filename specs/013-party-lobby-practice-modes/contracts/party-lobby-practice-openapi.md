# API Contract: Party Lobby And Practice Modes

The authoritative contract remains `backend/openapi/openapi.yaml`. This document records the intended delta.

## Party Lobby

Existing room endpoints remain:

- `POST /api/v1/rooms`
- `POST /api/v1/rooms/join`
- `GET /api/v1/rooms/{roomCode}`
- `PATCH /api/v1/rooms/{roomCode}/settings`
- `POST /api/v1/rooms/{roomCode}/ready`
- `POST /api/v1/rooms/{roomCode}/start`
- `DELETE /api/v1/rooms/{roomCode}/players/{playerId}`
- `GET /realtime/rooms/{roomCode}`

Creation accepts omitted `round_count`, `timer_seconds`, and `max_players`, defaulting to 5, 180, and 50. The response includes canonical `mode: party_lobby` and, once active or completed, a bounded `standings` array.

```json
{
  "placement": 1,
  "player_id": "uuid",
  "display_name": "Raven",
  "total_score": 24110,
  "total_distance_meters": 4200,
  "rounds_scored": 5,
  "tied": false
}
```

Legacy `private_room` stored games remain readable.

## Practice Creation

`POST /api/v1/games` accepts:

```json
{
  "mode": "practice",
  "map_id": "uuid"
}
```

For Practice, `round_count` and `timer_seconds` must be omitted or zero/null. The response reports `mode: practice`, `open_ended: true`, and `round_count: 1` as the materialized-round count.

Existing owner-only start, game read, current-round, and guess routes remain usable. Practice guesses reveal immediately, use `max_speed_bonus: 0`, keep `game_completed: false`, and set `next_round_available: true`.

## Create Next Practice Round

`POST /api/v1/games/{gameId}/rounds/next`

`Idempotency-Key` is required. Success returns the existing or newly created current round. Stable errors include `game_not_found`, `wrong_game_mode`, `current_round_incomplete`, `game_not_active`, `not_enough_locations`, and `idempotency_conflict`.

## Practice History

`GET /api/v1/games/{gameId}/rounds?limit=20&cursor=...`

Owner-only, default limit 20, maximum 100, ordered by round number ascending:

```json
{
  "items": [],
  "next_cursor": null,
  "has_more": false
}
```

`next_cursor` is an opaque, game-bound HMAC-signed cursor. Modified, malformed, or
cross-game cursors return the stable `invalid_cursor` error.

## End Practice

`POST /api/v1/games/{gameId}/end`

`Idempotency-Key` is required. It marks active Practice completed without deleting history. Replays return the existing terminal game. Other modes return `wrong_game_mode`.

## Security And Compatibility

- Existing session cookies and CSRF rules apply.
- Practice ownership failures use privacy-safe not-found behavior.
- No Practice realtime ticket or WebSocket channel exists.
- Neither mode returns competitive progression fields.
- Existing response fields remain source-compatible; new fields are additive.
