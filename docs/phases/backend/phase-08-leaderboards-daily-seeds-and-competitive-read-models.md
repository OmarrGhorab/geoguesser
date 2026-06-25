# Phase 8 - Leaderboards Daily Seeds And Competitive Read Models

Goal: add the read models and deterministic seed foundations needed for shared comparison modes.

## Scope

- `internal/leaderboards`
- leaderboard definition and entry materialization
- Redis hot leaderboard caching
- daily and map leaderboard reads
- deterministic daily/shared challenge seed model

## APIs

- `GET /api/v1/leaderboards/global`
- `GET /api/v1/leaderboards/daily`
- `GET /api/v1/leaderboards/maps/{mapId}`

## Durable Data

- `leaderboards`
- `leaderboard_entries`
- supporting `games` and `user_profiles` data

## Rules

- Public durable leaderboards default to registered users only.
- Leaderboard reads must be cursor-paginated and abuse-resistant.
- Daily/shared challenge support should reuse backend-controlled deterministic game selection rather than frontend-only seeding.

## Design Sources

- `docs/phase-1-product-definition.md`
- `docs/phase-3-database-design.md`
- `docs/phase-4-api-design.md`
- `docs/phase-5-backend-architecture.md`
- `docs/phase-8-technical-specifications.md` feature 6

## Design Gaps To Resolve In Implementation

- Daily challenge endpoints and exact challenge resource shape are implied by product and leaderboard docs, but not yet fully specified in OpenAPI.
- Shared challenge link resources need a concrete backend contract before implementation.

## Done When

- Completed registered games can feed leaderboard materialization.
- Global, daily, and map leaderboard reads are stable.
- Deterministic seed design for daily/shared challenge gameplay is documented close to implementation.

## Dependencies

- Phase 4
- Phase 5
- Phase 7
