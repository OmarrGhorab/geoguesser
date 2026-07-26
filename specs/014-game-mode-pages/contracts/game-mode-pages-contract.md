# Contract: Game Mode Pages

## Contract authority

backend/openapi/openapi.yaml remains the machine-readable API authority. This
document maps product pages to existing and planned operations and records the
contract corrections required before frontend integration.

All API paths below are relative to /api/v1. Unsafe operations require the
existing CSRF protection and authenticated or guest session stated by OpenAPI.
Idempotency keys are stable for one user intent, 16–128 characters where
required, and reused for network retries only.

## Page and route matrix

| Frontend route | Allowed selection/state | Initial authoritative reads | Primary commands | Destination |
|---|---|---|---|---|
| /[locale]/play?mode=solo | solo | GET /maps | POST /games, POST /games/{id}/start | /[locale]/games/{id} |
| /[locale]/play?mode=quick_play | quick_play | optional map summary | POST /games/quick-play | /[locale]/games/{id} |
| /[locale]/play?mode=practice | practice | GET /maps | POST /games, POST /games/{id}/start | /[locale]/games/{id} |
| /[locale]/daily-mission | daily | GET /challenges/daily | POST /challenges/daily/attempts | existing Daily play route |
| /[locale]/games/[gameId] | solo, quick_play, practice; dispatcher for legacy/daily | GET /games/{id}, GET current round, Practice history as needed | guess, next Practice round, end Practice | generic result or semantic redirect |
| /[locale]/games/[gameId]/results | solo, quick_play, ended Practice | GET /games/{id}/results or Practice history | none | replay or mode selection |
| /[locale]/rooms | party_lobby | GET /maps | POST /rooms, POST /rooms/join | /[locale]/rooms/{code} |
| /[locale]/rooms/[roomCode] | party_lobby/private_room recovery | GET /rooms/{code}, game/round result when active | settings, ready, start, kick, self-leave, guess | same stateful URL |
| /[locale]/matchmaking?mode=... | six canonical queue modes | GET /matchmaking/status; GET /parties/current for team modes | create party, queue join/leave | server match destination |
| /[locale]/parties/[partyId] | casual/ranked Duo/Squad preparation | GET /parties/{id}, invites | invite, accept/decline, ready, kick, leave, disband, queue | matchmaking or match |
| /[locale]/matches/[matchId] | six canonical match modes | GET /matches/{id}; ticket; round results after reveal | guess, leave, chat/mute/report | result page |
| /[locale]/matches/[matchId]/results | terminal casual/ranked | GET /matches/{id}/results | none | replay or mode selection |

The mode query selects presentation only. Existing queue, room, game, or match
state wins over a conflicting query and the UI offers an explicit safe exit or
resume action.

## Existing game operations

- POST /games: create solo or practice only.
- GET /games/{gameId}: participant-authorized state.
- POST /games/{gameId}/start: start a pending owner game.
- GET /games/{gameId}/rounds/current: reveal-safe current round.
- POST /games/{gameId}/rounds/{roundId}/guesses: idempotent guess.
- POST /games/{gameId}/rounds/{roundId}/timeout: Daily deadline transition.
- GET /games/{gameId}/results: durable final result.
- POST /games/{gameId}/rounds/next: Practice-only, required durable key.
- GET /games/{gameId}/rounds?cursor=&limit=: Practice-only bounded history,
  default 20 and maximum 100.
- POST /games/{gameId}/end: Practice-only explicit terminal transition.

Required correction: CreateGameRequest.mode becomes a narrow enum containing
solo and practice, rather than referencing every GameMode.

## New Quick Play operation

### POST /games/quick-play

Session: guest or registered. CSRF required. Idempotency-Key required.

Request body: empty object or omitted.

Server defaults:

- mode: quick_play
- map_id: QUICK_PLAY_DEFAULT_MAP_ID, validated as active/public/eligible
- round_count: 5
- timer_seconds: 60
- scoring/reveal/progression policy: Solo-compatible

Success: 201 with a GameResponse whose game is already active and includes the
resolved map, round count, timer, current round number, and timestamps. The
client may immediately GET the current round.

Errors:

- 400 invalid request/header
- 401 no usable session
- 403 CSRF failure
- 409 idempotency_conflict
- 422 no eligible configured map/locations
- 429 rate limited
- 503 dependency unavailable

Same actor/key/request returns the original game. A different request hash for
the same actor/key returns 409. The map setting is never trusted from the client.

## Daily operations

- GET /challenges/daily returns challenge, attempt state, last completed game,
  reset countdown, daily_games_played, and daily_games_total.
- POST /challenges/daily/attempts starts or replays the active daily attempt.
- Daily gameplay uses ordinary current-round, guess, timeout, and result
  operations.

The UI renders progress out of five. A completed attempt with fewer than five
games is a successful intermediate state with a Play next game action, not a
completed-day state.

## Party Lobby operations

Existing:

- POST /rooms
- POST /rooms/join
- GET /rooms/{roomCode}
- PUT /rooms/{roomCode}/settings
- POST /rooms/{roomCode}/ready
- POST /rooms/{roomCode}/start
- DELETE /rooms/{roomCode}/players/{playerId}
- WebSocket /realtime/rooms/{roomCode}

New:

- DELETE /rooms/{roomCode}/players/me
  - 204 when a non-host member leaves or has already left.
  - 409 host_action_required when a lobby host must cancel instead.
  - Privacy-safe 404 for missing/unauthorized rooms.
- DELETE /rooms/{roomCode}
  - Host-only cancellation while status is lobby; idempotent 204 once cancelled.
  - Returns 409 after the hosted game has started; active departure follows the
    established disconnect/grace and terminal policy.
- GET /games/{gameId}/rounds/{roundId}/results
  - Mount the existing shared-round result handler.
  - Participant-only; 409 round_not_revealed before shared closure.
  - Returns answer and bounded participant results only after reveal.

Room WebSocket origins use ROOM_REALTIME_ALLOWED_HOST, not hard-coded localhost
patterns. Cookies remain secure/same-site according to deployment configuration.
Events are notification hints and carry a monotonic room/game version.

## Premade party operations

- POST /parties with format duo or squad and required Idempotency-Key.
- GET /parties/current and GET /parties/{partyId}.
- POST /parties/{partyId}/invites.
- GET /party-invites.
- POST accept/decline invite operations.
- PUT /parties/{partyId}/readiness/me.
- DELETE self/member/disband operations.

Team matchmaking reads the latest Party.version and sends both party_id and
party_version. Only the leader can queue an exact, eligible, ready roster.

## Matchmaking operations

- POST /matchmaking/queue with canonical playlist and format.
- DELETE /matchmaking/queue.
- GET /matchmaking/status.

Canonical join request:

    {
      "playlist": "casual | ranked",
      "format": "solo | duo | squad",
      "party_id": "uuid or omitted",
      "party_version": "integer or omitted"
    }

party_id and party_version are omitted for Solo and required for Duo/Squad.
Legacy mode input remains read-compatible but new clients always send canonical
playlist/format.

Status is one of not_queued, searching, matched, or temporarily_unavailable.
When matched, the client follows the server destination and localizes it; it
does not synthesize a destination from the selected query.

Queue join is naturally idempotent through the active-ticket invariant. OpenAPI
must not claim required key-bound idempotency unless the handler actually
enforces that key. Leave returns a safe terminal or refreshable conflict when a
match has already formed.

## Matchplay and realtime operations

- GET /matches/{matchId}: authorized canonical snapshot.
- GET /matches/{matchId}/rounds/{roundId}/results: revealed shared result.
- GET /matches/{matchId}/results: terminal result; ranked may return 202
  progression_pending.
- POST /matches/{matchId}/leave: explicit forfeit/leave.
- GET/POST /matches/{matchId}/messages and mute/report operations for team modes.
- POST /realtime/tickets with channel_type party or match and channel_id.
- WebSocket /realtime/matches/{matchId} using geoguess.v1 plus one-use ticket
  subprotocol; tickets never appear in a URL or log.

The active match client always hydrates from GET snapshot before subscribing.
On an event version gap or reconnect it pauses optimistic transitions, refetches
the snapshot, and resumes. No opponent coordinates, individual guesses,
distance, accuracy, speed bonus, or hidden answers are rendered before the
authoritative reveal.

## Frontend mutation boundary

Low-frequency forms use Server Actions returning typed expected-error values:
game/Quick Play create, game start, Practice next/end, room create/join/settings/
ready/start/leave, party commands, and queue join/leave.

Latency-sensitive browser interactions use same-origin Route Handlers:

- game and match guess submission
- Daily timeout
- realtime ticket issuance
- chat and attachment operations

Route Handlers forward existing session/CSRF context, validate Zod input/output,
set no-store, attach/reuse the caller's idempotency key, and never expose backend
cookies or internal error bodies.

## Stable client error categories

| API category | User state/action |
|---|---|
| unauthenticated | Sign in; preserve safe return destination |
| forbidden/ineligible | Explain requirement; link to profile/party/mode selection |
| not_found | Generic unavailable state; never distinguish unauthorized |
| conflict/stale_version | Refresh authoritative snapshot; preserve safe input |
| idempotency_conflict | Stop retry; explain duplicate intent; refresh resource |
| already_active/already_matched | Resume authoritative destination |
| full/expired/cancelled | Return to room selection |
| round_not_revealed | Remain in submitted/reconnecting state and retry snapshot |
| rate_limited | Respect Retry-After and disable repeated submit |
| temporarily_unavailable | Degraded state with bounded manual retry |
| validation | Field-level localized messages |
| internal/dependency | Safe generic error with request correlation ID only |

## Contract verification

- Validate backend/openapi/openapi.yaml in CI.
- Add handler/service contract tests for each new/corrected operation.
- Add Zod fixture tests for success, nullable fields, legacy recovery, and every
  stable status/error union.
- Assert pre-reveal payloads and unauthorized errors contain no hidden data.
- Assert every one of the eleven registry entries maps to one canonical contract
  and destination.
