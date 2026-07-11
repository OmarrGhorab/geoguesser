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

## Implementation Status (2026-07-11)

Feature delivered on branch `008-matchmaking-ranked` (spec `specs/008-matchmaking-ranked/`):

### Backend

- Registered-only join/leave/status with CSRF, per-user rate limits, privacy-safe DTOs
- Redis atomic join/leave/claim/finalize/release with original-priority preservation
- Durable formation: ranked game + two players + rounds + match + match_players
- Concurrent formation race (exactly one match)
- Ranked multiplayer (`games.mode=ranked`) with scheduled first-round start
- Match lifecycle + completed-result eligibility (no ratings)
- Migration `00014_matchmaking_ranked.sql` validated up/down/up

### Frontend

- Localized `/[locale]/matchmaking` with 2s non-overlapping polling, leave race recovery, backoff
- Ranked destination `/[locale]/games/[gameId]` with countdown, guess submit, results
- EN/AR catalogs with parity tests; accessible controls/live regions/alerts
- Same-origin status proxy

**Explicitly out of scope still**: ratings, seasons, divisions, rewards, parties, tournaments, duel combat, Phase 12 background matcher/sweeper.

**Residual**: interactive two-session browser QA and full AR/RTL keyboard pass (documented in `specs/008-matchmaking-ranked/quickstart.md` and `plan.md`).

## Dependencies

- Phase 6
- Phase 8
