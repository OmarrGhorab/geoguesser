# Phase 6 - Streak Modes

Goal: add lightweight replayable modes based on country or region guessing.

## Country Streak

- Player guesses country instead of exact location.
- Correct country increases streak by 1.
- Wrong country ends run.
- Reveal correct country after every guess.

## Region Streak

- Player guesses state/province/region where data supports it.
- Fallback to country streak for locations without region metadata.

## Required Features

- Country selector.
- Region selector where available.
- Streak counter.
- Best streak stored locally.
- Run summary after failure.

## Done When

- Player can play a full streak run.
- Correct/wrong rules are obvious.
- Best streak persists locally.
