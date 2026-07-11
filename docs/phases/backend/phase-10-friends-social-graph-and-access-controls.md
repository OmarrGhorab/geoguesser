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

## Implementation status (2026-07-11)

Backend feature branch `009-friends-social-graph` is implemented and verified locally:

- Goose migration `00015_friends_social_graph.sql` (`friendships` sorted pairs, lifecycle checks, indexes); up/down/up validated
- Package `backend/internal/friends` (requests, accept/decline, lists, remove, block/unblock, metrics)
- Authenticated `GET /leaderboards/friends` cohort ranking in `backend/internal/leaderboards`
- OpenAPI friends + friends leaderboard contracts
- Gates: `gofmt`, `go vet`, `golangci-lint`, unit `go test ./...`, Docker PG/Redis integration for friends/leaderboards/matchmaking/games/redis/app, `go build ./cmd/api`, Redocly OpenAPI valid, frontend lint/typecheck/build
- Linux `go test -race` remains the CI gate (Windows host lacks CGO race support)

## Dependencies

- Phase 7
- Phase 8
