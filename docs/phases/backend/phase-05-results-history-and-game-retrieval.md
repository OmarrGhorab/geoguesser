# Phase 5 - Results History And Game Retrieval

Goal: make completed backend game data reviewable and reusable for history and result surfaces.

## Scope

- game summary and result retrieval hardening
- aggregate result DTO shaping
- registered-user history queries
- public or participant-safe stats primitives needed by later profile and leaderboard features

## APIs

- strengthen `GET /api/v1/games/{gameId}`
- strengthen `GET /api/v1/games/{gameId}/results`
- prepare `GET /api/v1/users/{userId}/stats` dependencies for later profile phase

## Durable Data

- `games`
- `rounds`
- `game_players`
- `guesses`

## Rules

- Participant-only game reads remain enforced.
- Historical scores stay stable because guess distance and score are persisted.
- History queries must be indexed and cursor-friendly as volume grows.

## Design Sources

- `docs/phase-3-database-design.md`
- `docs/phase-4-api-design.md`
- `docs/phase-8-technical-specifications.md` features 1 and 13

## Done When

- Completed game results can be reloaded without recomputing hidden state.
- Registered-user gameplay history has a backend query model.
- Result DTOs are stable enough for frontend history and review screens.

## Dependencies

- Phase 4
