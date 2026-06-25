# Phase 4 - Solo Game Loop

Goal: ship the server-authoritative solo gameplay loop from game creation through scoring and reveal.

## Scope

- `internal/games`
- solo game creation
- round creation with no repeated locations per game
- timer ownership on the backend
- guess submission with idempotency
- score and distance calculation
- current round and final results reads

## APIs

- `POST /api/v1/games`
- `GET /api/v1/games/{gameId}`
- `POST /api/v1/games/{gameId}/start`
- `GET /api/v1/games/{gameId}/rounds/current`
- `POST /api/v1/games/{gameId}/rounds/{roundId}/guesses`
- `GET /api/v1/games/{gameId}/results`

## Durable Data

- `games`
- `rounds`
- `game_players`
- `guesses`

## Ephemeral Data

- current game state cache
- guess idempotency keys
- rate-limit keys

## Rules

- Backend selects round locations and owns round state transitions.
- A player can submit at most one guess per round.
- Late guesses are rejected by server time.
- Actual coordinates remain hidden until reveal.

## Design Sources

- `docs/phase-3-database-design.md`
- `docs/phase-4-api-design.md`
- `docs/phase-5-backend-architecture.md`
- `docs/phase-8-technical-specifications.md` features 1, 3, and 5

## Done When

- A guest or registered user can complete a full solo game against backend state.
- Guess results return server-computed distance and score.
- Final results and totals are durable and reloadable.
- Unit, handler, repository, and integration coverage exist for the core loop.

## Dependencies

- Phase 1
- Phase 2
- Phase 3
