# Quickstart: Solo Game Loop Validation

## Prerequisites

- PostgreSQL and Redis reachable through `backend/.env` (`DATABASE_URL` and `REDIS_URL`).
- Backend migrations applied.
- At least one active public map with at least five active locations.
- Guest or registered auth flow available from the previous phases.

## Setup

```powershell
cd backend
go test ./...
```

This repository's current `docker-compose.yml` defines `api` and `client`; it does not define `postgres` or `redis` services. Start the Postgres and Redis instances referenced by `backend/.env` before running the API or manual scenario.

Apply migrations from `backend/` using the project's Goose-compatible migration runner, for example:

```powershell
goose -dir migrations postgres $env:DATABASE_URL up
```

If seed data is needed, run the existing seed command or seed an active map with enough active locations before manual API validation.

## Automated Validation

Run all backend tests:

```powershell
cd backend
go test ./...
```

Run targeted game tests once the package exists:

```powershell
cd backend
go test ./internal/games/...
```

Run contract validation from the repository root:

```powershell
pnpm check:openapi
```

If the script is not present in the root `package.json`, validate `backend/openapi/openapi.yaml` with the repository's configured OpenAPI lint tool before release.

Release gates remain:

```powershell
pnpm check
```

## Manual API Scenario

1. Resolve a guest or registered session.
2. Create a solo game with a valid map id:

```http
POST /api/v1/games
Content-Type: application/json

{
  "mode": "solo",
  "map_id": "<map-id>",
  "round_count": 5,
  "timer_seconds": 120
}
```

Expected:

- Response status is `201`.
- Game status is `pending`.
- No actual coordinates are returned.

3. Start the game:

```http
POST /api/v1/games/{gameId}/start
```

Expected:

- Response status is `200`.
- Game status is `active`.
- Current round number is `1`.

4. Read the current round:

```http
GET /api/v1/games/{gameId}/rounds/current
```

Expected:

- Response status is `200`.
- Round contains media and server timestamps.
- Response does not contain `location_id`, actual latitude, or actual longitude.

5. Submit a guess:

```http
POST /api/v1/games/{gameId}/rounds/{roundId}/guesses
Idempotency-Key: <stable-key>
Content-Type: application/json

{
  "latitude": 30.0444,
  "longitude": 31.2357
}
```

Expected:

- Response status is `200`.
- Response includes `distance_meters`, `score`, `submitted_at`, and revealed actual location.
- Repeating the same request with the same idempotency key returns the same result.
- Sending a different guess with the same idempotency key returns a conflict without changing the original result.
- Sending a different guess for the same completed round without the same idempotency key is rejected as already guessed.

6. Repeat current-round and guess submission until all rounds complete.

Expected:

- Each round has a unique selected location.
- Each accepted guess reveals only that completed round.
- The game becomes `completed` after the final round.

7. Read final results:

```http
GET /api/v1/games/{gameId}/results
```

Expected:

- Response status is `200`.
- Results include all rounds, per-round distance and score, total score, and revealed locations.
- Reloading the same endpoint returns the same durable results.

## Edge Validation

- Submit latitude greater than 90 and confirm validation failure.
- Submit after the server-owned deadline and confirm `round_closed` or equivalent stable error.
- Submit a guess as a different user or guest and confirm forbidden/not found.
- Create a game on a map with fewer active locations than requested and confirm `not_enough_locations`.
- Read current round before start and confirm stable pending/not-started behavior.
