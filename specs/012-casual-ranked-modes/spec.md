# Feature Specification: Casual And Ranked Team Modes

**Feature Branch**: `[012-casual-ranked-modes]`

**Created**: 2026-07-18

**Status**: Draft

**Input**: User description: "Create casual and competitive game modes for 1v1, 2v2, and 4v4, including team chat, image sharing, teammate markers, spectating rules, ranked points, timed rounds, speed bonuses, divisions, and an exclusive top-500 eighth rank."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Play A Casual Match (Priority: P1)

A signed-in player selects Solo, Duo, or Squad casual play and enters an evenly sized 1v1, 2v2, or 4v4 match. Casual rounds have no visible gameplay countdown and never change competitive rank or rating.

**Why this priority**: Casual play provides the lowest-pressure way to use every requested team size and establishes the shared multiplayer loop before competitive progression is added.

**Independent Test**: Complete one casual match in each team size and verify that rounds advance only after all active players submit, results use the normal geography score, and no rank or rating changes.

**Acceptance Scenarios**:

1. **Given** a signed-in player or complete party, **When** the party leader queues for casual Solo, Duo, or Squad, **Then** the system searches only for the corresponding 1v1, 2v2, or 4v4 format.
2. **Given** a casual round in progress, **When** players are choosing guesses, **Then** no gameplay countdown or speed bonus is shown or applied.
3. **Given** every active player has submitted a locked guess, **When** the final submission is accepted, **Then** the round reveals the answer, shows player and team results, and advances together for all participants.
4. **Given** a casual match completes, **When** results are finalized, **Then** the result is recorded as casual and no participant's competitive rating, division, or rank changes.

---

### User Story 2 - Compete In A Ranked Match (Priority: P1)

An eligible signed-in player or complete party enters the ranked version of Solo, Duo, or Squad. Each round has a 60-second server-controlled timer, accurate early guesses can earn a small speed bonus, and the completed match changes each participant's competitive rating.

**Why this priority**: Timed fair competition and visible progression are the defining value of ranked play.

**Independent Test**: Complete equal-rating ranked matches in all three formats and verify authoritative timers, speed bonuses, team outcomes, rating gains or losses, and durable match history.

**Acceptance Scenarios**:

1. **Given** an eligible player or complete party, **When** ranked matchmaking is joined, **Then** the system searches for an equal-size opposing team within the allowed competitive rating range.
2. **Given** a ranked round begins, **When** the shared start time is reached, **Then** every participant receives the same 60-second deadline.
3. **Given** a player submits before the deadline, **When** the guess is scored, **Then** the player receives the normal accuracy score plus a speed bonus equal to 5% of that accuracy score multiplied by the fraction of round time remaining, rounded down and capped at 250 points.
4. **Given** a player does not submit before the deadline, **When** the round closes, **Then** that player receives zero points for the round and the match continues.
5. **Given** a ranked match reaches a win, loss, or draw, **When** it is finalized, **Then** every participant sees the previous rating, rating change, new rating, division/rank movement if any, and the inputs that affected the change.
6. **Given** a completed or cancelled match is processed again, **When** progression finalization is retried, **Then** rating changes are applied no more than once.

---

### User Story 3 - Collaborate With A Team (Priority: P1)

Players in Duo and Squad matches collaborate through private team text, safe image attachments, and color-coded proposed map markers. Each teammate makes and locks an individual guess, and the team score is the sum of all teammate scores.

**Why this priority**: Communication and combined guesses are what make 2v2 and 4v4 meaningfully different from several parallel solo matches.

**Independent Test**: Run a two-team match, exchange text and images, move each teammate's proposed marker, submit different guesses, and verify that only teammates receive the collaboration data and that the team total equals the sum of its members' points.

**Acceptance Scenarios**:

1. **Given** a Duo or Squad match, **When** a teammate sends text or an approved image, **Then** only current members of that teammate's team can receive it.
2. **Given** teammates are choosing guesses, **When** any teammate places or moves a proposed marker, **Then** teammates see one color- and name-identified marker for that player without changing anyone else's marker.
3. **Given** a player locks a guess, **When** the guess is accepted, **Then** that player's guess cannot be moved or resubmitted, while unlocked teammates may continue collaborating.
4. **Given** all player results for a round are known, **When** team results are calculated, **Then** the team total equals the sum of its members' accuracy and applicable speed-bonus points; missing guesses contribute zero.
5. **Given** an opponent or non-participant attempts to read team messages, images, live markers, or unrevealed guesses, **When** access is checked, **Then** access is denied without confirming private content.

---

### User Story 4 - Spectate After Submitting (Priority: P2)

A player who has locked a guess remains engaged by spectating an allowed live view until the round closes. A Solo player may spectate the opponent; Duo and Squad players may spectate only their own teammates.

**Why this priority**: Spectating avoids dead time while preserving the different information and collaboration rules of solo and team play.

**Independent Test**: Submit early in each format and verify the available spectating targets, hidden marker/coordinate rules, chat permissions, and automatic return to results when the round closes.

**Acceptance Scenarios**:

1. **Given** a Solo player has locked a guess, **When** the opponent is still playing, **Then** the player may watch the opponent's panorama or image navigation but cannot see the opponent's map marker, coordinates, or score before reveal.
2. **Given** a Duo or Squad player has locked a guess, **When** teammates are still playing, **Then** the player may switch among active teammates and continue using team chat and proposed markers.
3. **Given** a team player attempts to spectate an opponent, **When** the request is evaluated, **Then** access is denied.
4. **Given** the watched player disconnects, submits, or the round closes, **When** the spectating state changes, **Then** the spectator is moved to another allowed active teammate or the shared waiting/results state.

---

### User Story 5 - Progress Through Eight Ranks (Priority: P2)

Ranked players progress through a geography-themed ladder. The first seven ranks each contain Divisions III, II, and I; the eighth rank is awarded only to the 500 highest eligible players in the active season.

**Why this priority**: A clear, exclusive ladder gives competitive matches a long-term goal and makes the top leaderboard meaningful.

**Independent Test**: Place test players around every promotion, demotion, and top-500 boundary and verify names, divisions, rating changes, qualification, tie-breaking, and removal from the exclusive rank.

**Acceptance Scenarios**:

1. **Given** a ranked player below the top-500 tier, **When** rating crosses a division boundary, **Then** the player is promoted or demoted into the corresponding division and sees the change after the match.
2. **Given** the active-season leaderboard is recalculated, **When** an eligible player is ranked from 1 through 500, **Then** that player receives World Legend and their leaderboard position; no player below position 500 receives that rank.
3. **Given** a World Legend player falls below position 500 or becomes ineligible, **When** standings update, **Then** World Legend is removed and the player's rating maps back to the appropriate lower rank.
4. **Given** multiple players have the same rating at the cutoff, **When** order is determined, **Then** the player with more ranked wins is first, followed by the player who reached the tied rating earlier.
5. **Given** a season closes, **When** final standings are published, **Then** the final top 500 and each player's peak and ending rank remain viewable while a new active-season ladder begins under the documented reset policy.

### Rank Ladder

| Order | Rank | Divisions | Meaning |
| --- | --- | --- | --- |
| 1 | Scout | III, II, I | Learning the world and core clues |
| 2 | Pathfinder | III, II, I | Building consistent routes to an answer |
| 3 | Trailblazer | III, II, I | Reading terrain and regional patterns |
| 4 | Navigator | III, II, I | Reliably narrowing countries and regions |
| 5 | Cartographer | III, II, I | Strong map knowledge and precision |
| 6 | Explorer | III, II, I | Advanced global consistency |
| 7 | Geo Master | III, II, I | Elite players immediately below the leaderboard tier |
| 8 | World Legend | Top 500 only | Exclusive active-season leaderboard rank with no divisions |

### Edge Cases

- A party member disconnects, leaves, becomes ineligible, or changes account state while the party is queued or a match is forming.
- Simultaneous queue requests, leave requests, or match formation attempts occur from multiple tabs or devices.
- A Duo or Squad party is incomplete, exceeds the team size, or contains members outside the permitted ranked rating spread.
- One team is formed successfully but creation of the opposing team or game destination fails.
- A casual player refuses to submit indefinitely; no gameplay timer is introduced, but a 10-minute no-interaction safeguard ends the abandoned match without competitive penalty.
- A participant disconnects during play; a 90-second reconnect grace period applies before that participant forfeits, and a ranked forfeit applies the documented abandon penalty.
- A ranked guess arrives at the exact server deadline or is retried after a successful submission whose response was lost.
- A teammate submits while a proposed-marker update, message, or image upload is in flight.
- A chat image has a misleading extension, unsupported type, excessive size, unsafe content, or fails processing.
- A player mutes or reports a teammate, or a blocked player is placed in the same party or team.
- All players submit early, only some teammates submit, or every player times out.
- A match is tied after all rounds, including an exact team-point tie.
- Rating finalization races with cancellation, abandonment, season closure, or a retry.
- More than 500 players tie at the leaderboard cutoff.
- Loading, empty, searching, ready, active, submitted, spectating, reconnecting, timed-out, abandoned, result, promoted, demoted, and unavailable states must be distinct.
- English and Arabic experiences must convey identical rules and status, preserve usable RTL layouts, and keep player markers distinguishable without depending on color alone.
- Match state, chat, marker, and spectating updates taking longer than 1 second under normal conditions are degraded and must recover without duplicate actions.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST offer Casual and Ranked playlists in Solo (1v1), Duo (2v2), and Squad (4v4) formats.
- **FR-002**: Matchmaking MUST form two equal-size teams for the selected format and MUST NOT mix Casual and Ranked queues.
- **FR-003**: Duo and Squad matchmaking MUST require a complete premade party in v1; the party leader controls queue join and leave, while every member sees party and queue state.
- **FR-004**: A player MUST belong to at most one active party, queue entry, or match assignment at a time, including during retries and concurrent requests.
- **FR-005**: Casual rounds MUST have no visible gameplay countdown, no speed bonus, and no effect on competitive progression.
- **FR-006**: Casual rounds MUST close when all active participants submit, when a team forfeits, or when the no-interaction safeguard ends an abandoned match.
- **FR-007**: Ranked rounds MUST use one authoritative 60-second start and deadline shared by all participants.
- **FR-008**: Ranked rounds MUST close when all eligible participants submit or the deadline expires, whichever happens first.
- **FR-009**: Every player MUST be allowed one locked guess per round; duplicate retries MUST return the original outcome and conflicting submissions MUST NOT replace it.
- **FR-010**: The normal accuracy score MUST remain between 0 and 5,000 points per player per round.
- **FR-011**: Ranked guesses submitted before the deadline MUST receive `floor(accuracy score × 0.05 × remaining-time fraction)`, capped at 250; a zero accuracy score or timed-out guess receives no speed bonus.
- **FR-012**: A team's round and match totals MUST equal the sum of its members' accepted accuracy scores and applicable speed bonuses, with missing guesses worth zero.
- **FR-013**: Match outcome MUST be determined by total match points; equal totals produce a draw with no hidden tiebreaker.
- **FR-014**: Hidden answer coordinates MUST remain unavailable until round reveal, and opponents' unrevealed guesses, messages, images, and markers MUST remain private.
- **FR-015**: After locking a Solo guess, a player MUST be able to spectate only the opponent's navigation view, without the opponent's marker, coordinates, or unrevealed score.
- **FR-016**: After locking a Duo or Squad guess, a player MUST be able to spectate only active teammates and continue team collaboration until the round closes.
- **FR-017**: Duo and Squad teammates MUST have a private match chat supporting localized text and one image attachment per message.
- **FR-018**: Chat text MUST be limited to 500 characters; images MUST be JPEG, PNG, or WebP, no larger than 5 MB, and validated as safe before teammates can view them.
- **FR-019**: Players MUST be able to mute a teammate and report a message or image; message sending and image uploads MUST be rate-limited, and blocked users MUST NOT be placed in the same premade party.
- **FR-020**: Each teammate MUST have one live proposed map marker that is name- and color-identified, independently movable until that player's guess locks, and visible only to teammates.
- **FR-021**: Ranked matchmaking MUST reject incomplete parties and parties whose highest and lowest members are more than two named ranks apart.
- **FR-022**: Ranked rating changes MUST be based on match outcome and opposing-team strength, apply equally to members of the same team before abandon penalties, and be applied exactly once.
- **FR-023**: Speed bonuses MUST affect match points only and MUST NOT directly change the rating formula beyond their effect on win, loss, or draw.
- **FR-024**: Ranked abandonment after match formation MUST count as a loss for the abandoning player, apply an additional visible abandon penalty, and MUST NOT reward the opposing team more than a normal win.
- **FR-025**: Each new competitive participant MUST complete five placement matches before receiving a visible division; placement players still match and gain hidden progress under the same fair-play rules.
- **FR-026**: The visible ladder MUST use Scout, Pathfinder, Trailblazer, Navigator, Cartographer, Explorer, and Geo Master with Divisions III, II, and I in ascending order.
- **FR-027**: Each standard division MUST span 100 visible Rank Rating points, promote at its upper boundary, and demote when post-match rating falls below its lower boundary; a floor of zero MUST prevent negative visible rating.
- **FR-028**: World Legend MUST be the eighth rank, have no divisions, and be awarded only to positions 1–500 on the global active-season ranked leaderboard.
- **FR-029**: World Legend eligibility MUST require completion of placements, at least 25 completed ranked matches in the active season, good account standing, and a rating at or above Geo Master I.
- **FR-030**: Top-500 order MUST use rating descending, then ranked wins descending, then the earliest time the tied rating was reached; standings and World Legend membership MUST refresh within five minutes of an eligible result.
- **FR-031**: Competitive progression MUST be seasonal; closed seasons MUST preserve final top-500 standings plus each participant's ending and peak rank, and the new-season reset rule MUST be disclosed before a player queues.
- **FR-032**: Players MUST be able to view current rank, division, rating progress, placement progress, last-match change, seasonal wins and matches, peak rank, and top-500 position when applicable.
- **FR-033**: Match results MUST show individual accuracy, speed bonus, team totals, outcome, rating change for Ranked, and no-rating-change confirmation for Casual.
- **FR-034**: Queue, match, chat, marker, spectating, result, and progression changes MUST recover correctly after refresh or reconnection without leaking unauthorized state.
- **FR-035**: User-facing copy MUST be available in English and Arabic with equivalent meaning and usable right-to-left layout.
- **FR-036**: Interactive controls and live status changes MUST have accessible names, keyboard behavior, visible focus, non-color-only identification, and appropriate assistive announcements.
- **FR-037**: Operational records MUST support dispute, abuse, and rating diagnosis without logging hidden locations, raw chat/image content, authentication data, or precise private guesses.
- **FR-038**: Public contracts, retention rules, season policy, moderation behavior, and durable data changes MUST be documented in planning artifacts before implementation is complete.

### Key Entities

- **Playlist**: The selected Casual or Ranked ruleset combined with Solo, Duo, or Squad team size.
- **Party**: A temporary group of one, two, or four signed-in players with one leader, readiness, invitations, and a single shared queue state.
- **Match**: The authoritative two-team contest with playlist, lifecycle, round sequence, result, and ranked-finalization state.
- **Team Membership**: A player's match team, permissions, individual total, disconnect/forfeit state, and relationship to team totals.
- **Round Guess**: One locked player location with submission time, accuracy score, optional ranked speed bonus, and timeout state.
- **Proposed Marker**: A teammate-visible, round-scoped suggestion owned by one player and separate from the player's locked guess.
- **Team Message**: A team-scoped text message with optional approved image, author, moderation state, and match-relative ordering.
- **Competitive Profile**: A player's placement state, visible rating, rank, division, seasonal wins/matches, peak rank, and rating history.
- **Competitive Season**: The active or closed progression window with reset policy and frozen final standings.
- **Top-500 Standing**: An eligible player's active-season leaderboard position and tie-break facts; positions 1–500 grant World Legend.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Two complete compatible sides can enter the same Casual or Ranked game in the requested 1v1, 2v2, or 4v4 format, with no cross-format or cross-playlist assignments in concurrency tests.
- **SC-002**: 95% of party, queue, live match, marker, chat, and spectating state changes are visible to authorized participants within 1 second under normal operating conditions.
- **SC-003**: 100% of Casual matches complete without changing any participant's competitive rating and display no gameplay countdown.
- **SC-004**: 100% of Ranked participants share the same authoritative round deadline, and late or duplicate guesses never alter an accepted result.
- **SC-005**: In all scoring tests, individual speed bonuses match the disclosed formula, never exceed 250 points, and team totals exactly equal member totals.
- **SC-006**: In all authorization tests, opponents and non-participants cannot access private team chat, images, proposed markers, unrevealed guesses, or disallowed spectating views.
- **SC-007**: In all retry and concurrency tests, each completed Ranked match changes each eligible participant's rating no more than once.
- **SC-008**: Rank displays map correctly across all 21 standard divisions, and exactly 500 eligible players receive World Legend when at least 500 qualify.
- **SC-009**: Eligible-result updates appear in the active-season leaderboard and World Legend membership within five minutes.
- **SC-010**: At least 95% of test users can identify playlist, team size, timer rule, current score, result, and ranked change on their first attempt without external instructions.
- **SC-011**: English and Arabic journeys pass equivalent-content, RTL, keyboard-only, focus, screen-reader announcement, and non-color-only marker checks.
- **SC-012**: A refreshed or reconnected participant returns to the correct party, queue, round, submitted/spectating, chat, or results state within three seconds under normal conditions.
- **SC-013**: All text/image abuse controls reject invalid content, enforce mute/report outcomes, and prevent unauthorized retrieval in automated and manual safety checks.
- **SC-014**: All constitution-required backend, frontend, contract, persistence, concurrency, localization, accessibility, and browser-flow validation gates pass before release.

## Assumptions

- Both public playlists require signed-in, active accounts; guest play remains available only in existing non-matchmade experiences.
- Solo parties contain one player, and v1 Duo/Squad queues require complete premade parties rather than filling empty slots with strangers.
- A match uses five rounds and the existing geography accuracy curve; Ranked adds only the disclosed speed bonus.
- There is one global visible competitive rating across Ranked Solo, Duo, and Squad, and one global top-500 leaderboard. Playlist-specific hidden matchmaking tuning may be designed later without creating separate visible ranks.
- Ranked rating calculation and season reset amounts will be finalized and published during planning; outcome, opponent strength, and abandonment are the only permitted v1 rating inputs.
- Casual's 10-minute no-interaction safeguard and the 90-second reconnect grace are abandonment protections, not visible round timers or speed-scoring inputs.
- Match chat exists only during Duo/Squad matches and is not a general direct-message system. Normal participants cannot reopen chat after the result screen; safety-authorized retention follows the project's moderation and privacy policy.
- Images are uploaded files, not arbitrary remote URLs; animated images, SVG, audio, video, voice chat, and cross-team chat are outside v1.
- Solo opponent spectating shares only the opponent's navigable scene state; it never shares their map, cursor, marker, coordinates, or private client surface.
- Parties, matches, ratings, chat safety, and leaderboard data build on the existing authenticated sessions, friends/block rules, uploads, realtime rooms, multiplayer scoring, matchmaking, and leaderboard foundations.
