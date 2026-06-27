# Data Model: Solo Game Loop

## Overview

The solo game loop uses existing durable tables from `backend/migrations/00001_initial_schema.sql`: `games`, `rounds`, `game_players`, `guesses`, `maps`, `locations`, and `map_locations`. Implementation should add only targeted migrations for missing constraints or indexes.

## Entity: Solo Game

**Table**: `games`

**Fields used**

- `id`: game identifier.
- `mode`: must be `solo` for this feature.
- `status`: `pending`, `active`, `completed`, `abandoned`, or `cancelled`.
- `map_id`: selected playable map.
- `created_by_user_id`: registered creator when the owner is a user; null for guests.
- `round_count`: number of rounds, 1 to 10.
- `timer_seconds`: optional server-owned timer duration, 10 to 600 seconds when present.
- `scoring_version`: formula version, default `1`.
- `total_score`: aggregate score for the solo game/player.
- `started_at`: set when game starts.
- `completed_at`: set when final round completes.
- `created_at`, `updated_at`: durability and ordering timestamps.

**Relationships**

- Has one solo `Game Player`.
- Has many `Round` records ordered by `round_number`.
- Belongs to a `Map`.

**Validation**

- Only the owner can read or mutate the game.
- Solo games must have exactly one active player.
- A game can start only from `pending`.
- A game completes when the last round completes.

**State transitions**

```text
pending -> active -> completed
pending -> cancelled
active -> abandoned
```

## Entity: Game Player

**Table**: `game_players`

**Fields used**

- `id`: participant identifier.
- `game_id`: owning game.
- `user_id`: registered player identity when present.
- `guest_identity_hash`: guest player identity hash when present.
- `display_name`: snapshot used in results.
- `role`: `player` for solo.
- `status`: `active`, `disconnected`, `left`, or `kicked`.
- `total_score`: sum of accepted guess scores.
- `joined_at`, `left_at`: membership timestamps.

**Relationships**

- Belongs to one solo game.
- Has many guesses.

**Validation**

- At least one of `user_id` or `guest_identity_hash` must be present.
- For solo MVP, exactly one active game player belongs to the game.
- Owner matching uses registered user id or guest identity hash from the resolved request session.

## Entity: Round

**Table**: `rounds`

**Fields used**

- `id`: round identifier.
- `game_id`: owning game.
- `location_id`: selected hidden answer location.
- `round_number`: 1-based order.
- `status`: `pending`, `active`, `completed`, or `cancelled`.
- `starts_at`: server timestamp when active.
- `ends_at`: server deadline when timed.
- `revealed_at`: timestamp when answer can be shown.
- `created_at`: durability timestamp.

**Relationships**

- Belongs to one game.
- Belongs to one location.
- Has at most one guess for the solo player.

**Validation**

- `(game_id, round_number)` is unique.
- Location ids must not repeat within the same game.
- Current round is the active round, or the next pending round once the previous round is completed.
- Current-round DTOs must not expose `location_id` or actual coordinates before reveal.

**State transitions**

```text
pending -> active -> completed
active -> cancelled
```

## Entity: Guess

**Table**: `guesses`

**Fields used**

- `id`: guess identifier.
- `round_id`: guessed round.
- `game_player_id`: submitting player.
- `latitude`: submitted latitude.
- `longitude`: submitted longitude.
- `distance_meters`: server-computed distance.
- `score`: server-computed score.
- `idempotency_key`: optional retry key from `Idempotency-Key`.
- `submitted_at`: server timestamp.
- `created_at`: durability timestamp.

**Relationships**

- Belongs to one round.
- Belongs to one game player.

**Validation**

- One guess per `(round_id, game_player_id)`.
- Latitude must be between -90 and 90.
- Longitude must be between -180 and 180.
- Distance must be non-negative.
- Score must be between 0 and 5000.
- A missing guess scores 0 when a timed round closes without submission.
- Add or verify partial unique `(game_player_id, idempotency_key)` where `idempotency_key IS NOT NULL`.

## Entity: Location

**Table**: `locations`

**Fields used**

- `id`: location identifier.
- `latitude`, `longitude`: actual answer coordinates, hidden before reveal.
- `country_code`, `region`, `locality`: reveal/result location metadata.
- `provider`, `provider_ref`, `attribution`, `heading`: media metadata used by existing location/media flow.
- `status`: must be active for selection.
- `random_key`: indexed selection support.

**Relationships**

- Belongs to maps through `map_locations`.
- Can be selected by many rounds across different games.

**Validation**

- Only active locations in active selected maps are eligible.
- No repeated location within a single solo game.
- Current-round responses expose media only, never true coordinates.

## Entity: Map

**Tables**: `maps`, `map_locations`

**Fields used**

- `maps.id`, `slug`, `name`, `visibility`, `access_tier`, `difficulty`, `status`.
- `map_locations.map_id`, `location_id`, `selection_weight`.

**Validation**

- Map must be active and playable.
- Access tier decisions are enforced server-side.
- Selected map must provide at least `round_count` unique active locations.

## State Flow

1. Create solo game:
   - Resolve guest or registered session.
   - Validate map, round count, timer, and access.
   - Select `round_count` unique active locations.
   - Insert `games`, one `game_players` row, and all `rounds` in one transaction.

2. Start game:
   - Move game from `pending` to `active`.
   - Set `games.started_at`.
   - Activate round 1.
   - Set `rounds.starts_at` and `rounds.ends_at` when timed.

3. Read current round:
   - Authorize owner.
   - If active timed round expired, complete/reveal it server-side.
   - Return current active/pending round state without hidden coordinates.

4. Submit guess:
   - Authorize owner.
   - Validate current round and server deadline.
   - Apply idempotency behavior.
   - Compute distance and score.
   - Insert guess, complete/reveal round, update player and game totals in one transaction.
   - Activate next round or complete game.

5. Read results:
   - Authorize owner.
   - Return durable game, player, round, revealed location, and guess results when completed.

## Query and Index Requirements

- Use indexed lookups by `games.id`, `rounds.game_id`, `game_players.game_id`, and `guesses.round_id`.
- Avoid N+1 result reads by loading rounds, locations, guesses, and player data in bounded batched queries.
- Add a `rounds_game_id_status_round_number` index if current-round lookup needs it after query review.
- Add a partial unique `guesses_game_player_id_idempotency_key_unique` index if not already present.

## Security and Redaction

- Logs must not include exact actual coordinates before reveal or raw guest/session identifiers.
- API DTOs must not include raw GORM models.
- Result DTOs may include actual coordinates only after round completion/reveal.
- Authorization belongs in `games.Service`, not only middleware.
