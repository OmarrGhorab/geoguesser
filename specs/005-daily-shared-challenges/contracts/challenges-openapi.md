# Contract: Daily And Shared Challenges

The machine-readable source of truth remains `backend/openapi/openapi.yaml`. This document records the Phase 05 contract expectations that implementation must add or verify there.

## Authentication Model

- Challenge metadata for active daily and public shared challenges can be read by guests or signed-in users.
- Starting attempts, completing attempts, streak updates, and mission progress require an existing guest or signed-in session.
- Public daily leaderboard entries require signed-in account identity.
- Guest attempts can return personal results and session-scoped progress, but must not appear as durable public account leaderboard entries.

## Idempotency

All mutating endpoints should accept `Idempotency-Key` where repeat submissions are likely:

- Create shared challenge.
- Start or resume challenge attempt.
- Finalize challenge attempt/result if exposed as a distinct action.
- Claim mission reward/status if claiming exists.

Identical retry keys for the same actor and request body should replay the original successful result. Same-key conflicting requests should return conflict.

## Endpoints To Add

### `GET /challenges/daily`

Returns today's daily challenge metadata for the canonical reset boundary.

**Query parameters**
- `date` optional date for history/admin validation where allowed; defaults to current challenge date.

**Response**
- `challenge.id`
- `challenge.type = daily`
- `challenge.seed`
- `challenge.challenge_date`
- `challenge.reset_starts_at`
- `challenge.reset_ends_at`
- `challenge.map`
- `challenge.settings`
- `attempt_state`
- `streak`
- `missions_summary`
- `leaderboard_summary`
- `countdown`

**Errors**
- `404` when no daily challenge can be materialized.
- `422` when map pool/settings cannot provide enough unique locations.

### `POST /challenges/daily/attempts`

Starts or resumes the current actor's daily challenge attempt.

**Request**
- Optional idempotency key header.

**Response**
- `attempt.id`
- `challenge.id`
- linked game summary or playable game start response.
- locked settings.
- current state.

**Errors**
- `409` when the actor already has a completed leaderboard-credit attempt and replay is not possible.
- `422` when challenge is unavailable or no longer playable.

### `POST /challenges/shared`

Creates a shared fixed-seed challenge link from an allowed map and settings.

**Request body**
- `map_id`
- `round_count`
- `timer_seconds`
- movement/settings flags
- optional display label

**Response**
- `challenge.id`
- `share_url` or `share_code`
- `seed`
- map summary
- locked settings

**Errors**
- `400` for invalid settings.
- `422` for insufficient unique locations.
- `409` for idempotency conflict.

### `GET /challenges/shared/{code}`

Loads shared challenge metadata by stable link/code.

**Response**
- same safe challenge metadata shape as daily challenge.
- current actor attempt state when session exists.
- spoiler-safe result availability.

**Errors**
- `404` for invalid or hidden shared challenge.
- `410` if an expiry policy exists and the challenge expired.

### `POST /challenges/{challengeId}/attempts`

Starts or resumes an attempt for a daily or shared challenge.

**Response**
- attempt state.
- linked solo game/current round entry point.
- locked settings.

**Errors**
- `403` for unauthorized access to non-public challenge.
- `409` for duplicate completed leaderboard-credit attempt.
- `422` for unavailable challenge.

### `GET /challenges/{challengeId}/results`

Returns the current actor's completed result summary and spoiler-safe comparison context.

**Response**
- challenge summary.
- attempt summary.
- total score.
- total distance.
- per-round results if visible.
- rank context.
- streak impact.
- mission progress updates.

**Errors**
- `403` when actor does not own the attempt/result.
- `404` when challenge or attempt is missing.
- `422` when results are not ready.

### `GET /challenges/{challengeId}/leaderboard`

Returns a paginated leaderboard for a completed or visible challenge.

**Query parameters**
- `limit`
- `cursor`

**Response**
- challenge summary.
- `entries[]`
- `page.limit`
- `page.next_cursor`
- current actor rank context where available.

**Visibility**
- Must not reveal spoiler-sensitive details to unfinished players while challenge is playable.

### `GET /missions`

Returns active missions and the current actor's progress.

**Response**
- active missions.
- progress state.
- completed/claimable/expired state.
- localized copy keys or display-safe labels.

### `POST /missions/{missionId}/claim`

Marks a completed mission reward/status as claimed if claiming exists in implementation.

**Response**
- updated mission progress.

**Errors**
- `409` when already claimed.
- `422` when not completed or expired.

### `GET /streaks/daily`

Returns the current actor's daily streak state.

**Response**
- current count.
- best count.
- last qualifying date.
- status.
- protection state.
- guest persistence limits when applicable.

## DTO Requirements

### Challenge Summary

- `id`
- `type`
- `seed`
- `challenge_date`
- `reset_starts_at`
- `reset_ends_at`
- `map`
- `settings`
- `status`
- `share_url` or `share_code` for shared challenges where visible.

### Locked Settings

- `round_count`
- `timer_seconds`
- `movement_rules`
- `scoring_version`
- any additional mode settings required by the current game loop.

### Attempt Summary

- `id`
- `challenge_id`
- `status`
- `leaderboard_eligible`
- `started_at`
- `completed_at`
- `total_score`
- `current_round_number`
- linked game id or game state entry point.

### Leaderboard Entry

- `rank`
- `display_name`
- `score`
- `completion_duration_ms`
- `completed_at`
- optional current-player marker.

### Streak Summary

- `current_count`
- `best_count`
- `last_completed_challenge_date`
- `status`
- `protection_state`
- `guest_limited`

### Mission Summary

- `id`
- `code`
- `title_key`
- `description_key`
- `mission_type`
- `current_value`
- `target_value`
- `status`
- `active_ends_at`
- reward/status metadata.

## OpenAPI Updates To Verify

- Verified on 2026-06-27: `backend/openapi/openapi.yaml` includes `Challenges`, `Missions`, and `Streaks` tags.
- Verified on 2026-06-27: all endpoint paths above are documented under `/api/v1` server scope.
- Verified on 2026-06-27: schemas exist for challenge metadata, challenge attempts, challenge leaderboards, missions, streaks, settings snapshots, and spoiler-safe result views.
- Verified on 2026-06-27: stable shared error responses are referenced for invalid settings, insufficient locations, duplicate/idempotency conflict, results not ready, and unavailable challenges through common BadRequest, Conflict, NotFound, Forbidden, and UnprocessableEntity responses.
- Verified on 2026-06-27: challenge metadata, current attempt, mission, streak, and leaderboard schemas do not expose hidden coordinates before result reveal.
- Verified on 2026-06-27: leaderboard endpoint documents `limit` and `cursor` query parameters and a `page` object.
- Verified on 2026-06-27: auth/session requirements are documented for guest and account behavior on each challenge, mission, and streak endpoint.

Validation command:

```powershell
npx pnpm@10.24.0 check:openapi
```

Result: PASS. Redocly emitted warnings for existing repository-wide style rules and for the documented Phase 05 path shape `/challenges/shared/{code}` beside `/challenges/{challengeId}/...`; the OpenAPI document is valid.
