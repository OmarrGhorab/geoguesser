# Quickstart: Room Realtime Reconnection

## Prerequisites

- Docker services for PostgreSQL and Redis are running.
- Backend `.env` contains valid `DATABASE_URL`, `REDIS_URL`, auth secrets, and room realtime configuration.
- Client `.env.local` or environment contains `BACKEND_API_URL` and `NEXT_PUBLIC_REALTIME_URL` for local development.
- At least one active map has enough active locations for the configured round count.

## Setup

From repo root:

```powershell
docker compose up -d postgres redis
```

Run backend migrations using the repo's existing migration command or container workflow.

Start backend:

```powershell
cd backend
go test ./...
go run ./cmd/api
```

Start frontend:

```powershell
npx pnpm@10.24.0 --dir client install
npx pnpm@10.24.0 --dir client dev
```

## Validation Commands

Backend gates:

```powershell
cd backend
go test ./...
go build ./cmd/api
golangci-lint run
```

OpenAPI gate:

```powershell
npx pnpm@10.24.0 check:openapi
```

Frontend gates:

```powershell
npx pnpm@10.24.0 --dir client lint
npx pnpm@10.24.0 --dir client typecheck
npx pnpm@10.24.0 --dir client build
```

## Manual Happy Path

1. Open `http://localhost:3000/en/rooms` in Browser A.
2. Create a private room with a playable map, 2 or more max players, and a short timer.
3. Copy the room code or URL.
4. Open Browser B in a separate session and join the room.
5. Confirm both browsers show the same roster, host, settings, ready state, and presence without manual refresh.
6. From Browser A, update a lobby setting.
7. Confirm Browser B receives the update and any ready state reset.
8. From Browser B, attempt a host-only action.
9. Confirm the command is rejected and room state remains unchanged.
10. From Browser A, start the room.
11. Confirm both browsers receive the same round number, start time, deadline, and playable media.
12. Submit a guess from Browser A.
13. Confirm Browser B sees aggregate guess progress without answer coordinates.
14. Submit a guess from Browser B.
15. Confirm the round ends, results reveal consistently, and the next round starts or final results load.
16. Complete all rounds and reload both browsers.
17. Confirm final scores and per-round outcomes are durable and identical across reloads.

## Reconnect Validation

1. Create and join a private room with two browser sessions.
2. In Browser B, refresh while still in lobby.
3. Confirm Browser B rejoins the same participant slot and Browser A does not see a duplicate player.
4. Start the room.
5. Disconnect Browser B's network or close the realtime connection before guessing.
6. Confirm Browser A sees Browser B as reconnecting/disconnected after the presence threshold.
7. Reconnect Browser B before the round deadline.
8. Confirm Browser B restores the current round and can submit exactly one guess.
9. Repeat disconnect but keep Browser B offline through the deadline.
10. Confirm the room advances and Browser B receives 0 points for that missed round without blocking Browser A.

## Hidden Coordinate Validation

Before a player guesses or a round is revealed:

- `GET /api/v1/rooms/{roomCode}` does not include answer coordinates or `location_id`.
- `round.started` events do not include answer coordinates or `location_id`.
- `round.guess_count_changed` events include counts and player IDs only.

After reveal:

- Result payloads include revealed locations only for authorized participants and completed/revealed rounds.

## Localization And Accessibility Pass

1. Repeat lobby and gameplay flows under `/en`.
2. Repeat under `/ar`.
3. Confirm visible copy comes from message catalogs.
4. Confirm Arabic layout direction is RTL.
5. Keyboard through create, join, copy invite, ready, settings, start, remove player, submit guess, reconnect/retry, and result controls.
6. Confirm roster changes and reconnect state are announced politely and do not trap focus.

## Expected Evidence

- Backend test output covering room service, realtime handler, Redis presence/reconnect, room game progression, and hidden-coordinate contract tests.
- Frontend lint/typecheck/build output.
- OpenAPI validation output.
- Recorded browser notes or automated screenshots for the two-session happy path and reconnect path.

## Validation Evidence - 2026-06-30

- `go test ./...` from `backend/`: passed.
- `golangci-lint run` from `backend/`: passed after formatting reported Go files.
- `npx pnpm@10.24.0 --dir client test`: passed, 6 files and 10 tests.
- `npx pnpm@10.24.0 --dir client lint`: passed.
- `npx pnpm@10.24.0 --dir client typecheck`: passed.
- `npx pnpm@10.24.0 --dir client build`: passed.
- `npx pnpm@10.24.0 check:openapi`: valid with the existing warning set for license/tag descriptions/local/example servers, ambiguous challenge paths, and older operations missing 2xx/4xx responses.
- Goose migrations ran against the configured `.env` PostgreSQL and migrated successfully to version 10.
- Docker backend validation passed after Docker Desktop was started: `docker compose up -d --build api` built and started `geoguess-api-1`.
- `docker compose ps` reported `geoguess-api-1` as healthy on port `8080`.
- `GET http://localhost:8080/ready` and `GET http://localhost:8080/api/v1/ready` returned `status=ready` with Postgres and Redis checks `ok`.
- `GET http://localhost:8080/health` returned `status=ok`.
- `GET http://localhost:8080/realtime/rooms/ABC123` without a room session returned `403 Forbidden`, confirming the realtime route is mounted and auth-gated.
