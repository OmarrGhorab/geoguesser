# Feature Specification: Daily And Shared Challenges

**Feature Branch**: `005-daily-shared-challenges`

**Created**: 2026-06-27

**Status**: Draft

**Input**: User description: "phase-05-daily-and-shared-challenges.md; make all optional stuff required; need missions system and streaks"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Play A Deterministic Daily Challenge (Priority: P1)

A guest or signed-in player can open the daily challenge, see the date-based seed, map pool, locked settings, and countdown, then play the same fixed rounds every other player receives for that day.

**Why this priority**: The daily challenge is the core promise of the phase: every player compares skill on an identical, date-bound game.

**Independent Test**: Can be fully tested by opening the daily challenge in two separate browser sessions on the same date, playing both games, and confirming the same seed, settings, round count, and locations are used while player guesses and scores remain independent.

**Acceptance Scenarios**:

1. **Given** two players open the daily challenge on the same calendar day, **When** each starts the challenge, **Then** both receive the same challenge seed, map pool, settings, and ordered round set.
2. **Given** a player is viewing the daily challenge page, **When** they inspect the challenge details before starting, **Then** they can see the seed, map pool, locked settings, daily countdown, and their current streak state.
3. **Given** a player starts a daily challenge, **When** they attempt to alter rules such as round count, timer, map pool, movement rules, or seed, **Then** the system prevents the change and keeps the daily rules unchanged.
4. **Given** the daily reset time arrives, **When** a player opens the daily challenge after reset, **Then** a new daily seed and round set are available and the previous day remains available only as historical results.

---

### User Story 2 - Share A Stable Challenge Link (Priority: P1)

A player can create or receive a shared challenge link that always resolves to the same seed, map pool, settings, and rounds so friends can compete asynchronously under identical rules.

**Why this priority**: Shared links turn solo gameplay into a social comparison loop and must be stable to preserve trust.

**Independent Test**: Can be fully tested by creating one shared challenge link, opening it in two browsers, and confirming both sessions load identical challenge metadata and rounds while maintaining separate results.

**Acceptance Scenarios**:

1. **Given** a player creates a shared challenge, **When** the link is generated, **Then** the link contains or resolves to a stable challenge identity that can be opened later by any eligible player.
2. **Given** two players open the same shared challenge link, **When** each starts the challenge, **Then** both receive the same seed, map pool, settings, and ordered round set.
3. **Given** a shared challenge has already been played by one or more players, **When** another player opens the link later, **Then** the original rules and rounds are still unchanged.
4. **Given** a player opens an invalid, expired, or unavailable shared challenge link, **When** the challenge cannot be loaded, **Then** they see a clear localized error state with a safe next action.

---

### User Story 3 - Compare Challenge Results And Leaderboards (Priority: P2)

A player can finish a daily or shared challenge and compare their result summary against other participants, including ranked scores, completion time where available, per-round scores, and personal best context.

**Why this priority**: Comparison is the reward loop for fixed-seed challenges, and daily leaderboard support is required in this phase.

**Independent Test**: Can be fully tested by completing the same challenge as multiple players and confirming the results summary and leaderboard order reflect scores, tie-breakers, and completion state consistently.

**Acceptance Scenarios**:

1. **Given** a player completes a daily challenge, **When** final results are shown, **Then** they see total score, per-round score, distance, rank, participants count, streak impact, earned mission progress, and countdown to the next daily.
2. **Given** multiple players complete the same daily challenge, **When** the leaderboard is viewed, **Then** players are ranked by deterministic challenge score rules with stable tie-breakers and without revealing unfinished-player answers early.
3. **Given** a player opens a challenge they have not completed, **When** they view comparison surfaces, **Then** hidden answer details and final leaderboard spoilers are withheld until completion or until the challenge is no longer playable.
4. **Given** a player reloads a completed challenge result, **When** the result summary is displayed again, **Then** the same score, rank context, and mission/streak impact are shown.

---

### User Story 4 - Maintain Streaks And Complete Missions (Priority: P2)

A player can build daily streaks and complete missions tied to daily and shared challenge activity, giving clear progress, rewards, and recovery rules.

**Why this priority**: Streaks and missions make daily play habitual and are explicitly required for this phase.

**Independent Test**: Can be fully tested by completing daily challenges across simulated dates and shared challenge actions, then confirming streak changes, mission progress, completion, and reset behavior.

**Acceptance Scenarios**:

1. **Given** a player completes the daily challenge before reset, **When** the result is saved, **Then** their daily streak increases or starts according to the streak rules and the new streak is visible.
2. **Given** a player misses a daily challenge day, **When** the next daily reset occurs, **Then** their active streak is broken unless an explicitly granted streak protection applies.
3. **Given** a player has active missions, **When** they complete qualifying challenge actions, **Then** mission progress updates immediately and completed missions are clearly marked with any earned reward or status.
4. **Given** a player views missions, **When** no missions are available or all missions are completed, **Then** they see an empty or completed state that explains when new missions appear.
5. **Given** a guest plays challenges, **When** streaks and missions are shown, **Then** guest progress is supported for the current device/session and the experience explains any limits compared with signed-in persistence.

---

### User Story 5 - Resume Challenge State Across Reloads (Priority: P3)

A player can reload or reopen a daily or shared challenge and return to the correct pending, active, completed, or unavailable state without changing rules or losing progress.

**Why this priority**: Reloadability prevents accidental loss and preserves confidence, but it builds on the deterministic challenge and result systems.

**Independent Test**: Can be fully tested by starting a challenge, completing at least one round, reloading the page, and confirming the same challenge state, locked rules, current round, mission progress, and streak context are restored.

**Acceptance Scenarios**:

1. **Given** a player has started but not completed a challenge, **When** they reload the challenge page, **Then** the same challenge identity, locked settings, completed rounds, and current playable state are restored.
2. **Given** a player has completed a challenge, **When** they reopen it from history or a shared link, **Then** the completed result summary is shown rather than starting a new attempt.
3. **Given** a challenge is unavailable because of reset, link problems, or access rules, **When** the page loads, **Then** the user sees a localized unavailable state rather than a broken or partially playable game.

### Edge Cases

- The daily reset happens while a player is on the daily challenge page or actively playing.
- Two browsers or devices for the same player attempt the same daily challenge at the same time.
- Two players open the same shared challenge link before, during, and after completion.
- A player attempts to replay the same daily challenge for leaderboard credit after completing it once.
- A player starts as a guest and later signs in; guest progress, missions, and streak continuity need clear merge or separation behavior.
- The challenge seed references a map pool that later changes, becomes archived, or loses locations.
- The challenge cannot provide enough unique active locations for its locked settings.
- A leaderboard has no completed players, one completed player, tied players, or many completed players.
- Mission progress is updated by multiple qualifying actions in quick succession.
- Countdown reaches zero while cached challenge data is visible.
- English and Arabic experiences need localized challenge, leaderboard, mission, streak, countdown, empty, error, disabled, and success states, including RTL layout for Arabic.
- Primary challenge loading, starting, and result comparison must feel responsive; delays longer than 2 seconds need visible loading or recovery states.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST generate a deterministic challenge seed for each daily challenge based on the challenge date and configured reset boundary.
- **FR-002**: System MUST ensure all players receive the same ordered locations, map pool, round count, timer, movement rules, scoring rules, and other settings for the same daily challenge.
- **FR-003**: System MUST provide a daily challenge page that displays the seed, map pool, locked settings, countdown to next reset, current player attempt state, streak state, mission progress entry points, and result availability.
- **FR-004**: System MUST prevent players or clients from changing any rules inside a daily or shared challenge after the challenge identity is created.
- **FR-005**: System MUST create shared challenge links that remain stable and always resolve to the same challenge seed, map pool, settings, and ordered rounds.
- **FR-006**: System MUST allow eligible guests and signed-in users to open shared challenge links and play separate attempts under the shared rules.
- **FR-007**: System MUST show a challenge result summary after completion, including total score, per-round score, per-round distance, completion state, rank context when available, and the underlying locked challenge details.
- **FR-008**: System MUST provide a daily leaderboard for accounts and a comparison view for shared challenges once results are eligible to be shown.
- **FR-009**: System MUST protect unfinished players from answer spoilers by hiding final answer details and leaderboard information that would reveal answers before the player completes the challenge or the challenge is no longer playable.
- **FR-010**: System MUST enforce one leaderboard-credit attempt per player per daily challenge while preserving completed result history.
- **FR-011**: System MUST support daily streaks that start, increment, break, and display according to daily challenge completion and reset rules.
- **FR-012**: System MUST provide clear streak recovery or protection behavior, including whether protection is unavailable, available, consumed, or expired.
- **FR-013**: System MUST provide a missions system with active missions, mission progress, completion state, and reward or status messaging tied to challenge activity.
- **FR-014**: System MUST include missions for daily challenge completion, shared challenge participation, score thresholds, leaderboard milestones, streak milestones, and round accuracy achievements.
- **FR-015**: System MUST update mission and streak progress after qualifying actions without requiring the player to manually refresh.
- **FR-016**: System MUST preserve challenge attempt state across page reloads, browser restarts where supported, and repeated visits to the same daily or shared challenge.
- **FR-017**: System MUST handle empty, loading, unavailable, invalid link, already completed, not completed, reset, and error states for daily challenges, shared challenges, leaderboards, missions, and streaks.
- **FR-018**: System MUST make all visible text available in supported locale message catalogs for English and Arabic, with Arabic supporting RTL layout.
- **FR-019**: Interactive controls MUST expose accessible names, keyboard focus behavior, disabled states, and non-color-only status indicators.
- **FR-020**: System MUST record enough historical challenge, result, streak, mission, and leaderboard facts to make completed results stable after map pools or mission rotations change.
- **FR-021**: System MUST define safe behavior for guests, including device/session-scoped progress, visible limits, and sign-in prompts that do not block basic challenge play.
- **FR-022**: System MUST provide fair tie-breakers for leaderboards and comparison views, including score, completion time or equivalent non-spoiling performance measure, and deterministic final ordering.
- **FR-023**: System MUST expose challenge history or completed challenge access so players can revisit prior daily and shared challenge results.
- **FR-024**: System MUST make any API or data contract changes traceable in planning artifacts and user-facing contracts.

### Key Entities *(include if feature involves data)*

- **Challenge**: A fixed-seed playable event with identity, type, seed, map pool, locked settings, availability window, and status.
- **Daily Challenge**: A challenge generated for a specific challenge date and reset boundary, shared by all players for that day.
- **Shared Challenge**: A player-created or system-created challenge accessed through a stable link and fixed rules.
- **Challenge Attempt**: A player's single playthrough of a challenge, including attempt state, round progress, score, completion time, and eligibility for comparison.
- **Challenge Result**: The completed summary of an attempt, including total score, per-round outcomes, rank context, and durable display facts.
- **Leaderboard Entry**: A ranked result for a challenge, with player identity display, score, tie-breaker data, and visibility rules.
- **Streak**: A player's daily completion continuity state, including current count, best count, last qualifying day, break state, and protection state.
- **Mission**: A goal with eligibility rules, progress target, current progress, completion state, reward or status messaging, and active window.
- **Mission Progress Event**: A qualifying player action that updates mission progress, such as completing a daily, sharing a challenge, reaching a score threshold, or maintaining a streak.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Two independent players opening the same daily challenge on the same date receive identical challenge metadata and ordered rounds in 100% of validation attempts.
- **SC-002**: Two independent players opening the same shared challenge link receive identical challenge metadata and ordered rounds in 100% of validation attempts.
- **SC-003**: At least 95% of players can open and start a daily or shared challenge within 2 seconds under normal operating conditions.
- **SC-004**: Completed challenge result summaries remain stable across reloads and repeated visits in 100% of validation attempts.
- **SC-005**: Leaderboard ordering is deterministic and repeatable for tied and untied results in 100% of validation attempts.
- **SC-006**: Daily streak state updates correctly across completion, missed day, and reset scenarios in 100% of date-bound validation cases.
- **SC-007**: Mission progress updates within 5 seconds of a qualifying challenge action in at least 95% of normal user sessions.
- **SC-008**: Users can understand the next available daily challenge time from the countdown without additional help in usability validation.
- **SC-009**: English and Arabic challenge, leaderboard, mission, streak, countdown, empty, error, disabled, and success states pass localization and RTL review.
- **SC-010**: All required validation gates for the feature pass before release, or any omitted gate has a documented blocker and residual risk in planning artifacts.

## Assumptions

- Daily challenge reset uses one canonical global reset boundary rather than each user's local midnight unless product later defines regional daily challenges.
- Guest players can participate in daily and shared challenges, but long-term cross-device streaks, missions, and leaderboard identity are strongest for signed-in accounts.
- Daily leaderboard entries require signed-in accounts for persistent public identity; guests can still see personal results and may receive session-scoped comparisons where safe.
- Shared challenge links are intended to be playable asynchronously and must not depend on the creator being online.
- Challenge settings include at least map pool, round count, timer, movement rules, scoring version, seed, and availability window.
- Streak protection is required as a visible state, but earning and consuming protection can be defined during planning as long as missed-day behavior is unambiguous.
- Missions are challenge-focused for this phase and do not need a full economy, shop, or complex reward redemption system.
- Existing solo game loop behavior is available as the underlying play experience, but this specification defines product behavior rather than implementation details.
