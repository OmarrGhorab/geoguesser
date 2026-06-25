# Phase 7 - Profiles Stats And Persistent Progress

Goal: turn registered play into persistent account progress and public-safe stats.

## Scope

- `internal/profiles`
- profile read and update flows
- public stats aggregation
- registered-only saved progress foundations

## APIs

- `GET /api/v1/profile`
- `PATCH /api/v1/profile`
- `GET /api/v1/users/{userId}/stats`

## Durable Data

- `users`
- `user_profiles`
- `games`
- `game_players`
- `guesses`

## Rules

- Guest sessions cannot access registered-only profile surfaces.
- Public stats must exclude private account data.
- Profile updates need validation and rate limiting.

## Design Sources

- `docs/phase-3-database-design.md`
- `docs/phase-4-api-design.md`
- `docs/phase-5-backend-architecture.md`
- `docs/phase-8-technical-specifications.md` features 11 and 13

## Done When

- Registered users can read and update profile data safely.
- Public stats aggregate from completed games.
- Backend foundations exist for frontend saved history and persistent progress.

## Dependencies

- Phase 2
- Phase 4
- Phase 5
