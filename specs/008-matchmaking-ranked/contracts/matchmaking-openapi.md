# Contract: Matchmaking And Ranked Foundations

## Contract Scope

This artifact defines the Backend Phase 9 updates for `backend/openapi/openapi.yaml`, backend DTOs, and frontend matchmaking types. It replaces the existing guest/quick-play placeholder while preserving the three established paths.

Base API prefix: `/api/v1`.

## Authentication And Security

- All matchmaking operations require a registered access-cookie session.
- Guest and anonymous sessions receive the shared unauthorized response.
- `POST` and `DELETE` require the existing CSRF cookie/header pair.
- The service revalidates that the registered account is active before queue entry and again before durable formation.
- Command and status routes use separate per-user rate-limit budgets.
- Responses never include another queued player, opponent identity, queue population, hidden location data, guesses, session data, email, or private profile fields.

## Status Schema

`MatchmakingStatusResponse`:

```json
{
  "status": "searching",
  "queue": {
    "mode": "ranked_standard",
    "search_started_at": "2026-07-11T12:00:00Z",
    "lease_expires_at": "2026-07-11T12:00:30Z"
  },
  "match": null
}
```

Required fields:

- `status`: `not_queued`, `searching`, `matched`, or `temporarily_unavailable`.
- `queue`: present only for `searching`; otherwise null.
- `match`: present only for `matched`; otherwise null.

`MatchmakingQueue`:

- `mode`: `ranked_standard`.
- `search_started_at`: original enqueue timestamp, stable across retry and lease renewal.
- `lease_expires_at`: current ephemeral lease expiry.

`MatchmakingMatch`:

- `match_id`: UUID.
- `game_id`: UUID.
- `mode`: `ranked_standard`.
- `formed_at`: durable match time.
- `destination`: locale-independent application path such as `/games/{game_id}`; clients apply locale-aware navigation.

Rules:

- `not_queued` has both details null.
- `searching` has queue details and null match.
- `matched` has match details and null queue.
- `temporarily_unavailable` has both details null and signals a recoverable uncertainty, not a confirmed absence.
- Unknown properties should be rejected on command requests and ignored defensively by clients on responses.

## Join Queue

```text
POST /api/v1/matchmaking/queue
```

Request:

```json
{
  "mode": "ranked_standard"
}
```

Rules:

- `mode` is required and accepts only approved modes; Backend Phase 9 supports `ranked_standard`.
- `region`, `map_id`, display name, rating, and opponent preferences are not accepted in this phase.
- First join and repeated/concurrent joins for the same valid entry return the same semantic searching state and preserve `search_started_at`.
- A successful join triggers one bounded formation attempt.

Responses:

- `202 Accepted`: search accepted or existing search returned; body is `MatchmakingStatusResponse` and may already be `matched` if synchronous formation completes.
- `400 Bad Request`: malformed JSON, unknown properties, or invalid mode shape.
- `401 Unauthorized`: anonymous or guest.
- `403 Forbidden`: disabled/deleted/ineligible account.
- `409 Conflict`: player already has another durable active assignment or conflicting active game that cannot be returned as the current match.
- `422 Unprocessable Entity`: recognized but currently unsupported competitive mode or unavailable configured content.
- `429 Too Many Requests`: command budget exceeded; includes `Retry-After`.
- `503 Service Unavailable`: queue state cannot safely be accepted because its ephemeral dependency is unavailable.

## Leave Queue

```text
DELETE /api/v1/matchmaking/queue
```

Rules:

- Idempotent when the player is absent or searching.
- If searching, removes only the caller's exact current entry.
- If pair claim already won the race, leave must not destroy the claim or durable assignment. The client follows with status to recover the authoritative outcome.
- Repeated leave requests never affect another player.

Responses:

- `204 No Content`: caller is not searching after the operation, including already absent.
- `401 Unauthorized`: anonymous or guest.
- `409 Conflict`: assignment is currently being finalized and cannot be cancelled as a queue entry; client should immediately refresh status.
- `429 Too Many Requests`: command budget exceeded; includes `Retry-After`.
- `503 Service Unavailable`: no durable assignment exists and ephemeral state cannot be changed safely.

## Get Status

```text
GET /api/v1/matchmaking/status
```

Rules:

- Checks active durable assignment before Redis queue state.
- While searching, renews the activity lease without changing original queue priority and performs at most one bounded match/recovery attempt.
- A durable match remains recoverable when Redis is unavailable.
- Stale or expired queue state returns `not_queued`, not `searching`.

Responses:

- `200 OK`: `MatchmakingStatusResponse` for all normal states, including `temporarily_unavailable` when a safe recoverable response is possible.
- `401 Unauthorized`: anonymous or guest.
- `429 Too Many Requests`: polling budget exceeded; includes `Retry-After`.
- `503 Service Unavailable`: neither durable nor ephemeral state can be resolved into a safe status response.

## Stable Error Codes

Use the existing error envelope and shared codes where possible. Add only UI-relevant matchmaking codes:

- `unsupported_matchmaking_mode`
- `matchmaking_account_ineligible`
- `matchmaking_already_assigned`
- `matchmaking_active_game_conflict`
- `matchmaking_claim_in_progress`
- `matchmaking_content_unavailable`
- `matchmaking_unavailable`
- shared `unauthorized`
- shared `forbidden`
- shared `invalid_request`
- shared `conflict`
- shared `rate_limited`
- shared `internal_error`

Error messages remain safe and generic; implementation details, Redis state, opponent identity, and account-status distinctions must not leak.

## Ranked Game Contract Impact

Existing game paths remain the gameplay destination:

- `GET /api/v1/games/{gameId}`
- `GET /api/v1/games/{gameId}/rounds/current`
- `POST /api/v1/games/{gameId}/rounds/{roundId}/guesses`
- `GET /api/v1/games/{gameId}/results`

Required behavior updates:

- Both ranked participants are authorized through their `game_players` membership.
- Ranked games cannot be created through the public solo-game create operation.
- Ranked round state is server-controlled and uses the multiplayer rule: advance after all eligible players submit or deadline expiry.
- A guess before scheduled round start is rejected.
- Pre-reveal game/round payloads retain existing hidden-coordinate guarantees.
- Results remain participant-only and include no future rating change in this phase.

Any schema changes required for a ranked countdown or multi-player result summary must update the main OpenAPI artifact in the same implementation slice.

## Frontend Contract

Add shared TypeScript shapes under `client/features/matchmaking/types.ts` matching the response union. Prefer a discriminated union so impossible queue/match combinations cannot be represented locally.

Add a localized `client/app/[locale]/games/[gameId]/page.tsx` destination backed by server-only game helpers and a narrow ranked-game Client Component for map interaction, countdown, guess submission, round refresh, and results. It must reuse the backend's existing game/round/guess/result contracts and must not trust client timers for acceptance.

Server-only helper operations in `client/lib/api/matchmaking.ts`:

- `joinMatchmaking(mode)`
- `leaveMatchmaking()`
- `getMatchmakingStatus()`

Server Actions in `client/features/matchmaking/actions.ts`:

- validate the single supported mode;
- invoke join/leave helpers;
- return stable serializable UI action states;
- never return raw backend error bodies or secrets.

Same-origin polling Route Handler:

```text
GET /api/matchmaking/status
```

- Calls the server-only backend status helper with request cookies.
- Returns the safe status union.
- Preserves meaningful `429`/`Retry-After` and safe availability failures.
- Is not a second canonical API and contains no business logic.

Polling client rules:

- Poll only while `searching`, approximately every 2 seconds.
- Never overlap status requests.
- Stop when `matched`, `not_queued`, or persistently unavailable.
- Honor `Retry-After` and apply bounded backoff for transient failures.
- Navigate to the locale-aware match destination only after a durable `matched` response.
- Announce matched/search/error changes through accessible live regions without stealing focus.
