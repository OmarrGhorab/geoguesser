# Phase 10 - Friends Social Graph And Access Controls

Goal: add the social graph needed for friends-only comparisons and future social features.

## Scope

- `internal/friends`
- friend request and acceptance flows
- blocked relationship handling
- friends-only leaderboard access control

## Durable Data

- `friendships`
- supporting `users`
- supporting `user_profiles`

## Rules

- Only registered users can participate.
- Friends leaderboards must include accepted friends only.
- Blocking rules must prevent unwanted social visibility where applicable.

## Design Sources

- `docs/phase-3-database-design.md`
- `docs/phase-5-backend-architecture.md`
- `docs/phase-8-technical-specifications.md` feature 8

## Done When

- Social graph rules are durable and queryable.
- Friends-only competitive reads can be enforced safely.
- Blocking and access-control rules have backend tests.

## Dependencies

- Phase 7
- Phase 8
