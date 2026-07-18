# Contract: Casual And Ranked Team Modes

This is the design source for additive changes to `backend/openapi/openapi.yaml`. Paths below are relative to `/api/v1` unless explicitly marked realtime. All HTTP bodies reject unknown properties. UUIDs are canonical strings; timestamps are UTC RFC 3339; errors use the existing `{ "error": { "code", "message", "request_id?", "fields?" } }` envelope.

## Global Security And Command Rules

- Every endpoint requires an active registered session unless stated otherwise.
- Unsafe cookie-authenticated HTTP methods require the existing CSRF protection.
- Create/accept/queue/guess/message/report/upload-complete commands require `Idempotency-Key`, except naturally idempotent PUT/DELETE operations. Reusing a key with a different body returns `409 idempotency_conflict`.
- Party, match, chat, attachment, and progression reads authorize membership on every request; a missing and unauthorized resource both return privacy-safe `404 not_found` where revealing existence would leak information.
- Cursor pages default to 20 and allow 1–100. Cursors are opaque, versioned, and scoped to the caller/query.

## Stable Enums

```text
playlist: casual | ranked
format: solo | duo | squad
mode: casual_solo | casual_duo | casual_squad |
      ranked_solo | ranked_duo | ranked_squad
legacy join mode: ranked_standard -> ranked_solo
party status: forming | queued | in_match | closed
match status: matched | active | completed | cancelled | failed_to_start
match result: team_one_win | team_two_win | draw | forfeit | abandoned | cancelled
queue status: not_queued | searching | matched | temporarily_unavailable
```

## Party HTTP API

### `POST /parties`

Create a forming Duo or Squad party led by the caller.

Request:

```json
{ "format": "duo" }
```

Response `201 PartyResponse`. Errors: `409 active_party_conflict`, `409 active_queue_or_match`, `422 unsupported_format`.

### `GET /parties/current`

Return the caller's current active party or `{ "party": null }` with `200`.

### `GET /parties/{partyId}`

Return an authorized current snapshot. Active members only.

### `POST /parties/{partyId}/invites`

Leader creates a 15-minute accepted-friend invitation.

```json
{ "user_id": "uuid" }
```

Response `201 PartyInviteResponse`. Errors: `403 leader_required`, `404 friend_unavailable` (covers missing/inactive/not-friend/blocked), `409 party_full`, `409 already_invited`, `409 target_busy`.

### `GET /party-invites`

List caller's pending incoming invitations with inviter public profile and party format/capacity; expired invites are omitted.

### `POST /party-invites/{inviteId}/accept`

Accept atomically and return `PartyResponse`. Revalidates friendship/block, account, capacity, party version, and active membership.

### `POST /party-invites/{inviteId}/decline`

Idempotently decline; `204`.

### `PUT /parties/{partyId}/readiness/me`

```json
{ "ready": true }
```

Only `forming`; response `PartyResponse`.

### `DELETE /parties/{partyId}/members/me`

Leave a forming party. Leadership transfers when needed. `204`; queue/in-match states return `409 party_locked`.

### `DELETE /parties/{partyId}/members/{userId}`

Leader removes another member from a forming party; `204`.

### `DELETE /parties/{partyId}`

Leader closes a forming party and revokes invites; `204`. Queue must be left first; active match must be terminal.

### Party Shapes

```json
{
  "party": {
    "id": "uuid",
    "format": "squad",
    "capacity": 4,
    "status": "forming",
    "version": 7,
    "leader_user_id": "uuid",
    "active_match_id": null,
    "members": [
      {
        "user_id": "uuid",
        "display_name": "Player",
        "avatar_url": null,
        "ready": true,
        "is_leader": true,
        "joined_at": "2026-07-18T10:00:00Z"
      }
    ],
    "created_at": "2026-07-18T10:00:00Z",
    "updated_at": "2026-07-18T10:01:00Z"
  }
}
```

Invite responses never reveal block ownership or target private account data.

## Matchmaking HTTP API

Existing paths remain; request/response shapes become additive.

### `POST /matchmaking/queue`

Canonical request:

```json
{
  "playlist": "ranked",
  "format": "duo",
  "party_id": "uuid"
}
```

- Solo requires `party_id: null`/omitted and creates an implicit one-user ticket.
- Duo/Squad require a complete party matching format, caller as leader, all members ready, and exact current party version.
- Ranked also requires active season, completed/hidden standing for every user, and no more than two named-rank tiers between highest and lowest member. Placement users use hidden rating for this check.
- Deprecated request `{ "mode": "ranked_standard" }` is accepted as Ranked Solo. Supplying both legacy and canonical fields is `422 invalid_mode_selection` unless they are equivalent.

Response `202 MatchmakingStatusResponse`.

### `DELETE /matchmaking/queue`

Solo caller or party leader leaves a searching ticket; idempotent `204`. A finalized assignment returns `409 match_already_formed` plus safe match destination in the error details.

### `GET /matchmaking/status`

Any queued party member receives the shared ticket/match state. Reading renews the ticket lease but never changes original queue priority.

```json
{
  "status": "searching",
  "queue": {
    "ticket_id": "opaque",
    "playlist": "ranked",
    "format": "duo",
    "mode": "ranked_duo",
    "party_id": "uuid",
    "search_started_at": "2026-07-18T10:00:00Z",
    "lease_expires_at": "2026-07-18T10:00:30Z",
    "rating_window": { "minimum": 900, "maximum": 1100 }
  },
  "match": null
}
```

Matched shape:

```json
{
  "status": "matched",
  "queue": null,
  "match": {
    "match_id": "uuid",
    "game_id": "uuid",
    "playlist": "ranked",
    "format": "duo",
    "mode": "ranked_duo",
    "formed_at": "2026-07-18T10:01:00Z",
    "destination": "/matches/uuid"
  }
}
```

The current response's `mode`, `match_id`, `game_id`, `formed_at`, and `destination` fields remain for compatible consumers.

## Match And Game HTTP API

### `GET /matches/{matchId}`

Return the caller-authorized canonical snapshot used for first render and version-gap recovery.

```json
{
  "match": {
    "id": "uuid",
    "game_id": "uuid",
    "playlist": "ranked",
    "format": "duo",
    "status": "active",
    "result": null,
    "team_size": 2,
    "viewer": {
      "user_id": "uuid",
      "game_player_id": "uuid",
      "team_slot": 1,
      "submitted": true,
      "can_chat": true,
      "allowed_spectate_player_ids": ["uuid"]
    },
    "teams": [
      {
        "slot": 1,
        "score": 8200,
        "players": [
          {
            "game_player_id": "uuid",
            "user_id": "uuid",
            "display_name": "Player",
            "status": "active",
            "submitted": true,
            "total_score": 4100
          }
        ]
      }
    ],
    "round": {
      "id": "uuid",
      "number": 2,
      "status": "active",
      "starts_at": "2026-07-18T10:02:00Z",
      "ends_at": "2026-07-18T10:03:00Z",
      "media": { "type": "panorama", "panorama_id": "safe-provider-ref" },
      "submitted_count": 2,
      "eligible_count": 4
    },
    "team_markers": [],
    "realtime_version": 42,
    "formed_at": "2026-07-18T10:00:00Z"
  }
}
```

Before reveal, opponent player totals may be team-level only and no guess coordinates, distance, accuracy, bonus, or answer fields are returned. After a round closes, snapshot may include `last_round_result` with all players' revealed results.

### `POST /games/{gameId}/rounds/{roundId}/guesses`

Existing path; request unchanged (`latitude`, `longitude`) and `Idempotency-Key` required. Multiplayer response changes additively and makes reveal nullable:

```json
{
  "guess": {
    "id": "uuid",
    "latitude": 1.23,
    "longitude": 4.56,
    "distance_meters": 12345,
    "accuracy_score": 4020,
    "speed_bonus": 101,
    "score": 4121,
    "submitted_at": "2026-07-18T10:02:25Z",
    "timed_out": false
  },
  "round_completed": false,
  "game_completed": false,
  "submitted_count": 3,
  "eligible_count": 4,
  "actual_location": null,
  "max_accuracy_score": 5000,
  "max_speed_bonus": 250,
  "outcome": "submitted"
}
```

When the accepted guess closes the round, `round_completed=true` and `actual_location` is present. Solo/daily response compatibility aliases (`max_score`, `score_percent`) remain until their clients migrate. A multiplayer submit never reveals the answer while `round_completed=false`.

### `GET /matches/{matchId}/rounds/{roundId}/results`

Available only after the round is completed and to participants. Returns answer, every participant's revealed guess/timeout, accuracy, bonus, score, and both team totals. `409 round_not_revealed` while active.

### `GET /matches/{matchId}/results`

Terminal results include all round results and:

```json
{
  "result": "team_one_win",
  "winner_team_slot": 1,
  "teams": [{ "slot": 1, "score": 42100, "players": [] }],
  "progression": {
    "applied": true,
    "old_rating": 980,
    "base_delta": 17,
    "abandon_penalty": 0,
    "total_delta": 17,
    "new_rating": 997,
    "old_rank": { "code": "navigator_3" },
    "new_rank": { "code": "navigator_3" },
    "placement": null,
    "top500_position": null
  }
}
```

Casual returns `progression: { "applied": false, "reason": "casual" }`. Ranked returns `202 progression_pending` only during a recoverable finalization retry; reads retry finalization and eventually return the exact stored change.

### `POST /matches/{matchId}/leave`

Explicitly leave/abandon. Active Ranked: team forfeits, caller gets visible extra abandon penalty, opponents receive a normal win only. Active Casual: team forfeits without rating. Terminal replay is idempotent `204`.

## Team Chat And Moderation API

These paths exist only for Duo/Squad matches.

### `POST /matches/{matchId}/messages`

```json
{
  "client_message_id": "uuid",
  "text": "I think this is northern Spain",
  "file_id": null
}
```

Text is trimmed; either non-empty text or one ready team-chat file is required. Response `201 TeamMessageResponse`; idempotent replay returns `200`. Limits: 10/10 seconds and 60/minute per user plus five image messages per user per match.

### `GET /matches/{matchId}/messages?after={sequence}&limit=50`

Return only viewer-team messages in ascending sequence, excluding muted authors. Access is limited to the active match and 15 minutes after terminal completion. Attachment metadata never contains a storage key or raw URL.

```json
{
  "data": [
    {
      "id": "uuid",
      "sequence": 120,
      "author": { "user_id": "uuid", "display_name": "Player" },
      "text": "Look at the road lines",
      "attachment": { "file_id": "uuid", "content_type": "image/jpeg", "size_bytes": 123456 },
      "status": "active",
      "created_at": "2026-07-18T10:02:00Z"
    }
  ],
  "page": { "limit": 50, "next_cursor": null }
}
```

### `GET /matches/{matchId}/messages/{messageId}/attachment`

Same-team access during the active match or 15-minute result session; returns a private signed URL to the sanitized derivative with ≤5-minute expiry. Never returns the raw upload.

### `PUT /matches/{matchId}/mutes/{userId}` / `DELETE ...`

Idempotently mute/unmute a current teammate; `204`.

### `POST /matches/{matchId}/messages/{messageId}/report`

```json
{ "reason": "harassment" }
```

Response `202`; idempotent per reporter/message. The target is not notified through match events.

## Upload API Additions

Existing `POST /uploads` accepts additive fields:

```json
{
  "file_name": "clue.webp",
  "content_type": "image/webp",
  "size_bytes": 123456,
  "purpose": "team_chat",
  "context_id": "match-uuid"
}
```

- Omitting `purpose` preserves existing `general` behavior.
- Team chat requires active same-match authorization, JPEG/PNG/WebP, and ≤5 MB.
- Existing `POST /uploads/complete` performs synchronous sanitization for team-chat purpose and returns only ready sanitized file metadata. Rejection returns `422 unsafe_image` or `422 invalid_image`; raw cleanup is guaranteed.
- Existing owner-only `/files/{fileId}/signed-url` remains owner-only. Teammates must use the authorized message attachment endpoint.

## Competitive API

### `GET /competitive/profile`

```json
{
  "season": {
    "id": "uuid",
    "sequence": 3,
    "name": "Season 3",
    "starts_at": "2026-07-18T00:00:00Z",
    "ends_at": "2026-10-10T00:00:00Z",
    "reset": { "anchor": 800, "factor_percent": 50 },
    "top500_min_matches": 25
  },
  "profile": {
    "rating": 2034,
    "placements_completed": 5,
    "placements_required": 5,
    "rank": { "code": "world_legend", "name_key": "worldLegend", "division": null },
    "standard_rank": { "code": "geo_master_1", "division": 1 },
    "division_progress": 34,
    "matches_played": 52,
    "wins": 31,
    "losses": 18,
    "draws": 3,
    "abandons": 0,
    "peak_rating": 2050,
    "top500_position": 412,
    "last_change": { "match_id": "uuid", "delta": 14 }
  }
}
```

Before placements finish, `rating` and `rank` are null, while placement progress and last result remain visible.

### `GET /competitive/leaderboard?limit=50&cursor=...`

Returns active-season eligible standings ordered by the specified tie-break. Each entry includes position, public profile, rating, wins, matches, standard rank, and `world_legend` for positions ≤500. Default/max limit 50/100. Response includes viewer entry separately if outside the page.

### `GET /competitive/history?limit=20&cursor=...`

Returns the caller's rating changes newest first with match playlist/format/outcome, old/new rating, base delta, abandon penalty, and rank transition.

### `GET /competitive/seasons`

Returns active and closed season summaries. Closed entries include the caller's ending/peak rank and final position if eligible.

### `GET /competitive/seasons/{seasonId}/leaderboard`

Frozen closed-season leaderboard, limited to final top 500. Active season redirects semantically to the current leaderboard shape.

## Realtime Ticket HTTP API

### `POST /realtime/tickets`

```json
{ "channel_kind": "match", "channel_id": "uuid" }
```

After membership authorization, returns `201`:

```json
{
  "ticket": "opaque-one-time-value",
  "expires_at": "2026-07-18T10:00:30Z",
  "websocket_url": "wss://api.example/realtime/matches/uuid"
}
```

Ticket is bound to user/channel, consumed once, and not refreshable. `party` channel is supported with the same endpoint.

OpenAPI path: `POST /realtime/tickets` (schemas `RealtimeTicketRequest` / `RealtimeTicketResponse`). Marker WebSocket commands use `round.marker.set` with team-scoped fanout (`round.marker_changed`); see WebSocket Protocol below. Team chat HTTP paths are also documented under Team Chat And Moderation API and mirrored in `backend/openapi/openapi.yaml`.

## WebSocket Protocol

Endpoints outside `/api/v1`:

```text
GET /realtime/parties/{partyId}
GET /realtime/matches/{matchId}
```

Origin allowlist is configured; ticket authorization is mandatory. The browser offers `geoguess.v1` and `ticket.{opaque-one-time-value}` in `Sec-WebSocket-Protocol`; the server consumes the ticket atomically and selects only `geoguess.v1`. Tickets never appear in URLs or logs. Protocol version is `1`.

### Server Event Envelope

```json
{
  "protocol_version": 1,
  "event_id": "uuid",
  "type": "round.guess_locked",
  "channel_kind": "match",
  "channel_id": "uuid",
  "game_id": "uuid",
  "round_id": "uuid",
  "occurred_at": "2026-07-18T10:00:00Z",
  "version": 43,
  "payload": {}
}
```

Known party events:

```text
party.snapshot, party.member_joined, party.member_left, party.member_removed,
party.leader_changed, party.readiness_changed, party.queue_changed,
party.match_assigned, party.closed, channel.error
```

Known match events:

```text
match.snapshot, match.started, match.player_disconnected, match.player_reconnected,
match.player_forfeited, round.started, round.guess_locked, round.marker_changed,
round.spectate_target_changed, round.view_changed, round.ended,
round.results_revealed, chat.message_created, chat.message_removed,
chat.mute_changed, match.completed, competitive.rating_applied, channel.error
```

Events are audience-filtered by the server. Clients must discard unknown event types, ignore events at/below their current version, and fetch the HTTP snapshot on a version gap.

### Client Command Envelope

```json
{
  "protocol_version": 1,
  "command_id": "uuid",
  "type": "round.marker.set",
  "expected_version": 42,
  "payload": { "round_id": "uuid", "latitude": 1.2, "longitude": 3.4 }
}
```

Supported match commands:

- `presence.heartbeat`: no payload; at most every 20 seconds.
- `round.marker.set`: round/lat/lng; at most 2/s; rejected after caller guess locks.
- `round.view.update`: round plus `{ panorama_id?, heading, pitch, zoom }` or provider-safe image pan/zoom; at most 4/s and only material changes. Map/marker/guess/score/answer/cursor fields are rejected. Non-material updates are accepted without fanout (`changed: false`). Material updates publish `round.view_changed` only to currently authorized submitted spectators (Solo: submitted opponents watching this navigator; Duo/Squad: submitted teammates watching this navigator). Scene payload never includes map, marker, cursor, coordinates, score, or answer.
- `round.spectate.select`: `{ target_game_player_id?, round_id? }`; caller must have submitted on an active round. Server resolves the preferred target against `allowed_spectate_player_ids` policy and applies automatic fallback to the first allowed target when preferred is missing or no longer eligible. Success acks with `{ target_game_player_id, allowed_spectate_player_ids, fallback, version }` and publishes private `round.spectate_target_changed` to the caller only. Denials use privacy-safe `spectate_forbidden` without explaining roster reasons.

### Spectating Policy (server-enforced)

| Format | Viewer requirement | Allowed targets | Ends when |
|--------|--------------------|-----------------|-----------|
| Solo | Guess locked/submitted | Unsubmitted, active opponent only | Target submits/disconnects/leaves, or round closes |
| Duo / Squad | Guess locked/submitted | Unsubmitted, active same-team teammates only (never opponents) | Target submits/disconnects/leaves/abandons, or round closes |

Unsubmitted viewers cannot spectate. Snapshot field `viewer.allowed_spectate_player_ids` lists currently allowed game-player IDs (empty when ineligible). Clients must not widen targets locally.

### Provider-safe scene schema (`round.view.update` / `round.view_changed`)

```json
{
  "round_id": "uuid",
  "user_id": "uuid",
  "panorama_id": "optional-provider-ref",
  "heading": 120.0,
  "pitch": -5.0,
  "zoom": 1.5,
  "version": 43
}
```

Optional image-mode fields: `image_pan_x`, `image_pan_y`, `image_zoom`. Forbidden keys (rejected): `latitude`, `longitude`, `lat`, `lng`, `marker`, `markers`, `map`, `cursor`, `guess`, `guesses`, `score`, `answer`, `actual_location`, and any other private client fields.

### Spectate target event (`round.spectate_target_changed`)

```json
{
  "round_id": "uuid",
  "target_game_player_id": "uuid-or-null",
  "allowed_spectate_player_ids": ["uuid"],
  "fallback": true
}
```

Delivered only to the spectator user. Round close clears allowed targets (clients return to waiting/results via snapshot/`round.ended` / `round.results_revealed`).

Chat, guesses, uploads, party membership, queue changes, and reports remain authenticated HTTP commands. WebSocket command acknowledgements use `command.accepted` or private `channel.error` with the original command ID. Duplicate command IDs replay the prior acknowledgement without republishing.

### Connection Behavior

- First message is an authorized snapshot at current version.
- Server ping interval 20 seconds; presence TTL/reconnect grace 90 seconds.
- Client reconnect backoff: 1, 2, 4, 8 seconds then 10-second cap, ±20% jitter; obtain a new ticket each attempt.
- Outbound queue is bounded. A slow client is closed with application code `4008`; unauthorized/expired ticket uses `4003`; version recovery uses HTTP snapshot.
- Event payloads are compact and must not contain access tokens, raw ticket values, storage keys, answer coordinates before reveal, opponent chat/markers, or precise private guesses.

## Stable Error Codes

```text
unauthorized, forbidden, not_found, validation_failed, rate_limited,
idempotency_conflict, account_ineligible, active_party_conflict,
active_queue_or_match, leader_required, party_full, party_incomplete,
party_not_ready, party_locked, friend_unavailable, target_busy,
unsupported_playlist, unsupported_format, invalid_mode_selection,
party_rating_spread, season_unavailable, content_unavailable,
match_already_formed, match_not_active, round_not_active,
round_not_revealed, already_guessed, guess_locked, round_closed,
spectate_forbidden, teammate_required, chat_unavailable,
message_too_long, attachment_not_ready, invalid_image, unsafe_image,
image_service_unavailable, progression_pending, temporarily_unavailable
```

Client catalogs map codes to localized English/Arabic messages; backend strings remain diagnostic fallbacks and never encode hidden state.
