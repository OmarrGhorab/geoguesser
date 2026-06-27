# Feature Specification: Solo Game Loop

**Feature Branch**: `004-solo-game-loop`

**Created**: 2026-06-27

**Status**: Draft

**Input**: User description: "docs/phases/backend/phase-04-solo-game-loop.md - ship the server-authoritative solo gameplay loop from game creation through scoring and reveal."

## User Scenarios & Testing

### User Story 1 - Complete a Solo Game (Priority: P1)

A guest or registered player can create a solo game, start it, play every round, submit guesses, and reach final results without relying on client-owned game state.

**Why this priority**: This is the core gameplay loop and the minimum playable backend slice for solo GeoGuess.

**Independent Test**: A guest or registered player can create and start a game, receive each current round, submit one valid guess per round, and retrieve final results with total distance and score.

**Acceptance Scenarios**:

1. **Given** a guest or registered player with access to gameplay, **When** the player creates and starts a solo game, **Then** the game becomes playable and returns the first round without exposing the actual answer coordinates.
2. **Given** an active solo game round, **When** the player submits a valid guess before time expires, **Then** the round records the guess, returns distance and score, and reveals the actual answer for that round.
3. **Given** a player has completed all rounds in a solo game, **When** the player requests final results, **Then** the game returns durable per-round results and total score.

---

### User Story 2 - Enforce Fair Round Rules (Priority: P1)

The system controls round selection, round state transitions, timing, and reveal rules so the player cannot gain an unfair advantage by changing client state.

**Why this priority**: Solo game results are only meaningful if scoring, timing, and answer reveal are controlled by the server-side game state.

**Independent Test**: A player cannot see answer coordinates before reveal, cannot receive repeated locations within one game, cannot submit late guesses, and cannot submit more than one guess for the same round.

**Acceptance Scenarios**:

1. **Given** a solo game with multiple rounds, **When** the system creates rounds for that game, **Then** no location appears more than once in the same game.
2. **Given** a round has not been guessed or timed out, **When** the player reads the current round, **Then** the actual answer coordinates remain hidden.
3. **Given** the round timer has expired according to server time, **When** the player submits a guess, **Then** the guess is rejected and does not change the round result.
4. **Given** the player already submitted a guess for a round, **When** the player submits another guess for that round, **Then** the second guess does not create a second result or change the original score.

---

### User Story 3 - Retry Guess Submission Safely (Priority: P2)

A player whose network request is retried can submit the same guess request safely without creating duplicate guesses or inconsistent scoring.

**Why this priority**: Guess submission happens during timed gameplay, so safe retry behavior prevents accidental duplicate submissions and confusing results.

**Independent Test**: Repeating the same guess submission with the same retry identity returns the same recorded result and does not alter the round.

**Acceptance Scenarios**:

1. **Given** a guess submission succeeds but the client does not receive the response, **When** the same submission is retried, **Then** the player receives the original result without creating a duplicate guess.
2. **Given** a completed round result exists, **When** a conflicting retry attempts to change the guess, **Then** the original result remains authoritative.

---

### User Story 4 - Resume and Inspect Game State (Priority: P2)

A guest or registered player can reload an in-progress or completed solo game and see the correct current round, completed round results, and final totals.

**Why this priority**: Gameplay should survive browser refreshes and repeated reads, and final scores must remain durable.

**Independent Test**: A player can start a solo game, submit at least one guess, reload game state, continue from the correct current round, finish the game, and reload the final results.

**Acceptance Scenarios**:

1. **Given** an in-progress solo game, **When** the player reloads game state, **Then** the returned state identifies the current playable round and already completed round results.
2. **Given** a completed solo game, **When** the player requests results later, **Then** the same per-round results and totals are returned.

### Edge Cases

- What happens when a player starts a game that is already active or completed?
- What happens when a player asks for the current round before starting the game?
- What happens when a player submits a guess for a game or round they do not own?
- What happens when a player submits a guess with latitude or longitude outside valid Earth coordinate ranges?
- What happens when there are not enough unique eligible locations to fill all game rounds?
- What happens when two guess submissions for the same round arrive at nearly the same time?
- What happens when a round expires while a guess request is in flight?
- What state is returned when the player requests results before all rounds are complete?
- What loading, empty, error, disabled, and success states can client screens derive from game creation, round reads, guess submission, reveal, and final results?
- How does the experience behave in English and Arabic, including RTL layout, when client screens display gameplay errors or result messages?
- What performance or latency boundary would make gameplay feel broken? Current round, guess result, and final results should feel immediate during normal play.

## Requirements

### Functional Requirements

- **FR-001**: The system MUST allow a guest or registered player to create a solo game.
- **FR-002**: The system MUST allow the owning player to start a created solo game.
- **FR-003**: The system MUST create a fixed sequence of rounds for each solo game once the game starts.
- **FR-004**: The system MUST select round locations without repeating a location within the same game.
- **FR-005**: The system MUST own round timing and determine whether a guess is on time using server-side time.
- **FR-006**: The system MUST expose the current playable round for an in-progress solo game.
- **FR-007**: The system MUST hide actual answer coordinates until a round is completed or revealed.
- **FR-008**: The system MUST allow the owning player to submit one guess per round.
- **FR-009**: The system MUST reject late guesses and guesses for games or rounds that are not currently playable.
- **FR-010**: The system MUST make guess submission safe to retry so the same retry returns the same outcome without duplicate scoring.
- **FR-011**: The system MUST calculate distance between the guessed coordinates and actual answer coordinates.
- **FR-012**: The system MUST calculate a score for each submitted guess using the game scoring rules.
- **FR-013**: The system MUST reveal the actual answer and computed result after a valid guess or completed round.
- **FR-014**: The system MUST provide final solo game results including per-round distance, per-round score, total distance, and total score.
- **FR-015**: The system MUST make completed game results durable and reloadable.
- **FR-016**: The system MUST prevent players from reading or changing solo games they do not own.
- **FR-017**: The system MUST provide stable error and state outcomes that client interfaces can present in supported locale message catalogs.
- **FR-018**: Gameplay-facing state changes MUST be reflected in the feature planning artifacts, including any user-visible contract changes.

### Key Entities

- **Solo Game**: A single-player game owned by a guest or registered player, with lifecycle state, round count, timing rules, and aggregate results.
- **Game Player**: The player identity associated with a solo game, including whether the player is a guest or registered user.
- **Round**: One playable challenge within a solo game, linked to a selected location, a round number, timing state, and reveal state.
- **Guess**: A player's submitted latitude and longitude for a round, with submission time, retry identity, computed distance, and computed score.
- **Location**: A playable answer location selected for a round. Its answer coordinates remain hidden from the player until reveal.
- **Game Result**: The durable summary of completed rounds, per-round outcomes, total distance, total score, and completion time.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A guest or registered player can complete a full solo game from creation to final results in one continuous flow.
- **SC-002**: 100% of completed solo games return reloadable final results with per-round distance, per-round score, total distance, and total score.
- **SC-003**: 100% of active solo game rounds hide actual answer coordinates until the round is completed or revealed.
- **SC-004**: Duplicate or retried guess submissions for the same round do not create duplicate scores or alter the original result.
- **SC-005**: Late guesses are rejected based on server-owned timing.
- **SC-006**: Current round reads, accepted guess results, and final result reads complete within 1 second for normal solo gameplay.
- **SC-007**: Automated verification covers the core loop at unit, request/response, data persistence, and end-to-end service levels.

## Assumptions

- Solo games are owned by exactly one guest or registered player.
- A solo game uses the project's default round count and round duration unless a later phase introduces custom settings.
- The available location pool from the previous phase is sufficient for at least one full solo game.
- Score calculation follows the existing project scoring rules from the backend design sources.
- A round is considered complete after a valid guess or after server-owned time expires.
- Results are visible only to the owning player during this phase.
- Leaderboards, multiplayer rooms, custom map pools, and imported coordinate pools are outside this phase.
- Client implementation work is outside this backend phase, but backend outcomes must support localized and accessible client states.
