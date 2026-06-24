# Phase 1 - Core Game Loop

Goal: make the game playable from start to finish without accounts, backend, multiplayer, or advanced modes.

## Player Experience

- Player sees a Street View or panorama clue.
- Player can look around and optionally move if the provider allows it.
- Player places one guess on a world map.
- Player submits the guess.
- Game reveals the real location.
- Game shows distance, score, and answer name.
- Player can continue to the next round.
- Game tracks total score across rounds.

## Required Features

- Location dataset with at least 20 test locations.
- Random location selection.
- Avoid immediate repeated locations.
- Panorama viewer.
- Guess map.
- Click-to-place or drag-to-place guess marker.
- Submit button disabled until a guess exists.
- Distance calculation using Haversine.
- Score formula with max score, usually `5000`.
- Result overlay with:
  - correct place
  - guessed distance
  - round score
  - next round button
- HUD with:
  - current round
  - total score
- Basic loading and error states.

## Technical Scope

- Client-only state is fine.
- No database.
- No login.
- No persistent score history.
- No custom settings yet.
- No timer yet.
- No multiplayer.

## Suggested Files

- `lib/locations.ts`
- `lib/haversine.ts`
- `lib/scoring.ts`
- `components/StreetView.tsx`
- `components/GuessMap.tsx`
- `components/ResultCard.tsx`
- `components/ScoreBar.tsx`
- `app/page.tsx`

## Done When

- A player can play at least 5 rounds.
- Each round can be guessed and scored.
- The map reveals both guess and answer.
- Total score updates correctly.
- Refreshing the page starts a fresh game.
- Build and lint pass.
