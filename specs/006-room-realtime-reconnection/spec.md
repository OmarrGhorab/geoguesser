# Feature Specification: Room Realtime Reconnection

**Feature Branch**: `006-room-realtime-reconnection`

**Created**: 2026-06-30

**Status**: Draft

**Input**: User description: "phase-06-private-rooms-realtime-and-reconnection.md; implement realtime communication within the room/channel"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Host A Private Realtime Room (Priority: P1)

A guest or signed-in host can create a private room, receive a non-guessable room code, share it with another player, and see lobby membership update live as players join, disconnect, reconnect, or leave.

**Why this priority**: Private rooms are the foundation for multiplayer play; without a synchronized lobby, players cannot coordinate or trust the start state.

**Independent Test**: Can be fully tested by creating a room in one browser session, joining it from a second session, and confirming both sessions show the same room code, settings, host identity, player list, presence status, and room state without manual refresh.

**Acceptance Scenarios**:

1. **Given** a host creates a private room, **When** the room is created, **Then** the host receives a room code, shareable room state, and an active host presence.
2. **Given** a second player enters a valid room code, **When** the join succeeds, **Then** both host and player see the same lobby roster and the join event appears without a manual refresh.
3. **Given** a player disconnects in the lobby, **When** their presence expires, **Then** connected clients see that player as disconnected or removed according to the lobby rules.
4. **Given** a player rejoins with the same identity during the allowed window, **When** the room state reloads, **Then** the same room membership is restored rather than creating a duplicate player.

---

### User Story 2 - Enforce Host Controls And Room Rules (Priority: P1)

The host can update lobby settings, remove players, and start the room while non-host players can view state and join play but cannot mutate privileged room controls.

**Why this priority**: Multiplayer fairness depends on a server-owned authority model where one host controls setup and other players cannot start or change rules unexpectedly.

**Independent Test**: Can be fully tested by attempting room setting changes, player removal, and start commands as both host and non-host sessions, then confirming only host-authorized actions succeed and all connected clients receive the updated state.

**Acceptance Scenarios**:

1. **Given** the room is still in the lobby, **When** the host changes map, round count, timer, or player limit within allowed values, **Then** every connected client sees the updated settings.
2. **Given** a non-host player attempts to change settings or start the room, **When** the command is submitted, **Then** the command is rejected and the room state remains unchanged.
3. **Given** the host removes a player from the lobby, **When** removal succeeds, **Then** the removed player can no longer participate in that room and connected clients see the updated roster.
4. **Given** the room has already started, **When** any player attempts to change lobby-only settings, **Then** the system rejects the change and keeps the active game rules locked.

---

### User Story 3 - Play Synchronized Multiplayer Rounds (Priority: P1)

Players in an active private room receive synchronized round starts, shared countdowns, guess progress updates, round results, next-round transitions, and final results while the backend remains authoritative for timing and scoring.

**Why this priority**: The main value of realtime rooms is playing together; synchronized rounds and result transitions make the experience feel shared and fair.

**Independent Test**: Can be fully tested by starting a two-player room, submitting guesses from both sessions, and confirming round start time, remaining time, guess count, result reveal, next round, and final score stay consistent across both clients.

**Acceptance Scenarios**:

1. **Given** the host starts a room with at least two players, **When** the game begins, **Then** every active player receives the same round number, start time, deadline, media, and hidden-coordinate-safe state.
2. **Given** one player submits a guess during an active round, **When** the guess is accepted, **Then** all clients see aggregate guess progress without revealing hidden answers early.
3. **Given** all active players submit before the deadline, **When** the final required guess is accepted, **Then** the round ends early for everyone and results are revealed consistently.
4. **Given** the round deadline passes before all players submit, **When** the system closes the round, **Then** late guesses are rejected and all clients transition to the same result or next-round state.
5. **Given** the final round completes, **When** results are shown, **Then** every player can reload and see durable final scores and per-round outcomes.

---

### User Story 4 - Recover From Reconnects And Refreshes (Priority: P2)

A player can refresh the page or briefly lose their realtime connection and return to the current room, round, submitted guess state, and scoreboard without corrupting gameplay or duplicating their participant slot.

**Why this priority**: Browser refreshes and network hiccups are common; multiplayer should survive them without making the room unfair or stuck.

**Independent Test**: Can be fully tested by disconnecting one player during lobby and active round states, reconnecting before and after the round deadline, and confirming restored state, preserved guesses, missed-round scoring, and roster status.

**Acceptance Scenarios**:

1. **Given** a player refreshes during the lobby, **When** the page reloads with the same identity, **Then** the player returns to the same lobby membership and current room state.
2. **Given** a player disconnects during an active round before guessing, **When** they reconnect before the deadline, **Then** they can continue the same round and submit one valid guess.
3. **Given** a player has already submitted a guess, **When** they disconnect and reconnect, **Then** their submitted guess remains preserved and cannot be duplicated.
4. **Given** a player remains disconnected until a round closes, **When** the room advances, **Then** that player receives no score for the missed round without blocking other players.
5. **Given** a realtime event arrives out of order or after reconnect, **When** the client detects stale state, **Then** it discards stale local state and reloads the authoritative room/game state.

---

### User Story 5 - Handle Room Failure And Recovery States (Priority: P3)

Players see clear localized states for invalid room codes, expired rooms, full rooms, authorization failures, reconnecting, disconnected, kicked, loading, empty, disabled, success, and unexpected errors.

**Why this priority**: Recovery states make the feature usable when live multiplayer inevitably hits timing, network, and permission edge cases.

**Independent Test**: Can be fully tested by exercising each failure or boundary condition and confirming the user sees a localized, accessible state with an appropriate next action.

**Acceptance Scenarios**:

1. **Given** a player enters an invalid or expired room code, **When** the room cannot be joined, **Then** the player sees a safe localized error state and a way back to room creation or join.
2. **Given** a room is full, active, completed, expired, or cancelled, **When** a new player tries to join, **Then** the join is rejected with a clear state-specific message.
3. **Given** the realtime connection drops, **When** the client is trying to reconnect, **Then** the UI announces reconnecting status and disables unsafe actions until authoritative state is restored.
4. **Given** Arabic is selected, **When** room lobby, gameplay, realtime, and recovery states are shown, **Then** visible copy is localized and layout supports RTL.

### Edge Cases

- Two hosts or duplicated tabs attempt to start the same room at the same time.
- A non-host attempts host-only commands while the host is disconnected.
- The host disconnects in lobby before the game starts.
- A player is kicked while connected, disconnected, or attempting to reconnect.
- A player refreshes during room start, during a round deadline, during result reveal, and after game completion.
- A player submits a guess at nearly the same moment the round deadline passes.
- All players submit early while one player disconnects at the same time.
- Realtime events are duplicated, delayed, missing, or received out of order.
- The cached room snapshot is stale compared with durable game facts.
- Live update delivery is temporarily unavailable while room commands are still reachable.
- A room code is guessed repeatedly by an abusive actor.
- A room reaches player limit, expires, is cancelled, or completes while another player is joining.
- Hidden location coordinates and provider metadata must remain unavailable before a player guess or authorized reveal.
- English and Arabic experiences need localized lobby, presence, timer, reconnect, kicked, full, expired, empty, error, disabled, and success states, including RTL layout for Arabic.
- Room updates should feel immediate; delays longer than 2 seconds need visible loading, reconnecting, or stale-state recovery indicators.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow eligible guests and signed-in users to create private rooms with a host identity, map selection, round count, timer setting, maximum player count, and room expiration boundary.
- **FR-002**: System MUST generate private room codes that are uppercase, non-enumerable, unique among joinable rooms, and safe to share.
- **FR-003**: System MUST allow eligible guests and signed-in users to join private rooms by code while preventing joins to full, expired, cancelled, completed, or non-rejoinable active rooms.
- **FR-004**: System MUST prevent the same registered user or guest session from occupying duplicate player slots in the same room.
- **FR-005**: System MUST preserve room membership history and current participant identity across refreshes and reconnects.
- **FR-006**: System MUST expose an authoritative room state that includes room status, settings, host, participants, presence status, current game state when started, and a monotonic state version.
- **FR-007**: System MUST deliver live room/channel updates for lobby membership, presence, settings changes, host commands, room start, round start, guess progress, round end, result reveal, game completion, reconnect state, and player removal.
- **FR-008**: Realtime events MUST be treated as versioned hints; clients MUST be able to recover by reloading authoritative room/game state after reconnect, missing events, duplicate events, or version mismatch.
- **FR-009**: System MUST authorize host-only actions server-side, including settings changes, starting the room, and removing players.
- **FR-010**: System MUST reject non-host attempts to mutate host-only room controls with a stable forbidden error and unchanged room state.
- **FR-011**: System MUST lock gameplay settings once a room leaves the lobby state.
- **FR-012**: System MUST start private room games from one shared server-owned start moment and provide the same round start time, deadline, round number, and playable media to all active players.
- **FR-013**: System MUST calculate countdown and late-guess eligibility from server-owned timestamps, not client clocks.
- **FR-014**: System MUST accept at most one guess per player per round and preserve accepted guesses through disconnects, reconnects, refreshes, and duplicate submissions.
- **FR-015**: System MUST reject late guesses and guesses for non-current rounds without changing scores or room state incorrectly.
- **FR-016**: System MUST advance multiplayer rounds when all active eligible players submit or when the server-owned deadline passes.
- **FR-017**: System MUST assign 0 points for a timed round to an active player who misses the round deadline without an accepted guess.
- **FR-018**: System MUST keep completed game results, per-player scores, per-round outcomes, and final status reloadable after the realtime session ends.
- **FR-019**: System MUST track player presence through heartbeats or equivalent liveness signals and mark players disconnected after a missed-presence threshold.
- **FR-020**: System MUST provide a reconnect grace window of 30 seconds by default for private room presence recovery.
- **FR-021**: System MUST allow a player to reconnect only when the current registered user session or guest identity matches the original room participant.
- **FR-022**: System MUST prevent clients from taking over another participant by submitting another player identifier during reconnect or room commands.
- **FR-023**: System MUST define host disconnect behavior: during lobby, host control is transferred to the earliest joined active player after the reconnect grace window; during active gameplay, host disconnect does not pause round timers or block round progression.
- **FR-024**: System MUST preserve completed guesses, completed rounds, and final scores when a player disconnects or fails to reconnect.
- **FR-025**: System MUST provide safe behavior when live update delivery is unavailable: room commands remain authoritative, clients display degraded/reconnecting state, and room/game state can be manually or automatically refreshed.
- **FR-026**: System MUST hide exact location coordinates, location identifiers, and provider metadata that would reveal the answer until the relevant player has submitted, the round is revealed, or final results are authorized.
- **FR-027**: System MUST rate limit room creation, room join attempts, host commands, realtime connection attempts, and guess submission to reduce abuse and room-code enumeration.
- **FR-028**: System MUST provide localized visible text for English and Arabic room, lobby, gameplay, presence, reconnect, loading, empty, error, disabled, kicked, full, expired, cancelled, success, and result states.
- **FR-029**: Interactive controls MUST expose accessible names, keyboard focus behavior, disabled states, and non-color-only status indicators.
- **FR-030**: System MUST update contract planning artifacts for room commands, room state responses, live event envelopes, reconnect behavior, error codes, and hidden-coordinate guarantees.
- **FR-031**: System MUST record operationally useful but privacy-safe logs and metrics for active rooms, active realtime connections, joins, disconnects, reconnects, room starts, round transitions, guess submission, rejected late guesses, and realtime delivery failures.

### Key Entities *(include if feature involves data)*

- **Room**: A private multiplayer lobby and game container with a code, host, status, settings, player limit, expiration, and optional linked game.
- **Room Participant**: A registered or guest player occupying one slot in a room, with display name, role, membership status, presence status, and linked game participant identity.
- **Room Settings**: The locked setup values for a private room, including map, round count, timer, maximum players, and any future fairness-affecting rules.
- **Room State Snapshot**: The current authoritative state returned to clients, including room metadata, roster, settings, version, active game/round summary, and safe visibility fields.
- **Realtime Event**: A versioned room/channel notification that tells connected clients which room state changed without replacing authoritative state.
- **Presence Record**: Short-lived liveness state for each connected participant, including heartbeat freshness, connection status, and reconnect eligibility.
- **Reconnect Window**: The time-limited period during which a disconnected participant can reclaim their room slot using the same identity.
- **Private Room Game**: A multiplayer game created from room settings, with shared rounds, synchronized timers, submitted guesses, scores, and final results.
- **Round Progress**: Aggregate per-round state showing current round number, timing, submitted guess count, eligible player count, reveal status, and transition state without leaking hidden answers early.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A host and one invited player can create, join, start, play all rounds, and view final private room results in 100% of happy-path validation runs.
- **SC-002**: Lobby membership, settings updates, room start, round start, guess progress, round end, and final result updates appear on connected clients within 2 seconds in at least 95% of normal sessions.
- **SC-003**: Host-only actions are rejected for non-host participants in 100% of authorization validation cases.
- **SC-004**: Two players in the same active room receive identical round start timestamps and deadlines in 100% of synchronization validation cases.
- **SC-005**: Late guesses, duplicate guesses, and guesses for non-current rounds are rejected or handled idempotently in 100% of validation cases.
- **SC-006**: A player refreshing or reconnecting within the grace window returns to the correct room, round, guess, and score state in 100% of validation cases.
- **SC-007**: A player disconnected through a round deadline receives 0 points for that round without blocking the remaining players in 100% of validation cases.
- **SC-008**: Hidden coordinates and answer-revealing metadata are absent from pre-reveal room, round, and realtime payloads in 100% of contract/security validation cases.
- **SC-009**: Room code abuse protections reject repeated invalid join attempts without affecting legitimate joins in abuse validation.
- **SC-010**: English and Arabic room, lobby, reconnect, gameplay, result, empty, error, disabled, and success states pass localization, RTL, keyboard, and screen-reader review.
- **SC-011**: All required validation gates for the feature pass before release, or any omitted gate has a documented blocker and residual risk in planning artifacts.

## Assumptions

- Private rooms are in scope for this phase; public matchmaking, teams, spectators, voice/video, free-form chat, and advanced host controls are out of scope unless later specified.
- Guests and signed-in users can create and join private rooms; persistent profile and account-only features remain limited to signed-in users.
- Room commands such as create, join, settings, start, remove player, and guess submission are authoritative writes; realtime communication is used for room/channel updates and recovery hints.
- The realtime transport choice can be finalized during planning, as long as the user experience supports bidirectional-enough room updates, presence, and reconnect behavior.
- Ready-state controls are required for this phase as live room state, but the host can still start without all players ready unless planning defines stricter product rules.
- The default reconnect grace window is 30 seconds.
- Host transfer in lobby uses earliest joined active player after the grace window; active gameplay continues without waiting for host recovery.
- Existing identity, guest sessions, maps, locations, solo game loop, scoring, and challenge work are available as foundations for this feature.
- Realtime event payloads are intentionally compact and versioned; clients reload room/game state when they need full details or detect stale state.
