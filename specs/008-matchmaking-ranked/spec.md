# Feature Specification: Matchmaking And Ranked Foundations

**Feature Branch**: `[008-matchmaking-ranked]`

**Created**: 2026-07-11

**Status**: Draft

**Input**: User description: "Matchmaking and ranked"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Join Ranked Matchmaking (Priority: P1)

A registered player can enter ranked matchmaking for a supported game mode and receive a clear waiting state while the system searches for one compatible opponent. The player cannot occupy multiple queues or create duplicate queue entries by retrying the request.

**Why this priority**: Entering a trustworthy server-controlled queue is the minimum useful foundation for ranked play and every later competitive feature.

**Independent Test**: Can be fully tested by signing in as an eligible registered player, joining a supported queue, checking the waiting state repeatedly, and confirming that only one active queue entry exists.

**Acceptance Scenarios**:

1. **Given** an eligible registered player who is not queued or playing, **When** they join a supported ranked queue, **Then** they receive a waiting status that identifies the selected mode and the time at which the search began.
2. **Given** a player already waiting in a queue, **When** the same player retries the join action, **Then** the existing queue state is returned without creating a duplicate entry or resetting their waiting priority.
3. **Given** a guest, disabled player, unsupported mode, or player already committed to another active match, **When** they try to join, **Then** the request is rejected with a safe, understandable outcome and no queue entry is created.

---

### User Story 2 - Form A Fair Match (Priority: P1)

Two compatible queued players are paired into one ranked match, removed from active matchmaking, and given the same authoritative match destination. The match remains recognizable after refresh or reconnection and cannot be assigned to either player twice.

**Why this priority**: Match formation is the core value of matchmaking; a queue that cannot safely produce a durable game is not useful.

**Independent Test**: Can be fully tested by placing two compatible registered players into the same queue, forming a match, and confirming that both players see one shared match assignment while neither remains queued.

**Acceptance Scenarios**:

1. **Given** two eligible compatible players waiting in the same mode, **When** a match is formed, **Then** exactly one durable match is created with both players and both queue entries are closed.
2. **Given** both matched players check their matchmaking status, **When** match formation has completed, **Then** each receives the same match assignment and can proceed to the authoritative game or lobby.
3. **Given** match formation is retried or processed concurrently, **When** completion is recorded, **Then** each player belongs to at most one active match and no duplicate match is created.
4. **Given** a queued player becomes ineligible before assignment, **When** pairing is attempted, **Then** that player is not placed into a new match and the other eligible player remains safely searchable or receives a recoverable state.

---

### User Story 3 - Leave Or Recover Queue State (Priority: P2)

A waiting player can leave matchmaking before assignment, inspect their current state after a refresh or temporary disconnection, and recover cleanly from expired or unavailable queue state.

**Why this priority**: Players need control over waiting and must not become trapped, unknowingly queued, or uncertain whether a match was found.

**Independent Test**: Can be fully tested by joining a queue, refreshing and checking status, leaving it, and confirming the player is no longer eligible for pairing.

**Acceptance Scenarios**:

1. **Given** a player waiting in matchmaking, **When** they leave the queue before assignment, **Then** their queue entry is closed and they cannot be selected for a later match from that entry.
2. **Given** a player who is not queued, **When** they request to leave, **Then** the operation succeeds safely without changing another player's state.
3. **Given** a queued or newly matched player refreshes or reconnects, **When** they check status, **Then** they receive one of a small set of explicit states: not queued, searching, matched, or temporarily unavailable.
4. **Given** a queue entry has expired or become inconsistent, **When** status is checked, **Then** stale state is not presented as active and the player can join again safely.

---

### User Story 4 - Preserve Ranked Match Records (Priority: P2)

Players and future ranked progression features can rely on a durable record of who was matched, which competitive mode was selected, when the match was formed, and which game represents its outcome.

**Why this priority**: Ranked ratings, seasons, dispute investigation, and competitive history require stable match facts even though rating calculation is outside this feature.

**Independent Test**: Can be fully tested by forming and completing a match, then confirming its participants, mode, lifecycle, and linked game remain stable and are not altered by queue retries.

**Acceptance Scenarios**:

1. **Given** a ranked match is formed, **When** its record is inspected later, **Then** it retains its unique identity, participants, mode, timestamps, lifecycle state, and linked game.
2. **Given** a formed match reaches a terminal game outcome, **When** the outcome is associated with the match, **Then** the association is recorded once and remains suitable for later rating calculation.
3. **Given** a match is cancelled or cannot start, **When** its lifecycle closes, **Then** it is distinguishable from a completed competitive result and cannot be counted as a ranked outcome by future progression features.

---

### User Story 5 - Understand Matchmaking States Accessibly (Priority: P3)

Players receive localized and accessible feedback for joining, searching, matching, leaving, eligibility failures, delays, and service errors in both supported languages.

**Why this priority**: Clear state communication prevents accidental duplicate actions and makes a time-sensitive competitive flow usable for keyboard, screen-reader, English, and Arabic users.

**Independent Test**: Can be fully tested by exercising each matchmaking state in English and Arabic using keyboard navigation and assistive status announcements.

**Acceptance Scenarios**:

1. **Given** matchmaking is searching, matched, cancelled, delayed, or unavailable, **When** the state changes, **Then** the player receives localized, non-color-only feedback with an appropriate next action.
2. **Given** the interface uses Arabic, **When** matchmaking states and controls are displayed, **Then** the meaning matches English and the layout follows right-to-left expectations.
3. **Given** a player uses only a keyboard or screen reader, **When** they join, leave, or become matched, **Then** controls have accessible names and important state changes are announced without unexpected focus loss.

### Edge Cases

- A player submits simultaneous join requests from multiple tabs or devices.
- A player tries to join a second mode while already searching or assigned to a match.
- A leave request races with match assignment.
- Two matching processes try to claim the same player at the same time.
- One selected player becomes disabled, signs out, disconnects, or starts another game before assignment finishes.
- A newly formed game or lobby cannot be created after players have been selected.
- Queue state expires while the player is actively viewing the search screen.
- A player reconnects after assignment but before entering the linked game.
- The matchmaking service is temporarily unavailable while durable match data remains available.
- Search takes longer than expected because no compatible opponent is available.
- Users repeatedly join and leave in an attempt to manipulate priority or overload matchmaking.
- Loading, searching, matched, empty, cancelled, unavailable, ineligible, rate-limited, and retry states need distinct user feedback.
- English and Arabic experiences must convey equivalent queue and match states, with usable right-to-left layout.
- Waiting-state updates that take more than 2 seconds to appear should be treated as degraded; match formation may take longer depending on player availability.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow eligible registered players to join matchmaking for a supported ranked mode.
- **FR-002**: The system MUST deny guests, disabled accounts, and otherwise ineligible players without creating queue or match records.
- **FR-003**: A player MUST have no more than one active matchmaking entry across all modes at a time.
- **FR-004**: Repeated or concurrent join requests from the same player MUST return or converge on the same active queue state without changing the player's original waiting priority.
- **FR-005**: A waiting player MUST be able to leave matchmaking before match assignment becomes final.
- **FR-006**: Repeated leave requests MUST be safe and MUST NOT affect another player or a match that has already been finalized.
- **FR-007**: Players MUST be able to retrieve a current status of not queued, searching, matched, or temporarily unavailable.
- **FR-008**: Searching status MUST identify the selected mode and search start time without exposing other queued players.
- **FR-009**: The system MUST only pair players who are eligible, searching in compatible modes, and not already committed to another active match.
- **FR-010**: Match assignment MUST be authoritative, and all participants in a formed match MUST receive the same match and game or lobby destination.
- **FR-011**: Match formation MUST create no more than one active match for each participating player, including during retries or concurrent processing.
- **FR-012**: Finalized match formation MUST remove or close the participants' active queue entries so they cannot be selected again.
- **FR-013**: If match formation fails before finalization, the system MUST avoid leaving players simultaneously marked as matched and searchable and MUST provide a recoverable outcome.
- **FR-014**: Stale or expired queue entries MUST stop being eligible for matching and MUST NOT prevent a player from safely joining again.
- **FR-015**: The system MUST preserve a durable ranked match record containing its identity, participants, competitive mode, lifecycle state, formation time, and associated game when one exists.
- **FR-016**: Each ranked match participant MUST be represented once per match and MUST have a stable relationship to the resulting game outcome.
- **FR-017**: Cancelled, failed-to-start, and completed matches MUST remain distinguishable so future rating logic cannot treat a non-result as a completed ranked outcome.
- **FR-018**: Queue and match state changes MUST be observable for operational diagnosis without exposing private player data, authentication data, hidden locations, or precise guesses.
- **FR-019**: The system MUST limit abusive join, leave, and status traffic and provide a recoverable rate-limited outcome.
- **FR-020**: User-facing matchmaking copy MUST be available in English and Arabic, including loading, searching, matched, empty, cancelled, unavailable, ineligible, rate-limited, and retry states.
- **FR-021**: Matchmaking controls MUST expose accessible names, keyboard focus behavior, disabled states, and non-color-only status indicators, and important asynchronous state changes MUST be announced accessibly.
- **FR-022**: External contracts and durable-data changes MUST be reflected in the feature's planning artifacts before implementation is considered complete.
- **FR-023**: Matchmaking behavior MUST preserve existing server-authoritative game rules, including hidden-location protections and participant authorization.

### Key Entities *(include if feature involves data)*

- **Queue Entry**: A player's current request to find a compatible match, including player, ranked mode, lifecycle state, search start, expiry, and enough identity to prevent duplicate or stale assignment.
- **Ranked Match**: The durable competitive pairing formed by matchmaking, including unique identity, mode, lifecycle state, formation and completion timestamps, and its associated game or lobby.
- **Match Participant**: A registered player's unique membership in a ranked match, including participant state and relationship to the eventual competitive outcome.
- **Matchmaking Status**: The public-safe current state returned to a player: not queued, searching, matched, or temporarily unavailable, plus the limited details needed for the next action.
- **Competitive Mode**: An approved ruleset or queue category that determines whether two searching players are compatible; this feature begins with the existing standard competitive geography ruleset.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 95% of eligible players see confirmation that they joined or left matchmaking within 1 second under normal operating conditions.
- **SC-002**: 95% of status checks show the player's latest queue or match state within 1 second under normal operating conditions.
- **SC-003**: When two compatible eligible players are available, 95% of match assignments are visible to both players within 3 seconds.
- **SC-004**: In all retry and concurrency tests, no player has more than one active queue entry or more than one active ranked match assignment.
- **SC-005**: 100% of finalized matches retain one durable participant record per player and one stable association to their resulting game.
- **SC-006**: 100% of cancelled or failed-to-start matches remain excluded from completed-ranked-result eligibility.
- **SC-007**: 100% of guest, disabled, duplicate, incompatible-mode, and already-matched attempts produce a safe outcome without corrupting another player's state.
- **SC-008**: A player who refreshes or reconnects during searching or immediately after assignment can recover the correct actionable state in 100% of acceptance tests.
- **SC-009**: All matchmaking states and controls have equivalent English and Arabic presentation and pass keyboard and accessible-name checks before release.
- **SC-010**: All required backend, contract, integration, security, and user-flow validation gates pass before the feature is considered complete.

## Assumptions

- Matchmaking is limited to registered, eligible players because ranked identity and durable outcomes require an account.
- The initial release pairs two players for the existing standard competitive geography ruleset; larger teams, parties, and tournaments are outside this feature.
- Compatibility initially requires the same supported competitive mode. Skill-rating range expansion, geographic region selection, and latency-based pairing may be added later without changing the core queue lifecycle.
- This feature records ranked-ready match facts but does not calculate or display rating changes, divisions, placement matches, season standings, rewards, or decay.
- Duel-specific health, damage, elimination, and combat rules require a separate specification and are outside this feature.
- Existing authentication, profiles, private-room/realtime behavior, server-authoritative gameplay, and leaderboard foundations are reused as dependencies.
- A match assignment becomes final only after one durable match and its game or lobby destination can be recovered by both participants.
- Players may wait indefinitely when no compatible opponent exists, but queue entries require expiry or heartbeat rules so abandoned searches do not remain matchable forever.
