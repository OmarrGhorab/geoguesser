# Contract: Solo Game API

The machine-readable source of truth remains `backend/openapi/openapi.yaml`. This document records the Phase 04 contract expectations that implementation must verify and update there.

## Auth and Safety

- All endpoints accept the owning registered user session or signed guest session.
- Unsafe methods require CSRF protection through the existing middleware.
- Retryable writes accept `Idempotency-Key`.
- Responses never return raw persistence models.
- Error responses use the shared error envelope with stable `error.code`.

## `POST /api/v1/games`

Creates a pending solo game and all round records.

**Request**

- `mode`: must be `solo` for this phase.
- `map_id`: active playable map id.
- `round_count`: integer 1 to 10, default 5 if the implementation supports defaults.
- `timer_seconds`: null or integer 10 to 600.
- Header `Idempotency-Key`: recommended for retry safety.

**Success**

- `201 Created`
- Body: `GameResponse`
- `game.status`: `pending`
- `game.current_round_number`: null before start.

**Errors**

- `400 validation_failed` for malformed coordinates, missing map, invalid mode, round count, or timer.
- `401 unauthorized` when no guest or registered session is available.
- `403 forbidden` when map access is denied.
- `409 conflict` for conflicting idempotency replay.
- `422 not_enough_locations` when the map cannot provide enough unique active locations.

## `GET /api/v1/games/{gameId}`

Returns game state visible to the owner.

**Success**

- `200 OK`
- Body: `GameResponse`
- For active games, includes `current_round_number`.
- Does not include actual hidden coordinates.

**Errors**

- `401 unauthorized`
- `403 forbidden` for non-owner access.
- `404 not_found` when hidden or missing.

## `POST /api/v1/games/{gameId}/start`

Starts a pending solo game.

**Request**

- Header `Idempotency-Key`: recommended.

**Success**

- `200 OK`
- Body: `GameResponse`
- `game.status`: `active`
- `game.started_at`: server timestamp.
- `game.current_round_number`: `1`.

**Errors**

- `401 unauthorized`
- `403 forbidden`
- `404 not_found`
- `409 game_already_started` for already active/completed game, unless the exact idempotent replay returns the original response.
- `422 invalid_game_transition` for non-startable state.

## `GET /api/v1/games/{gameId}/rounds/current`

Returns the current playable round.

**Success**

- `200 OK`
- Body: `CurrentRoundResponse`
- Includes round id, number, status, start/end timestamps, and media.
- Excludes true latitude, true longitude, `location_id`, provider refs that reveal the answer, and admin notes.

**State behavior**

- Before start: returns a stable not-started error or pending state as documented in OpenAPI.
- If a timed round expired, the backend may complete/reveal it before returning current state.
- If all rounds are complete, returns a completed-game state or directs clients to results through a stable error/code.

## `POST /api/v1/games/{gameId}/rounds/{roundId}/guesses`

Submits one guess for the owner in the current round.

**Request**

- Body latitude: -90 to 90.
- Body longitude: -180 to 180.
- Header `Idempotency-Key`: required or strongly recommended; implementation decision must be reflected in OpenAPI.

**Success**

- `200 OK`
- Body: `GuessResultResponse`
- Includes submitted guess, `distance_meters`, `score`, server `submitted_at`, and revealed actual location for that round.

**Idempotency**

- Same key and same request returns the original result.
- Same key with different coordinates returns `409 conflict`.
- No second guess may create or alter scoring after a first accepted guess.

**Errors**

- `400 validation_failed`
- `401 unauthorized`
- `403 forbidden`
- `404 not_found`
- `409 already_guessed` or `idempotency_conflict`
- `422 round_closed`, `round_not_current`, or `game_not_active`
- `429 rate_limited`

## `GET /api/v1/games/{gameId}/results`

Returns durable solo game results.

**Success**

- `200 OK`
- Body: `GameResultsResponse`
- Includes game summary, players, rounds, revealed locations, guesses, per-round score, per-round distance, and total score.

**Errors**

- `401 unauthorized`
- `403 forbidden`
- `404 not_found`
- `422 game_not_completed` if final results are requested before completion and the implementation chooses to withhold partial final results.

## OpenAPI Updates To Verify

- `CreateGameRequest.mode` should document that only `solo` is accepted in Phase 04 even if the enum is broader.
- `Idempotency-Key` requirement for create/start/guess should be explicit.
- `CurrentRoundResponse` must remain free of answer coordinates and `location_id`.
- `RoundResult` must include enough revealed answer information for final result display.
- Error codes for `not_enough_locations`, `round_closed`, `already_guessed`, `round_not_current`, `game_not_completed`, and `idempotency_conflict` should be represented in examples or descriptions.
