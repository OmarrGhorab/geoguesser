# Feature Specification: Daily Mission Game Flow

## User Story

As an authenticated player, I can start today's daily mission once, complete
five geography rounds, resume an unfinished attempt, and see my persisted total
score with a five-round score breakdown.

## Acceptance Scenarios

1. Given today's calendar day and no existing attempt, when the player selects
   Play, then one dated daily attempt and one five-round challenge game are
   created and the player is taken to round one.
2. Given an active attempt for today, when the player selects Play again or
   reloads the game, then the same attempt and current round are resumed.
3. Given five accepted guesses, when round five completes, then the attempt is
   completed, its total score and five round scores are stored, and the final
   result view shows the total and ordered breakdown.
4. Given a completed attempt for today, when the player returns to the daily
   mission, then another attempt is not created and View Results scrolls to the
   stored result directly below the completed summary.
5. Given a past or future calendar day, Play is disabled; only today's mission
   can be started.
6. Given English or Arabic, all added controls, statuses, errors, and result
   labels are localized and remain usable with keyboard input and RTL layout.
7. Given a previously materialized daily challenge whose locations have no
   playable media, when an unplayed attempt is retried, then its five rounds
   are replaced from a playable public map without changing any scored game.
8. Given an active or completed daily mission, its localized URL uses semantic
   `daily-mission/play` and `daily-mission/results` segments without exposing a
   challenge identifier in the query string.
9. Given a completed five-round mission, the result opens on a map-first score
   summary, and Breakdown reveals every guess, correct position, connecting
   distance line, score, and distance for all five rounds.
10. Given today's attempt is completed, returning to the daily mission replaces
    the pre-play feature cards with a marked five-round map, real score summary,
    completed-state artwork, and a detailed Results/Map section directly below
    the summary that View Results scrolls to.

## Requirements

- **FR-001**: Reuse the existing dated `challenges`, `challenge_attempts`,
  `games`, `rounds`, `guesses`, and `challenge_results` records.
- **FR-002**: The backend MUST remain authoritative for current UTC challenge
  date, attempt ownership, one-attempt uniqueness, round order, scoring, and
  result visibility.
- **FR-003**: A daily mission MUST contain exactly five rounds and its result
  breakdown MUST contain exactly five ordered score entries after completion.
- **FR-004**: Starting an existing pending or active attempt MUST resume it;
  starting a completed attempt MUST preserve completed status.
- **FR-005**: Current-round responses MAY expose an opaque playable panorama ID
  but MUST NOT expose answer latitude, longitude, location ID, or raw database
  field names before guess submission.
- **FR-006**: Guess submission MUST be CSRF-protected, idempotent, validated,
  and sent through a Next.js Server Action.
- **FR-007**: The daily mission page MUST show real participant count and real
  attempt state, and distinguish Play, Resume, and View Results.
- **FR-008**: The game route MUST lazy-load Google Maps only on the gameplay
  surface and keep the selected unsent pin in local client state.
- **FR-009**: Missing round media MUST show an explicit recoverable error and
  MUST NOT remain in an indefinite loading state.
- **FR-010**: Completed results MUST use the existing owned game-results read
  endpoint and MUST NOT require a new backend contract.
- **FR-011**: Personal result routes MUST provide localized descriptive metadata
  while remaining `noindex` and `nofollow`.
- **FR-012**: The completed-today landing MUST derive every displayed result
  value from the owned game-results response and MUST reuse the same marked-map
  renderer as the detailed result page.
- **FR-013**: The completed-today summary and detailed Results/Map presentation
  MUST be sections of the same daily-mission page; View Results MUST navigate to
  the detailed section without another API request or route transition.

## Out of Scope

- Replaying historical daily challenges.
- Multiple attempts for the same player and date.
- XP, medals, rewards, or economy state.
- Multiplayer gameplay or realtime spectators.
- Pixel-identical reproduction of third-party GeoGuessr screens.

## Success Criteria

- Duplicate Play actions reuse one attempt and game.
- Completion persists one result snapshot with five ordered round marks.
- Current-round payloads contain no answer coordinates before reveal.
- Backend and frontend required gates pass, plus focused daily mission tests.
