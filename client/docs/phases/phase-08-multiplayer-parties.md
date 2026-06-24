# Phase 8 - Multiplayer Parties

Goal: let friends play the same rounds together in a private room.

## Required Features

- Create room.
- Join room by code/link.
- Host selects:
  - map pool
  - rounds
  - timer
  - movement mode
- Lobby player list.
- Ready/start flow.
- Synchronized round start.
- Everyone guesses independently.
- Round scoreboard.
- Final scoreboard.

## Technical Scope

- Real-time transport, usually WebSocket.
- Server-authoritative room state.
- Reconnect support.
- Host migration or room close if host leaves.

## Done When

- At least two players can complete a private game together.
- Late/reconnecting players do not break the room.
- Scores match server calculations.
