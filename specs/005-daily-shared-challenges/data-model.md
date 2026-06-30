# Data Model: Daily And Shared Challenges

## Overview

The feature adds a challenge layer above the existing solo game loop. Challenges define immutable fixed-seed rules and selected locations. Attempts connect a player to a challenge and, once started, to a concrete solo game. Leaderboards, streaks, missions, and mission progress are derived from durable challenge attempt/result events.

## Entities

### Challenge

Represents a fixed-seed playable event.

**Fields**
- `id`: stable challenge identifier.
- `type`: `daily` or `shared`.
- `slug_or_code`: public stable link/code for shared challenges and optional readable daily identifier.
- `seed`: deterministic seed used for selection/audit.
- `challenge_date`: date for daily challenges; empty for shared challenges unless explicitly scheduled.
- `reset_starts_at`, `reset_ends_at`: availability window for daily challenges and optional expiry for shared challenges.
- `map_id`: source map pool.
- `settings_snapshot`: locked settings such as round count, timer, movement rules, scoring version, and visibility.
- `status`: `draft`, `active`, `completed`, `archived`, or `unavailable`.
- `created_by_user_id`: nullable creator for shared challenges.
- `created_at`, `updated_at`.

**Validation**
- Daily challenges require `challenge_date`, reset window, map, seed, and active status before play.
- Shared challenges require a stable link/code, map, settings snapshot, seed, and creator/session attribution where available.
- Settings snapshot is immutable after activation.
- Daily challenge uniqueness: one active daily per challenge date and ruleset.
- Shared challenge link/code uniqueness.

### Challenge Location

Stores the selected ordered location snapshot for a challenge.

**Fields**
- `challenge_id`.
- `round_number`.
- `location_id`.
- `selection_version`.
- `created_at`.

**Validation**
- Unique `(challenge_id, round_number)`.
- Unique `(challenge_id, location_id)` to prevent repeated locations within a challenge.
- Round numbers start at 1 and match the challenge round count.

### Challenge Attempt

Represents one player's playthrough of a challenge.

**Fields**
- `id`.
- `challenge_id`.
- `game_id`: linked solo game once gameplay is created.
- `user_id`: nullable for guests.
- `guest_identity_hash`: nullable for signed-in users.
- `status`: `pending`, `active`, `completed`, `abandoned`, `expired`.
- `leaderboard_eligible`: true for qualifying account attempts.
- `started_at`, `completed_at`.
- `total_score`, `total_distance_meters`, `completion_duration_ms`.
- `created_at`, `updated_at`.

**Validation**
- Exactly one of `user_id` or `guest_identity_hash` is present.
- One leaderboard-credit attempt per user per daily challenge.
- Reopening a completed challenge returns the completed attempt rather than creating a second leaderboard attempt.
- Attempt game must use the challenge's immutable location and settings snapshot.

### Challenge Result

Represents stable completed-result display data.

**Fields**
- `attempt_id`.
- `challenge_id`.
- `total_score`.
- `total_distance_meters`.
- `round_results_snapshot`: stable per-round score, distance, guess, and revealed location fields.
- `rank_snapshot`: optional rank context at completion.
- `completed_at`.

**Validation**
- Created once per completed attempt.
- Result snapshots remain stable even if map metadata changes later.
- Spoiler-sensitive fields are only exposed after completion or when challenge visibility rules allow.

### Leaderboard Entry

Represents a ranked challenge result.

**Fields**
- `challenge_id`.
- `attempt_id`.
- `user_id`.
- `display_name_snapshot`.
- `score`.
- `completion_duration_ms`.
- `completed_at`.
- `rank`.
- `created_at`.

**Validation**
- Only account-backed eligible attempts appear in public daily leaderboard entries.
- Ordering is deterministic: score descending, completion duration ascending, completed time ascending, stable attempt/user tie-breaker.
- Reads are paginated.

### Streak

Represents daily completion continuity.

**Fields**
- `owner_user_id` or `guest_identity_hash`.
- `current_count`.
- `best_count`.
- `last_completed_challenge_date`.
- `status`: `active`, `broken`, `protected`, `inactive`.
- `protection_state`: `none`, `available`, `consumed`, `expired`.
- `updated_at`.

**Validation**
- A daily challenge can increment a streak at most once.
- Completing consecutive daily challenge dates increments the streak.
- Missing a daily date breaks the streak unless protection is available and consumed.
- Guest streaks are scoped to the guest identity/session and marked as limited.

### Streak Event

Auditable record of streak mutations.

**Fields**
- `id`.
- `owner_user_id` or `guest_identity_hash`.
- `challenge_id`.
- `challenge_date`.
- `event_type`: `started`, `incremented`, `protected`, `broken`, `reset`.
- `previous_count`, `new_count`.
- `created_at`.

**Validation**
- Idempotent per owner and challenge date for completion-derived events.

### Mission

Represents an active challenge-focused goal.

**Fields**
- `id`.
- `code`.
- `title_key`, `description_key`: localization keys.
- `mission_type`: `daily_completion`, `shared_participation`, `score_threshold`, `leaderboard_milestone`, `streak_milestone`, `round_accuracy`.
- `target_value`.
- `active_starts_at`, `active_ends_at`.
- `reward_snapshot`: status or reward metadata.
- `status`: `draft`, `active`, `archived`.
- `created_at`, `updated_at`.

**Validation**
- Active missions require a target and active window.
- Mission display text is localized through message catalogs, not stored as user-facing prose only.

### Mission Progress

Represents a player's progress toward a mission.

**Fields**
- `mission_id`.
- `owner_user_id` or `guest_identity_hash`.
- `current_value`.
- `target_value`.
- `status`: `not_started`, `in_progress`, `completed`, `claimed`, `expired`.
- `completed_at`, `claimed_at`.
- `updated_at`.

**Validation**
- Progress cannot exceed target for completion calculations, though raw events may be retained.
- Completion is idempotent per mission and owner.
- Guest progress is scoped and clearly marked as limited.

### Mission Progress Event

Auditable qualifying action for mission progress.

**Fields**
- `id`.
- `mission_id`.
- `owner_user_id` or `guest_identity_hash`.
- `source_attempt_id`.
- `source_challenge_id`.
- `event_type`.
- `delta`.
- `created_at`.

**Validation**
- Unique idempotency per mission, owner, source attempt, and event type where a source attempt exists.

## Relationships

- A `Challenge` has many `ChallengeLocation` records.
- A `Challenge` has many `ChallengeAttempt` records.
- A `ChallengeAttempt` may link to one solo `Game`.
- A `ChallengeAttempt` has one completed `ChallengeResult`.
- A `ChallengeResult` may produce one `LeaderboardEntry`.
- A player/guest has one current `Streak` and many `StreakEvent` records.
- A `Mission` has many `MissionProgress` records.
- A `ChallengeAttempt` completion may produce multiple `MissionProgressEvent` records.

## State Transitions

### Challenge

```text
draft -> active -> archived
active -> unavailable
```

- Daily challenges are materialized into `active` for their challenge date.
- Shared challenges become `active` when created and remain stable until archived or unavailable.

### Challenge Attempt

```text
pending -> active -> completed
pending -> expired
active -> abandoned
active -> expired
```

- A completed attempt cannot return to active.
- Completed daily attempts cannot be duplicated for leaderboard credit.

### Streak

```text
inactive -> active
active -> active       # consecutive completion increments
active -> protected    # missed day with protection
active -> broken       # missed day without protection
protected -> active    # next qualifying completion
broken -> active       # new streak starts
```

### Mission Progress

```text
not_started -> in_progress -> completed -> claimed
not_started -> completed
in_progress -> expired
```

## Query And Index Requirements

- Challenge lookup by type/date and by shared code.
- Challenge locations by challenge and round order.
- Attempts by challenge and owner.
- Unique leaderboard-credit daily attempts by challenge and account.
- Leaderboard reads by challenge, score, duration, completed time, and stable tie-breaker.
- Streak lookup by owner.
- Mission lookup by active window/status.
- Mission progress lookup by mission and owner.
- Mission/streak event idempotency by owner, source attempt, and event type.

## Retention And Privacy

- Completed challenge and result facts are retained for history and leaderboard integrity.
- Hidden coordinates and provider metadata follow the same spoiler-safe rules as solo game results.
- Logs and metrics must not include precise hidden coordinates, raw auth tokens, or private profile details.
