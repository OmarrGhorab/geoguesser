# Feature Specification: Party Lobby And Practice Modes

**Feature Branch**: `codex/party-lobby-practice-modes`

**Created**: 2026-07-18

**Status**: Draft

**Input**: User description: "Add a Party Lobby mode where a host creates a room, many players join and compete individually in an unranked game with a three-minute default timer; add a normal Practice mode with unlimited rounds, no timer, and no realtime features. Backend only for now."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Host A Large Free-For-All Party Lobby (Priority: P1)

A player creates a shareable Party Lobby, chooses a map and room settings, and allows many players to join by room code. The host starts an unranked free-for-all in which every player competes individually rather than as a Duo or Squad team.

**Why this priority**: This is the primary new social mode and must remain clearly separate from the small premade parties used to enter team matchmaking.

**Independent Test**: Create a lobby, join it from multiple distinct sessions, start the game, complete all configured rounds, and verify that every participant has an individual score and final placement while no competitive rating changes.

**Acceptance Scenarios**:

1. **Given** an eligible host, **When** they create a Party Lobby without specifying a timer, **Then** the lobby uses a 180-second round timer and returns a shareable room code.
2. **Given** a joinable lobby below its player limit, **When** another player submits its room code, **Then** that player joins the same free-for-all roster and all connected participants receive the updated lobby state.
3. **Given** a lobby with at least two participants, **When** the host starts it, **Then** every active participant receives the same round, playable scene, start time, and deadline.
4. **Given** a completed Party Lobby round, **When** results are revealed, **Then** players are ordered by their individual accepted score and no team totals or competitive changes are produced.
5. **Given** the final configured round completes, **When** final results are loaded, **Then** all players receive durable total scores, deterministic placements, and a declared tie where totals are equal.

---

### User Story 2 - Control And Recover A Party Lobby (Priority: P2)

The host controls lobby-only settings, readiness, player removal, and starting. Participants can refresh or reconnect without creating duplicate roster positions, and the active game continues according to its authoritative timer.

**Why this priority**: Large social rooms need reliable authority, recovery, and capacity behavior to avoid stuck or unfair sessions.

**Independent Test**: Exercise host and non-host commands, capacity boundaries, duplicate joins, disconnect/reconnect, deadline expiry, and refresh recovery while verifying one authoritative lobby and game state.

**Acceptance Scenarios**:

1. **Given** a lobby has not started, **When** the host changes the map, round count, timer, or maximum players within allowed bounds, **Then** the updated settings are visible to every participant.
2. **Given** a non-host participant, **When** they attempt a host-only operation, **Then** the operation is rejected and lobby state remains unchanged.
3. **Given** a participant refreshes or reconnects with the same identity, **When** they reload the lobby, **Then** their existing roster position and score are restored rather than duplicated.
4. **Given** the room reaches its configured capacity, **When** another new player tries to join, **Then** the join is rejected without affecting existing participants.
5. **Given** a player misses a Party Lobby deadline, **When** the round closes, **Then** that player receives zero for the round and does not block everyone else.

---

### User Story 3 - Play Unlimited Private Practice (Priority: P1)

A player selects a map and starts a private Practice session. They can make a guess, see the answer and score immediately, and request another round for as long as they want without a timer, matchmaking, room, party, chat, spectating, or realtime connection.

**Why this priority**: Practice provides a low-pressure learning loop that is operationally simple and independent of multiplayer availability.

**Independent Test**: Start one Practice session, complete more rounds than the normal fixed-round maximum, reload between rounds, and verify that each accepted guess reveals immediately, the next round remains available, and no competitive or multiplayer state is created.

**Acceptance Scenarios**:

1. **Given** an eligible player and valid map, **When** they start Practice, **Then** one untimed playable round is created without joining matchmaking or opening a realtime channel.
2. **Given** an active Practice round, **When** the player submits a valid guess, **Then** the guess is locked and scored, the answer is revealed immediately, and the response indicates that another round is available.
3. **Given** a completed Practice round, **When** the player requests the next round, **Then** exactly one new untimed round is appended with a sequential round number.
4. **Given** duplicate next-round or guess requests, **When** the same idempotency identity is replayed, **Then** the original result is returned and no duplicate round or guess is created.
5. **Given** a Practice session has completed many rounds, **When** the player reloads it, **Then** the accumulated score, completed-round history, and current round remain recoverable.

---

### User Story 4 - Keep New Modes Separate And Safe (Priority: P2)

Players can clearly distinguish Party Lobby, premade matchmaking parties, and private Practice state. Unauthorized users cannot inspect another player's Practice session or a lobby they have not joined.

**Why this priority**: The word "party" already represents a matchmaking group, and the new modes must not leak data or accidentally affect ranked progression.

**Independent Test**: Attempt cross-mode operations and unauthorized reads, then verify privacy-safe errors, unchanged competitive standings, and no cross-mode queue or party membership effects.

**Acceptance Scenarios**:

1. **Given** a player belongs to a Duo or Squad matchmaking party, **When** they create or join a Party Lobby, **Then** the lobby is represented as a separate hosted room and does not alter the premade party roster.
2. **Given** a Party Lobby or Practice game completes, **When** competitive profile and history are read, **Then** no rating, division, placement, or Top 500 fact has changed.
3. **Given** a non-participant requests lobby gameplay state or another player's Practice session, **When** authorization is evaluated, **Then** the request is rejected without exposing whether hidden rounds, guesses, or answers exist.
4. **Given** a backend dependency needed only for realtime delivery is unavailable, **When** a player uses Practice, **Then** the Practice loop remains available as long as core game persistence and map media are healthy.

### Edge Cases

- The host submits start twice or two host tabs submit start concurrently.
- Players join while the host starts or changes the maximum capacity.
- A room has tied individual scores, including multi-way ties.
- A Party Lobby player submits at the exact deadline.
- One or more Party Lobby participants disconnect before or after submitting.
- The host disconnects while the lobby is waiting or while a timed game is active.
- A client retries a successful guess, next-round request, room join, or start command.
- A Practice player requests a next round before completing the current one.
- A Practice session accumulates enough rounds that unbounded response payloads would become expensive.
- A map has no additional usable location for the next Practice round.
- A player attempts realtime tickets, team chat, markers, spectating, matchmaking, or ranked operations using a Practice game.
- Hidden answers and other players' unrevealed guesses remain absent from pre-reveal room state.
- Backend responses preserve stable mode names and error codes for future English, Arabic, RTL, loading, empty, error, and success frontend states.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST expose `party_lobby` as an unranked hosted-room game mode that is distinct from premade Solo, Duo, and Squad matchmaking parties.
- **FR-002**: Eligible guest and signed-in sessions MUST be able to create and join Party Lobbies using the existing hosted-room identity and room-code model.
- **FR-003**: A Party Lobby MUST support between 2 and 50 active players, with a host-selected maximum inside that range.
- **FR-004**: A Party Lobby MUST default to five rounds and a 180-second server-authoritative timer when those settings are omitted.
- **FR-005**: Before start, the host MUST be able to select a valid map, 1 to 10 rounds, a 10 to 600 second round timer, and a 2 to 50 player capacity that is not below the active roster size.
- **FR-006**: Party Lobby gameplay MUST be free-for-all: every player owns one guess per round, has an individual round and total score, and has no team slot or team score.
- **FR-007**: Party Lobby rounds MUST close when every eligible active player submits or the shared deadline expires, whichever happens first.
- **FR-008**: Missing Party Lobby guesses MUST score zero and MUST NOT block round or game progression after the deadline.
- **FR-009**: Party Lobby result ordering MUST use total score descending, then total distance ascending, then earliest lobby join time, then stable player identifier to resolve otherwise equal positions deterministically; exact equal score and distance MUST remain visibly representable as a tie even when deterministic ordering is required.
- **FR-010**: Party Lobby creation, joining, settings, readiness, removal, start, presence, reconnect, round progress, reveal, and final results MUST remain realtime and reloadable from authoritative state.
- **FR-011**: Party Lobby commands MUST preserve host authorization, capacity enforcement, identity-safe reconnect, rate limiting, idempotency, and hidden-answer protections.
- **FR-012**: Party Lobby games MUST NOT create or update competitive standings, placements, ranks, rating history, Top 500 membership, ranked matchmaking tickets, or team-chat state.
- **FR-013**: The system MUST expose `practice` as a private single-player game mode available through normal game APIs without matchmaking, hosted rooms, parties, or realtime tickets.
- **FR-014**: Practice MUST create one active round initially and MUST have no timer, deadline, fixed terminal round count, or automatic completion caused by a round-number limit.
- **FR-015**: A Practice guess MUST use the normal 0 to 5,000 geography accuracy score, have no speed bonus, lock after acceptance, and reveal the answer immediately.
- **FR-016**: After a Practice guess is accepted, the owning player MUST be able to request exactly one next round, which receives the next sequential round number and no deadline.
- **FR-017**: The system MUST reject a Practice next-round request while the current round is incomplete and MUST return the existing current round when a concurrent or idempotent replay already created it.
- **FR-018**: Practice sessions MUST support an unbounded user-facing sequence of rounds while keeping list and history responses bounded through pagination or explicit limits.
- **FR-019**: Practice MUST preserve accumulated score and durable round history until the player explicitly ends or abandons the session; ending a session MUST prevent additional rounds without deleting its history.
- **FR-020**: Only the registered user or guest identity that owns a Practice session MUST be able to read it, submit guesses, create the next round, or end it.
- **FR-021**: Practice MUST NOT require Redis, WebSockets, realtime presence, background deadline workers, chat, images, markers, spectating, rooms, or matchmaking.
- **FR-022**: Practice games MUST NOT affect competitive progression, matchmade statistics, daily challenges, missions, public leaderboards, or win/loss records.
- **FR-023**: Mode validation, persistence constraints, public contracts, operational metrics, and documentation MUST recognize both new modes without weakening legacy mode compatibility.
- **FR-024**: Logs and telemetry MUST omit exact private guesses, hidden answer coordinates, room codes, guest identity material, and authentication data.
- **FR-025**: All state-changing operations MUST be concurrency-safe and retry-safe, with stable privacy-preserving errors for unauthorized or invalid cross-mode operations.
- **FR-026**: Backend contracts MUST provide stable fields and error codes needed for future localized English and Arabic clients, without adding frontend implementation in this feature.

### Key Entities

- **Party Lobby**: A hosted room with a shareable code, host, map, round count, timer, player capacity, lifecycle state, and associated free-for-all game.
- **Party Lobby Participant**: A guest or registered-player snapshot with membership, presence, readiness, join ordering, individual score, and final placement.
- **Practice Session**: A private single-player game whose round sequence can grow until explicitly ended and whose score and history are durable.
- **Practice Round**: One untimed map location in a Practice session, created only after the previous round is completed.
- **Practice Guess**: The owning player's single locked guess for a Practice round, using accuracy-only scoring and immediate answer reveal.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A host can create a Party Lobby with default settings, share its code, and start a two-or-more-player game without interacting with matchmaking parties.
- **SC-002**: Party Lobby tests support 50 active participants while maintaining exactly one roster position and one accepted guess per participant per round.
- **SC-003**: 95% of authorized lobby joins, settings changes, readiness changes, round progress changes, and result changes become observable to connected participants within one second under normal conditions.
- **SC-004**: Every Party Lobby participant receives the same authoritative 180-second default deadline, and 100% of late or missing guesses are prevented from altering an already closed round.
- **SC-005**: Party Lobby final standings reproduce the same deterministic ordering across reloads in 100% of tie and concurrency test cases.
- **SC-006**: A Practice player can complete at least 1,000 sequential rounds in one logical session without encountering a fixed round-count limit or receiving an unbounded history response.
- **SC-007**: A Practice guess and next-round transition can be completed without any realtime connection, and remain usable when realtime-only dependencies are unavailable.
- **SC-008**: Unauthorized and cross-mode access tests expose no hidden answers, private guesses, room membership details, or another player's Practice state.
- **SC-009**: Party Lobby and Practice completion produce zero competitive rating, placement, rank, Top 500, mission, daily, public leaderboard, or matchmade-stat changes in all progression tests.
- **SC-010**: Backend create/read/guess/advance/end operations respond within 300 ms at p95 on representative local fixtures, excluding external map-media latency; Party Lobby final standings use bounded reads for at most 50 players.
- **SC-011**: All targeted unit, contract, integration, security, concurrency, migration, vet, lint, and full backend test gates pass, or exact environmental blockers and residual risks are documented before release.

## Assumptions

- "Party Lobby" is the product name for the existing hosted private-room concept, not the small premade Party used by Duo and Squad matchmaking.
- Party Lobby v1 remains join-by-code and supports both guests and signed-in users, matching the existing hosted-room identity rules.
- The existing room capacity ceiling of 50 is sufficient for "lots of players" in v1.
- Party Lobby defaults are five rounds, a 180-second timer, and host-configurable settings within existing safe bounds.
- Party Lobby has realtime room and game updates but no team chat, image chat, teammate markers, team spectating, or competitive progression.
- Practice is available to guests and signed-in users through the existing session identity model.
- Practice is private and progression-neutral; it exists for learning rather than farming stats, missions, achievements, or leaderboards.
- "Infinite" means no product round limit while the session is active. Storage and list endpoints may use pagination and operational retention without presenting a fixed gameplay endpoint.
- Practice next-round creation is client-driven after each completed guess, avoiding timers and background workers.
- This feature is backend-only. Future frontend work will provide localized English/Arabic and RTL presentation using the stable contracts created here.
