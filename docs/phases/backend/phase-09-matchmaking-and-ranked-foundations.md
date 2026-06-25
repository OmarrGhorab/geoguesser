# Phase 9 - Matchmaking And Ranked Foundations

Goal: introduce backend queueing and ranked match foundations after private rooms are stable.

## Scope

- `internal/matchmaking`
- queue enter, leave, and status flows
- Redis queue coordination
- match formation hooks into room or game creation
- ranked-ready backend contracts for competitive play

## APIs

- `POST /api/v1/matchmaking/queue`
- `DELETE /api/v1/matchmaking/queue`
- `GET /api/v1/matchmaking/status`

## Durable Data

- `matches`
- `match_players`
- downstream `games`

## Ephemeral Data

- matchmaking queues
- per-player queue state
- region or mode locks

## Rules

- Queue state lives in Redis, while formed match outcomes are durable.
- Match formation must remain server-authoritative.
- Ranked progression should build on this phase, not bypass it.

## Design Sources

- `docs/phase-3-database-design.md`
- `docs/phase-4-api-design.md`
- `docs/phase-5-backend-architecture.md`
- `docs/phase-8-technical-specifications.md` feature 7

## Design Gaps To Resolve In Implementation

- Rating change formulas, season lifecycle, and duel-specific combat rules are not fully specified in the current root backend docs.
- The old product phase for duels needs dedicated backend specification before direct implementation.

## Done When

- Players can enter and leave a backend queue safely.
- Match formation and status reporting are observable and testable.
- Ranked feature work has an approved backend foundation instead of ad hoc room logic.

## Dependencies

- Phase 6
- Phase 8
