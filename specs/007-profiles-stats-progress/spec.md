# Feature Specification: Profiles Stats Progress

**Feature Branch**: `[007-profiles-stats-progress]`

**Created**: 2026-07-01

**Status**: Draft

**Input**: User description: "Phase 7 - Profiles Stats And Persistent Progress: turn registered play into persistent account progress and public-safe stats."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Manage Registered Profile (Priority: P1)

A registered player can open their account profile, review the public identity fields associated with their account, and update editable profile details such as display name, avatar reference, country, locale, timezone, and preferences. The player receives clear feedback when an update is saved or rejected.

**Why this priority**: Persistent identity is the foundation for account progress, public stats, future social features, and localized account experiences.

**Independent Test**: Can be fully tested by signing in as a registered player, loading the profile, submitting a valid profile update, and confirming the updated fields are returned on the next profile load while private account fields remain hidden.

**Acceptance Scenarios**:

1. **Given** a registered player with an existing profile, **When** they view their profile, **Then** they see their editable profile fields and a current stats summary without seeing private authentication or session details.
2. **Given** a registered player submits valid profile changes, **When** the update is accepted, **Then** the saved profile reflects only the intended changes and preserves unchanged fields.
3. **Given** a registered player submits invalid profile changes, **When** validation fails, **Then** the existing profile remains unchanged and the player receives a clear localized error.
4. **Given** a guest session, **When** it tries to access registered-only profile management, **Then** access is denied without returning registered profile data.

---

### User Story 2 - View Public-Safe Player Stats (Priority: P1)

Any player can view another registered player's public gameplay stats without gaining access to private account details, hidden coordinates, session data, or other sensitive information.

**Why this priority**: Public-safe stats make registered progress visible and prepare the product for leaderboards, profiles, and social comparisons without compromising privacy.

**Independent Test**: Can be fully tested by completing games as a registered player, requesting that player's public stats from a separate session, and confirming the response contains aggregate gameplay outcomes only.

**Acceptance Scenarios**:

1. **Given** a registered player has completed eligible games, **When** another player views their public stats, **Then** the viewer sees aggregate totals such as games played, score summary, and best performance without private account data.
2. **Given** a registered player has no completed eligible games, **When** their public stats are viewed, **Then** the response shows a valid empty or zero-state summary rather than an error.
3. **Given** a requested user does not exist or is unavailable, **When** public stats are requested, **Then** the viewer receives a not-found style outcome without sensitive account information.

---

### User Story 3 - Preserve Account Progress Across Sessions (Priority: P2)

A registered player can return after playing games and see account progress reflected consistently, including completed game contributions to stats and saved progress foundations for future history and resume surfaces.

**Why this priority**: Persistent progress gives registered play lasting value and provides the data foundation for history, achievements, and future competitive features.

**Independent Test**: Can be fully tested by completing and reloading gameplay as a registered player, then confirming account stats and saved progress summaries remain stable across sign-out, sign-in, and page reloads.

**Acceptance Scenarios**:

1. **Given** a registered player completes a game, **When** their account progress is loaded later, **Then** the completed game contributes once to their aggregate stats.
2. **Given** a registered player has eligible in-progress or completed games, **When** saved progress is queried for account surfaces, **Then** the returned summaries are ordered predictably and do not expose hidden round answers.
3. **Given** the same game result is processed more than once, **When** account progress is recalculated or refreshed, **Then** stats remain stable and are not double-counted.

---

### User Story 4 - Handle Profile Privacy And Boundary States (Priority: P3)

Players receive consistent outcomes for profile and stats boundary conditions, including invalid profile fields, unsupported locales, excessive update attempts, empty stats, disabled accounts, and localized error states.

**Why this priority**: Boundary handling protects account data, avoids confusing profile experiences, and keeps public stats safe as the registered-user feature set grows.

**Independent Test**: Can be fully tested by exercising profile validation failures, guest access attempts, excessive update attempts, empty public stats, and English and Arabic localized states.

**Acceptance Scenarios**:

1. **Given** profile data includes unsupported or malformed values, **When** a player submits an update, **Then** the update is rejected with field-specific feedback and no partial invalid data is saved.
2. **Given** a player makes profile updates too frequently, **When** the update limit is reached, **Then** later updates are temporarily rejected with a clear recovery message.
3. **Given** the interface is shown in English or Arabic, **When** profile or stats states are loaded, empty, saved, invalid, disabled, or unavailable, **Then** all visible copy is localized and Arabic layout follows right-to-left expectations.

---

### Edge Cases

- A guest session attempts to read or update the registered-only current profile.
- A registered account exists but its profile row or equivalent profile record is missing.
- A display name is too short, too long, blank after trimming, or contains unsupported control characters.
- A locale, country, timezone, avatar reference, or preference value is unsupported or malformed.
- A player submits a profile update that omits optional fields or intentionally clears optional fields.
- Two profile updates are submitted close together from different sessions.
- A public stats request targets a missing, disabled, or private/unavailable user.
- A player has no completed eligible games yet.
- A player has active, abandoned, cancelled, or room games that should not reveal hidden answers in progress summaries.
- Stats refresh after game completion is delayed or repeated.
- User-facing profile and stats states must cover loading, empty, error, disabled, validation, rate-limited, and success outcomes.
- English and Arabic profile and stats experiences must provide equivalent meaning and usable right-to-left layout.
- Profile load, stats load, and profile update latency beyond a couple of seconds should be treated as a degraded experience.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow registered players to retrieve their current profile and account progress summary.
- **FR-002**: The system MUST prevent guest sessions from reading or updating registered-only profile management data.
- **FR-003**: Registered players MUST be able to update editable profile fields while preserving fields that were not part of the update.
- **FR-004**: The system MUST validate display names as trimmed, user-visible names between 2 and 32 characters.
- **FR-005**: The system MUST validate locale values against the product's supported locales and reject unsupported locale values.
- **FR-006**: The system MUST validate optional country, timezone, avatar, and preference values before saving them.
- **FR-007**: The system MUST provide clear success, validation error, authorization error, unavailable, empty, disabled, and rate-limited outcomes for profile and stats interactions.
- **FR-008**: The system MUST limit excessive profile update attempts without corrupting or partially applying profile changes.
- **FR-009**: The system MUST expose public stats for registered users using aggregate gameplay outcomes only.
- **FR-010**: Public stats MUST exclude email addresses, authentication details, session identifiers, private preferences, hidden locations, guess coordinates, and precise private account data.
- **FR-011**: Public stats MUST return a valid zero-state summary for registered users with no completed eligible games.
- **FR-012**: Stats MUST be based on completed eligible games and MUST avoid double-counting repeated result processing.
- **FR-013**: Saved progress summaries for registered accounts MUST identify eligible in-progress and completed gameplay without exposing hidden round answers or private location details.
- **FR-014**: Saved progress summaries MUST be ordered predictably and support bounded result sets so account history surfaces remain usable as play volume grows.
- **FR-015**: Profile and stats behavior MUST treat missing, disabled, or unavailable users consistently without exposing private account state.
- **FR-016**: User-facing copy MUST be available in supported locale message catalogs unless a behavior has no visible text.
- **FR-017**: Interactive controls MUST expose accessible names, keyboard focus behavior, disabled states, and non-color-only status indicators.
- **FR-018**: External data contract changes MUST be reflected in this feature's planning artifacts before implementation is considered complete.

### Key Entities *(include if feature involves data)*

- **Registered Profile**: The public and preference-bearing identity record for a registered player, including display name, avatar reference, country, locale, timezone, and editable preferences.
- **Public Stats Summary**: A privacy-safe aggregate of a registered player's completed eligible gameplay, such as games played and score summary values.
- **Saved Progress Summary**: A registered player's account-linked gameplay summary used for future history and resume experiences, covering eligible in-progress and completed play without answer spoilers.
- **Game Participation**: The relationship between a registered player and a gameplay session that determines whether a completed or in-progress game contributes to account progress.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 95% of registered players can load their current profile and progress summary in under 2 seconds under normal operating conditions.
- **SC-002**: 95% of valid profile updates are confirmed and visible on a subsequent profile load within 1 second under normal operating conditions.
- **SC-003**: 100% of guest attempts to access registered-only profile management are denied without returning registered profile fields.
- **SC-004**: 100% of public stats responses exclude private account fields, authentication data, session identifiers, hidden coordinates, and raw guess coordinates.
- **SC-005**: Completed eligible games are reflected in account stats and saved progress no more than once, even if completion processing is retried.
- **SC-006**: Registered users with no completed eligible games receive a valid empty stats state in 100% of requests.
- **SC-007**: All profile and stats user-facing states have English and Arabic copy with equivalent meaning before release.
- **SC-008**: All profile update controls and profile/stat status messages are usable with keyboard navigation and expose accessible names or status semantics before release.

## Assumptions

- This phase focuses on registered accounts; linking past guest play into a newly registered account is out of scope unless a later feature explicitly adds account linking.
- Supported locales for this phase are English and Arabic.
- Public stats are derived from completed eligible games; active, cancelled, abandoned, or unfinished games do not expose answer spoilers through public stats.
- Profile email, password, connected login providers, and account deletion are owned by existing or future account-management features, not this profile progress phase.
- Avatar management accepts only safe image references that the product can validate; full image upload and moderation workflows remain separate.
- Detailed public profile privacy controls are out of scope for this phase; the default public surface is limited to safe identity fields and aggregate stats.
