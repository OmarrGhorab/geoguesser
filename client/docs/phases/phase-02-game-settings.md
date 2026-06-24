# Phase 2 - Game Settings

Goal: let players change basic rules before starting a game.

## Required Features

- Start screen or settings panel.
- Round count setting:
  - 1
  - 3
  - 5
  - 10
- Timer setting:
  - off
  - 10 seconds
  - 30 seconds
  - 1 minute
  - 3 minutes
- Movement mode:
  - Move
  - No Move
  - No Move, Pan, Zoom
- Selected map pool.
- Start game button.
- Restart game button.

## Game Rules

- If timer reaches zero, auto-submit if a guess exists.
- If timer reaches zero without a guess, score `0`.
- No Move disables Street View links/navigation.
- No Move, Pan, Zoom also disables camera controls where possible.

## Done When

- Settings affect new games.
- Settings cannot corrupt an active round.
- Timer is visible during timed games.
- End-of-round behavior is clear when time expires.
