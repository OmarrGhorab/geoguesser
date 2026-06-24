# Phase 3 - Results And History

Goal: make finished games reviewable and satisfying.

## Required Features

- Final game summary screen.
- Per-round table:
  - round number
  - location name
  - distance
  - score
- Total score out of max possible score.
- Percentage score.
- Result map showing all guesses and answers.
- Play again button.
- Local recent games history.

## Optional Features

- Local best score per map.
- Copy result summary.
- Shareable image/card.
- Export game JSON.

## Storage

- Use `localStorage` first.
- Move to database only after accounts exist.

## Done When

- Player can review a full game after the last round.
- Refresh does not erase recent completed games.
- History can be cleared.
