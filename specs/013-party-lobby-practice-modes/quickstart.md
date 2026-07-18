# Quickstart: Party Lobby And Practice Modes

## Prerequisites

- Go 1.24+
- PostgreSQL with migrations through `00020`
- Redis for Party Lobby realtime validation only
- A seeded map with enough enabled locations
- Existing registered or guest session and CSRF token

## Static And Unit Gates

From `backend/`:

```powershell
gofmt -w ./internal/games ./internal/rooms
go test -count=1 ./internal/games ./internal/rooms ./internal/app
go vet ./...
golangci-lint run ./...
go test -count=1 ./...
```

## Party Lobby Validation

1. Create a room with a map and confirm `mode=party_lobby`, `round_count=5`, `timer_seconds=180`, and `max_players=50`.
2. Join multiple identities and verify one roster row per identity.
3. Verify non-host settings/start/remove requests are rejected.
4. Start and confirm identical round IDs, media, start time, and deadline.
5. Let some players miss a deadline; verify zero scores and forward progress.
6. Complete all rounds and verify standings order, ties, reload, and no competitive changes.
7. Replay join, settings, start, and guess commands and verify no duplicate state.

## Practice Validation

1. Create and start `mode=practice` with a valid map.
2. Confirm timer fields are null and no realtime ticket/channel is created.
3. Guess and verify immediate answer reveal, accuracy-only score, and next-round availability.
4. Request next twice concurrently with the same key; verify exactly one sequential round.
5. Request next before guessing and verify `current_round_incomplete`.
6. Complete more than ten rounds.
7. Page history and verify stable cursors, bounded counts, and owner-only answers.
8. Stop Redis and repeat Practice to prove no Redis dependency.
9. End the session, retain history, and reject further advancement.
10. Confirm competitive, mission, daily, leaderboard, and matchmade stats are unchanged.

## Migration Validation

1. Upgrade a database containing `private_room` games.
2. Confirm old rows remain readable and new room writes use `party_lobby`.
3. Confirm new known modes pass and unknown modes fail constraints.
4. Verify down on a fixture without new rows.
5. Verify down refuses destructive rollback when new-mode rows exist.

## Security And Observability

- Non-owners receive privacy-safe Practice errors.
- Non-members cannot read Party Lobby state.
- Pre-reveal Party Lobby state contains no answers or other players' guesses.
- Logs and metrics contain no room codes, raw coordinates, guest hashes, sessions, or hidden locations.
- Metrics distinguish bounded Party Lobby and Practice outcomes.

## Evidence

Recorded 2026-07-18 on Windows from `backend/`:

| Gate | Result | Evidence |
|---|---|---|
| Targeted packages | PASS | `go test -count=1 ./internal/games ./internal/rooms ./internal/app ./internal/profiles ./internal/realtime` |
| Full backend | PASS | `go test -count=1 ./...`; all packages green, including Redis/realtime |
| Static analysis | PASS | `go vet ./...` |
| Lint | PASS | `golangci-lint run ./...`; `0 issues` |
| OpenAPI/examples | PASS | Redocly 2.38.0 validated the contract (28 pre-existing recommendation warnings); new saved-response JSON decoded successfully |
| Dashboard JSON | PASS | Parsed successfully with seven panels including new mode operations |
| Diff whitespace | PASS | `git diff --check`; only configured Windows LF→CRLF notices |
| PostgreSQL/Redis disposable integration | BLOCKED | Docker Desktop Linux engine was unavailable; repository tests requiring `DATABASE_URL` skipped |
| Race detector | BLOCKED | Host is Windows with `CGO_ENABLED=0` and no `gcc`; Linux CI race evidence remains required |
| Live multi-client soak | NOT RUN | No disposable PostgreSQL/Redis-backed API environment was available; automated room/realtime suites passed |

Residual risk is limited to applying migration `00020` and exercising long-running 50-player/1,000-round flows against a real disposable PostgreSQL/Redis stack and Linux race detector. Core unit, contract, concurrency-path compilation, worker, realtime, and full backend suites are green.
