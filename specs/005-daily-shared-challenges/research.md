# Research: Daily And Shared Challenges

## Deterministic Challenge Seeds

**Decision**: Daily challenges use a canonical global challenge date and reset boundary to derive a stable seed. Shared challenges store an explicit generated seed at creation time.

**Rationale**: A daily challenge must be identical for every player on the same challenge date, while a shared challenge must remain stable independent of wall-clock date. A canonical reset boundary avoids timezone disagreement and makes streak and leaderboard rules auditable.

**Alternatives considered**:
- Per-user local-midnight daily challenges: rejected because players in different regions would not compare against the same daily.
- Randomly selecting daily locations on each request: rejected because request-time randomness breaks reproducibility and fairness.

## Immutable Challenge Snapshots

**Decision**: Persist challenge settings and selected location order as a snapshot when a challenge is created or first materialized.

**Rationale**: Map pools can be edited or archived after a challenge is created. Persisting selected location IDs and rule snapshots keeps daily results, shared links, and historical result views stable.

**Alternatives considered**:
- Recomputing selected locations from map state every time: rejected because later map changes would alter historical challenges.
- Storing only the seed and relying on unchanged selector behavior: rejected because selector implementations and map pools can evolve.

## Challenge Attempt Model

**Decision**: Represent each player playthrough as a challenge attempt linked to the challenge and, when gameplay begins, a concrete solo game.

**Rationale**: The existing solo game loop owns round play, scoring, hidden-coordinate behavior, and final results. Challenge attempts add eligibility, leaderboard, streak, mission, and replay rules without duplicating all gameplay mechanics.

**Alternatives considered**:
- Extending only `games.mode = daily`: rejected as insufficient because challenge links, mission progress, daily streaks, and leaderboard eligibility need their own durable facts.
- Creating fully separate challenge round/guess systems: rejected because it duplicates working solo-game scoring and fairness behavior.

## Spoiler-Safe Result Visibility

**Decision**: Hide answer-revealing final details and leaderboard context from unfinished players until they complete the challenge or the challenge is no longer playable.

**Rationale**: Fixed-seed challenges are vulnerable to answer sharing. The backend must shape safe DTOs based on attempt state rather than relying on frontend hiding.

**Alternatives considered**:
- Showing leaderboard immediately to all players: rejected because rank entries and scores can leak answer quality before completion.
- Frontend-only spoiler protection: rejected because clients can inspect network responses.

## Leaderboard Eligibility And Tie-Breakers

**Decision**: Account-backed attempts are eligible for public daily leaderboard ranking, with deterministic ordering by score, completion duration when available, completed timestamp, and stable player/attempt identity as a final tie-breaker. Guest attempts receive personal results and may appear in session-scoped shared comparison only where safe.

**Rationale**: Public leaderboards need durable identity and anti-duplication rules. Deterministic tie-breakers prevent rank flicker and make tests reproducible.

**Alternatives considered**:
- Allowing anonymous global leaderboard entries: rejected for abuse and identity ambiguity.
- Ordering ties randomly or by database insertion only: rejected because rank order would be hard to explain and verify.

## Streak Ownership

**Decision**: Daily streaks are account-backed for durable cross-device use, while guests receive device/session-scoped streak feedback with visible persistence limits.

**Rationale**: Guests should enjoy the daily loop without sign-in friction, but cross-device streaks require a durable account identity.

**Alternatives considered**:
- Requiring sign-in for streaks: rejected because guest challenge play is in scope.
- Treating guest streaks as public leaderboard identity: rejected because guest identity is not sufficiently stable.

## Streak Protection

**Decision**: Model streak protection as an explicit state even if v1 only supports "not available" or a simple earned protection token.

**Rationale**: The specification requires clear recovery/protection behavior. An explicit state keeps UI copy and future extension honest without inventing a full economy.

**Alternatives considered**:
- Omitting protection until later: rejected because the spec requires visible recovery/protection behavior.
- Building a full reward economy now: rejected as beyond phase scope.

## Mission System Scope

**Decision**: Missions are challenge-focused and event-driven, covering daily completion, shared challenge participation, score thresholds, leaderboard milestones, streak milestones, and round accuracy achievements.

**Rationale**: This gives meaningful progression without expanding into unrelated achievements or economy systems. Mission progress can be updated from durable challenge result events.

**Alternatives considered**:
- Generic achievement system for all game actions: rejected as too broad for this phase.
- Frontend-only local missions: rejected because mission progress and rewards must be stable and auditable for accounts.

## Idempotency And Concurrency

**Decision**: Attempt creation, shared challenge creation, result finalization, streak updates, and mission progress application must be idempotent per actor and challenge/result event.

**Rationale**: Challenge flows are retry-heavy and can run from multiple browser sessions. Idempotency prevents duplicate attempts, duplicate streak increments, and duplicate mission rewards.

**Alternatives considered**:
- Relying on UI disabled states: rejected because retries and concurrent sessions bypass UI-only protection.

## Frontend Data Boundaries

**Decision**: Use Next App Router Server Components and server-only data helpers for challenge metadata, attempts, leaderboards, missions, and streaks. Use Client Components only for countdown timers, interactive tabs/filters, share actions, and live-ish progress presentation.

**Rationale**: Local Next.js docs recommend Server Components for data fetching and Client Components for browser state/interactivity. This minimizes shipped JavaScript and keeps tokens/secrets server-side.

**Alternatives considered**:
- Fetching all challenge data from Client Components: rejected due to larger bundles, slower first render, and weaker data boundary discipline.
- Adding client global server-state stores: rejected by the constitution; Zustand is reserved for UI preferences, not canonical server state.

## Caching And Freshness

**Decision**: Cache immutable challenge metadata where safe, but fetch personalized attempt, mission, streak, and leaderboard visibility state at request time or with short-lived revalidation. Countdown is computed from server-provided reset timestamps and updated in the browser.

**Rationale**: Challenge metadata is stable, but player attempt state and mission/streak progress are personalized and must be fresh enough to avoid confusing users.

**Alternatives considered**:
- Caching whole challenge pages for all users: rejected because personalized attempt and spoiler visibility differ per player.
- No caching at all: rejected because stable challenge metadata and shared link lookups can be safely accelerated.

## Observability

**Decision**: Add structured logs and metrics for daily materialization, shared challenge creation, attempt start/completion, leaderboard reads, streak updates, mission progress application, and spoiler-guard rejections.

**Rationale**: This phase creates time-bound and social comparison flows where failures are user-visible and hard to debug after reset windows pass.

**Alternatives considered**:
- Relying only on HTTP request metrics: rejected because business-level events like streak updates and mission awards need direct visibility.
