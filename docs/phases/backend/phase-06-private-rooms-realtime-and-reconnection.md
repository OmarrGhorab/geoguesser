# Phase 6 - Private Rooms Realtime And Reconnection

Goal: support synchronized private-room play with server-authoritative room state.

## Scope

- `internal/rooms`
- `internal/realtime`
- multiplayer game start flow
- lobby membership and host controls
- room state caching in Redis
- synchronized timers and round transitions
- reconnect and disconnect handling

## APIs

- `POST /api/v1/rooms`
- `POST /api/v1/rooms/join`
- `GET /api/v1/rooms/{roomCode}`
- `PATCH /api/v1/rooms/{roomCode}/settings`
- `POST /api/v1/rooms/{roomCode}/start`
- `DELETE /api/v1/rooms/{roomCode}/players/{playerId}`
- realtime endpoint for room events

## Durable Data

- `rooms`
- `room_players`
- `games`
- `game_players`
- `rounds`
- `guesses`

## Ephemeral Data

- room snapshots
- presence
- ready state if enabled
- reconnect windows
- event streams
- room locks and versions

## Rules

- Room state is backend authoritative.
- Non-host players cannot mutate lobby settings or start the room.
- Realtime events are versioned hints, not the only source of truth.
- Disconnects must not corrupt completed guesses or final scores.

## Design Sources

- `docs/phase-2-system-design.md`
- `docs/phase-3-database-design.md`
- `docs/phase-4-api-design.md`
- `docs/phase-5-backend-architecture.md`
- `docs/phase-8-technical-specifications.md` features 2, 3, and 4

## Done When

- Two players can complete a private room game with synchronized rounds.
- Host authorization, room joins, reconnects, and late-guess rejection are enforced server-side.
- Redis-backed room state and realtime flow have integration coverage.

## Dependencies

- Phase 1
- Phase 2
- Phase 3
- Phase 4
