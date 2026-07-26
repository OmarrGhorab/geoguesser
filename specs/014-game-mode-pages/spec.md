# Feature Specification: Game Mode Pages

**Feature Branch**: `014-game-mode-pages`

**Created**: 2026-07-19

**Status**: Draft

**Input**: User description: "Implement complete pages and backend integration for every player-facing game mode exposed in the navbar."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Start a Solo Experience (Priority: P1)

As a player, I can choose Classic Solo, Quick Play, Practice, or Daily Challenge,
understand the rules before starting, enter the correct game screen, submit
guesses, see permitted answer feedback, and finish or continue according to that
mode's lifecycle.

**Why this priority**: Solo play is the smallest complete gameplay slice and
provides the shared map, round, guessing, scoring, recovery, and results
foundation used by the other experiences.

**Independent Test**: Start each of the four solo experiences from the Play
menu, complete at least one round, and verify that each mode applies its distinct
timer, progression, reveal, continuation, and completion rules.

**Acceptance Scenarios**:

1. **Given** an eligible player on the Play menu, **When** the player selects
   Classic Solo and chooses valid settings, **Then** a fixed-round game begins
   and leads to a final result after its configured rounds.
2. **Given** an eligible player on the Play menu, **When** the player selects
   Quick Play, **Then** a game starts with product defaults without requiring a
   configuration form.
3. **Given** the owner of an active Practice session, **When** a guess is
   submitted, **Then** feedback is revealed immediately and the owner can create
   exactly one next round, review paged history, or end the session.
4. **Given** a player on the Daily Challenge page, **When** today's challenge is
   available, **Then** the player can start or resume today's shared challenge
   and see the appropriate completed state after finishing it.

---

### User Story 2 - Enter Casual and Ranked Matchmaking (Priority: P1)

As an authenticated player, I can choose Casual or Ranked in Solo, Duo, or Squad
format, see eligibility and roster requirements, join or leave the correct queue,
recover an existing queue or assignment, and enter the formed match.

**Why this priority**: The six matchmaking choices are the largest group in the
menu and require one coherent flow so players do not enter the wrong queue or
lose an active assignment.

**Independent Test**: Exercise all six playlist/format combinations with valid
and invalid rosters, then verify queue status, cancellation, assignment recovery,
match entry, round feedback, team scoring, and terminal results.

**Acceptance Scenarios**:

1. **Given** an eligible authenticated player or compatible party, **When** a
   Casual or Ranked format is selected, **Then** the page explains team size and
   rules and enables queue entry only when its requirements are met.
2. **Given** a player searching in one queue, **When** the player reloads or
   returns to the page, **Then** the current queue state is restored without
   creating a duplicate ticket.
3. **Given** a match has been assigned, **When** the status is recovered or a
   realtime assignment arrives, **Then** the player enters the authorized match
   and cannot inspect hidden opponent answers before reveal.
4. **Given** a queue or formation failure, **When** the failure is recoverable,
   **Then** the player receives a safe localized explanation and an appropriate
   retry or leave action.

---

### User Story 3 - Host or Join a Party Lobby (Priority: P1)

As a player, I can create or join a Party Lobby, share its public room code,
manage readiness according to my role, survive reconnects, play the hosted
free-for-all game, and view final individual placements.

**Why this priority**: Party Lobby is the direct play-with-friends experience and
must connect room ownership, realtime state, game lifecycle, and results into one
usable journey.

**Independent Test**: Create a lobby, join from a second session, exercise host
and member controls, disconnect and reconnect, play all rounds, and confirm
deterministic final standings.

**Acceptance Scenarios**:

1. **Given** a player on the Party Lobby page, **When** valid lobby settings are
   submitted, **Then** a lobby is created with a shareable code and the creator
   is identified as host.
2. **Given** a valid non-full lobby code, **When** another eligible player joins,
   **Then** both players see membership and readiness changes without exposing
   private identity or hidden-location data.
3. **Given** all start requirements are satisfied, **When** the host starts the
   lobby, **Then** every active participant enters the same timed round sequence.
4. **Given** the hosted game is complete, **When** results are shown, **Then** all
   participants see deterministic individual placements including ties.

---

### User Story 4 - Navigate and Recover Across Modes (Priority: P2)

As a player, I can use one consistent localized shell to reach any supported
mode, understand an active or completed session, recover after refresh or brief
network loss, and return safely to mode selection.

**Why this priority**: Cross-mode consistency prevents duplicated UI behavior,
broken deep links, and session loss while allowing the P1 mode slices to remain
independently deliverable.

**Independent Test**: Deep-link to setup, active-game, queue, lobby, and result
states in English and Arabic; refresh each state; verify authorization, recovery,
responsive layout, focus behavior, and safe exit navigation.

**Acceptance Scenarios**:

1. **Given** any supported locale and viewport, **When** a mode page loads,
   **Then** its title, rules, controls, loading state, and error state use the
   same product vocabulary and remain keyboard operable.
2. **Given** an expired, forbidden, missing, or completed session link, **When**
   the player opens it, **Then** a safe state-specific explanation and next action
   are shown without leaking session or location details.
3. **Given** an Arabic locale, **When** any mode flow is used, **Then** navigation,
   cards, forms, score layouts, direction-sensitive icons, and focus order work
   correctly in right-to-left layout.

### Edge Cases

- A player opens a mode URL whose backend mode is a legacy alias or unsupported
  value.
- Quick Play defaults cannot be resolved because no eligible map is available.
- A player already owns or participates in an incompatible active game, queue,
  match, or room.
- A player has an active queue ticket when opening a different matchmaking mode.
- A Duo or Squad player loses party membership, leadership, or eligibility while
  searching.
- A party is full, already started, expired, cancelled, or its code is malformed.
- The host disconnects before start, during a round, or while a command is being
  retried.
- Two clients submit the same create, join, guess, next-round, ready, start,
  leave, or end action concurrently.
- A player refreshes during countdown, active guessing, reveal, round transition,
  terminal scoring, or result finalization.
- Media is unavailable, delayed, unsupported, or lacks required attribution.
- Realtime delivery is delayed or unavailable while authoritative reads remain
  available.
- The Practice history exceeds one page or a session exceeds 1,000 rounds.
- A daily attempt was already completed on another device.
- Locale changes during an active session without changing canonical identifiers.
- Small-screen, 200% zoom, reduced-motion, keyboard-only, and screen-reader use.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST expose exactly these player-facing choices: Classic
  Solo, Practice, Daily Challenge, Quick Play, Party Lobby, Casual Solo, Casual
  Duos, Casual Squads, Ranked Solo, Ranked Duos, and Ranked Squads.
- **FR-002**: The system MUST NOT present legacy compatibility aliases as
  separate player choices.
- **FR-003**: Each choice MUST have a stable, directly addressable localized page
  or page state that preserves the selected mode across refresh and sharing.
- **FR-004**: Each mode entry page MUST explain its objective, team size, timing,
  round lifecycle, progression effect, and eligibility before the player commits.
- **FR-005**: Classic Solo MUST allow valid map, round-count, and timer selection
  and MUST start a fixed-round game owned by the initiating session.
- **FR-006**: Quick Play MUST start with deterministic product defaults in one
  primary action and MUST show the resolved configuration before the first guess.
- **FR-007**: Practice MUST be untimed, owner-only, open-ended, and
  progression-neutral, with immediate reveal, one-at-a-time round advancement,
  paged history, and explicit session ending.
- **FR-008**: Daily Challenge MUST preserve one shared daily definition while
  supporting start, resume, completion, and already-completed states.
- **FR-009**: Casual and Ranked matchmaking MUST support Solo, Duo, and Squad as
  distinct format choices with the correct roster size and playlist rules.
- **FR-010**: Matchmaking pages MUST show not queued, searching, assigned,
  temporarily unavailable, ineligible, conflict, and recoverable failure states.
- **FR-011**: Queue joins, leaves, retries, status recovery, and assignment
  handling MUST NOT create duplicate active tickets or matches.
- **FR-012**: Team matchmaking MUST clearly identify party requirements and MUST
  disable queue entry until the current roster is compatible.
- **FR-013**: Ranked pages MUST explain competitive timing and progression impact;
  Casual pages MUST explain their untimed or relaxed rules and progression impact.
- **FR-014**: Party Lobby MUST support create, join by code, leave, readiness,
  host start, host controls, reconnect, live membership, round play, and final
  standings within existing authorization rules.
- **FR-015**: Party Lobby MUST use its canonical product name while remaining able
  to recover compatible legacy hosted-room records.
- **FR-016**: A shared game screen MUST render the correct mode-specific timer,
  map media, attribution, guess controls, submission state, reveal policy, round
  transition, score presentation, and terminal actions.
- **FR-017**: Multiplayer screens MUST hide actual coordinates, answers, and
  opponent guesses until the authoritative reveal condition is reached.
- **FR-018**: Every mutation that supports idempotency MUST reuse a stable key for
  retries and MUST prevent double submission in the visible UI.
- **FR-019**: Refreshing or reopening a deep link MUST recover the authoritative
  active state or show a safe terminal/not-found/forbidden outcome.
- **FR-020**: Client state MUST be treated as transient presentation state; game,
  queue, party, round, score, and result truth MUST come from authoritative reads.
- **FR-021**: All mode pages MUST define loading, empty, error, disabled, success,
  reconnecting, and stale-state recovery behavior where applicable.
- **FR-022**: All user-facing text, status messages, accessible names, and media
  descriptions MUST be available in English and Arabic.
- **FR-023**: Arabic pages MUST preserve correct right-to-left reading, focus,
  score, grid, icon, and navigation behavior.
- **FR-024**: Every control MUST be keyboard operable, expose an accessible name,
  preserve a visible focus indicator, and announce asynchronous status changes
  when needed.
- **FR-025**: Mode pages MUST remain usable on supported mobile, tablet, and
  desktop widths and at 200% browser zoom.
- **FR-026**: User-visible failures MUST use safe stable categories and MUST NOT
  reveal room secrets, session identifiers, private identity data, hidden
  locations, raw provider errors, or internal diagnostics.
- **FR-027**: Existing completed sessions, daily attempts, legacy hosted rooms,
  and active matchmaking assignments MUST remain recoverable through their
  supported compatibility behavior.
- **FR-028**: The system MUST record enough bounded operational information to
  distinguish mode, operation, outcome, and degraded realtime behavior without
  logging sensitive values.
- **FR-029**: Unsupported mode values and invalid transitions MUST fail safely and
  provide a navigable recovery action.
- **FR-030**: The delivered experience MUST include automated coverage of mode
  selection, contract mapping, transition rules, authorization, localization,
  accessibility, recovery, and primary end-to-end journeys.

### Key Entities

- **Mode Entry**: The player-facing identity, grouping, description, eligibility,
  route, icon, and canonical mode value for a selectable experience.
- **Game Configuration**: The selected map and any mode-appropriate round or timer
  settings used to create a session.
- **Game Session**: The authoritative mode lifecycle, owner/participants, current
  round, totals, timing, and terminal state.
- **Round**: A numbered location challenge with media, timing, reveal status, and
  completion state.
- **Guess**: One participant's submitted location and derived score components for
  one round.
- **Practice History Page**: A bounded ordered slice of completed Practice rounds
  with guesses and revealed answers.
- **Matchmaking Selection**: Playlist, format, team size, and optional party
  context chosen before queue entry.
- **Matchmaking Ticket**: The player's authoritative searching, assigned, or
  unavailable state and its recovery information.
- **Party Lobby**: A hosted room with public code, host, members, settings,
  readiness, realtime status, linked game, and terminal state.
- **Result Summary**: Final individual or team placement, round breakdown, totals,
  progression effect, and safe next actions.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A player can reach any of the eleven supported mode entry states in
  at most two interactions from the authenticated home navigation.
- **SC-002**: In test sessions, at least 95% of players can start an eligible solo
  experience without assistance in under 30 seconds.
- **SC-003**: Queue, lobby, round, reveal, and result state changes are visibly
  reflected within one second of the client receiving authoritative updates.
- **SC-004**: Refresh and reconnection tests recover the correct authoritative
  state in 100% of covered lifecycle checkpoints without duplicate mutations.
- **SC-005**: All eleven mode choices map to exactly one canonical mode and all
  legacy aliases remain absent from player-facing selection.
- **SC-006**: All primary flows can be completed using only a keyboard in English
  and Arabic, with no critical automated accessibility violations.
- **SC-007**: Every primary mode page remains usable at supported mobile, tablet,
  desktop, right-to-left, and 200% zoom test configurations.
- **SC-008**: Hidden locations, private guesses, room secrets, session tokens, and
  private identity data never appear in unauthorized responses, logs, or
  pre-reveal interfaces during security tests.
- **SC-009**: Loading, empty, error, disabled, success, reconnecting, and terminal
  states have deterministic acceptance coverage for every surface where they can
  occur.
- **SC-010**: All required repository quality gates and targeted end-to-end mode
  journeys pass before release.

## Assumptions

- Existing session, map, media, scoring, challenge, room, realtime, matchmaking,
  party, profile, and progression capabilities are reused where their current
  contracts already satisfy the requirement.
- The current Daily Challenge product remains a five-game daily program; start,
  resume, progress, and completed states reflect games played out of five.
- The eleven labels in FR-001 are the product surface; legacy aliases remain
  read-compatible only.
- Quick Play is a distinct one-action product experience even if it reuses solo
  gameplay rules and default configuration internally.
- Duo and Squad matchmaking reuse the existing party roster and leadership model.
- The existing authenticated application shell and localized routing remain the
  common navigation frame for all pages.
- Mode pages may share gameplay and result components when behavior is expressed
  through explicit mode capabilities rather than duplicated screens.
- Monetization, new maps, new scoring formulas, spectator mode, tournaments, and
  social voice/video are outside this feature.
