# Research: Solo Game Loop

## Decision: Use PostgreSQL as the solo game source of truth

**Rationale**: Games, rounds, players, guesses, and final results are durable gameplay facts and already exist in `backend/migrations/00001_initial_schema.sql`. Keeping them in PostgreSQL preserves reloadability, auditability, and stable historical scoring.

**Alternatives considered**: Redis-only game state was rejected because final results must be durable and reloadable. Hybrid write-behind was rejected because it makes scoring and replay consistency harder for MVP.

## Decision: Keep Redis limited to ephemeral gameplay helpers

**Rationale**: Redis is appropriate for short-lived idempotency keys, current game/round cache entries, and rate-limit counters. Database uniqueness still protects one-guess-per-round and durable consistency.

**Alternatives considered**: Durable idempotency records in PostgreSQL were considered but are unnecessary for short-lived gameplay retries. Pure database idempotency was rejected for retry response replay because it cannot cheaply detect repeated same-body requests without extra state.

## Decision: Select all solo round locations at game creation

**Rationale**: The phase source requires round creation with no repeated locations per game. Selecting all locations in the create-game transaction allows a unique round sequence, avoids mid-game pool drift, and gives deterministic game state.

**Alternatives considered**: Selecting each round just-in-time was rejected because it complicates no-repeat guarantees and replayability. Selecting from the client was rejected because the backend must own fairness.

## Decision: Use existing map selection strategy and require enough unique active locations

**Rationale**: `maps.Repository.SelectLocations` already uses `locations.random_key`, map membership, and active status instead of `ORDER BY random()`. The games service can fail game creation with a domain error when fewer unique active locations are returned than requested.

**Alternatives considered**: Adding a new random query inside `games` was rejected because map membership and active-map filtering already belong to `maps`.

## Decision: Backend clock owns round timing

**Rationale**: Existing `internal/platform/clock` gives deterministic tests and production UTC time. `rounds.starts_at`, `rounds.ends_at`, and `rounds.revealed_at` are the durable timing source. Client timestamps are ignored.

**Alternatives considered**: Client deadline submission was rejected because it can be manipulated. Background-only round closure was rejected for MVP because request paths can close/reveal rounds when current state is read or a guess arrives.

## Decision: Use haversine distance and versioned exponential score

**Rationale**: Phase 8 defines MVP scoring as haversine distance plus `score = round(5000 * e^(-distanceKm / 1492))`, with a full-score threshold at 25 meters. Storing `games.scoring_version`, `guesses.distance_meters`, and `guesses.score` keeps historical results stable.

**Alternatives considered**: PostGIS distance calculation was rejected for MVP because haversine is simple, deterministic, fast, and sufficient for gameplay scoring. Client-calculated scoring was rejected because scoring must be server-authoritative.

## Decision: Treat duplicate guess retries as replay when the idempotency key and body match

**Rationale**: The API contract already includes `Idempotency-Key`. A repeated key with the same guess should return the stored result. A different body for the same completed round should return a conflict, while the database unique `(round_id, game_player_id)` remains the final one-guess guard.

**Alternatives considered**: Always returning conflict for any duplicate was rejected because it makes network retry UX brittle. Allowing the second guess to update the first was rejected because it violates fairness.

## Decision: Add `guesses` idempotency uniqueness if not already present

**Rationale**: The database design calls for optional unique `(game_player_id, idempotency_key)` where the key is not null. The current initial migration includes the column but no unique index. Adding a new Goose migration prevents duplicate replay rows across retry races.

**Alternatives considered**: Relying only on Redis was rejected because Redis expiration can remove the replay guard while the round remains durable. Relying only on `(round_id, game_player_id)` catches duplicates but does not distinguish safe retry from conflicting request body.

## Decision: Return DTOs only, never raw models

**Rationale**: Hidden coordinate safety depends on response shaping. Current-round responses must not include `round.location_id`, `locations.latitude`, `locations.longitude`, provider refs, or admin-only fields before reveal.

**Alternatives considered**: Reusing persistence models in responses was rejected because it risks hidden-coordinate leakage.

## Decision: Implement backend first, leave UI to future client phase

**Rationale**: The requested phase is in `docs/phases/backend/phase-04-solo-game-loop.md`. Backend contracts must still support client loading, disabled, error, reveal, and final-result states, but no Next.js files are needed now.

**Alternatives considered**: Implementing frontend screens in the same phase was rejected to keep scope aligned with the backend plan and avoid mixing framework-sensitive UI work into this server feature.
