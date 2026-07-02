# Data Model: Profiles Stats Progress

## Registered Profile

Represents the editable public identity and preference-bearing profile for a registered account.

### Fields

- `user_id`: Stable identifier of the registered account that owns the profile.
- `display_name`: Public display name shown on profile, stats, and future social surfaces.
- `avatar_url`: Optional safe avatar reference or mediated image URL.
- `country_code`: Optional ISO country code for display and future regional stats.
- `locale`: Supported locale preference, initially `en` or `ar`.
- `timezone`: Optional IANA timezone used for localized account experiences.
- `preferences`: Optional profile/account preferences that are safe to expose only to the owning user.
- `created_at`: Profile creation timestamp.
- `updated_at`: Last successful profile update timestamp.

### Validation Rules

- `display_name` is required, trimmed, and 2 to 32 user-visible characters.
- `display_name` must reject blank-after-trim and unsupported control characters.
- `locale` must be allowlisted to supported product locales.
- `country_code`, when present, must be a valid two-letter country code.
- `timezone`, when present, must be a valid timezone identifier.
- `avatar_url`, when present, must be a safe supported image reference; full upload/moderation is out of scope.
- Updates must preserve omitted fields and only clear optional fields when explicitly requested.

### Relationships

- Belongs to one registered user.
- Supplies public display identity for stats, history, leaderboards, and future social features.

## Public Stats Summary

Represents privacy-safe aggregate gameplay outcomes for a registered user.

### Fields

- `user_id`: Registered account being summarized.
- `display_name`: Public profile display name.
- `avatar_url`: Optional public-safe avatar reference.
- `country_code`: Optional public-safe country code.
- `games_played`: Count of completed eligible games.
- `total_score`: Sum of eligible completed game scores.
- `average_score`: Average score across eligible completed games.
- `best_score`: Best eligible completed game score.
- `last_played_at`: Optional timestamp of most recent eligible completed game.

### Validation Rules

- Missing completed games return zero-valued stats, not an error.
- Only completed eligible games count toward public stats.
- Public stats must exclude email, auth/session data, private preferences, hidden locations, guess coordinates, and precise private account data.
- Repeated completion/result processing must not double-count the same game participation.

### Relationships

- Derived from registered profile plus completed game participation.
- References gameplay facts from games and game player participation.

## Saved Progress Summary

Represents a bounded account-linked gameplay summary for future history and resume surfaces.

### Fields

- `game_id`: Stable game identifier.
- `map_id`: Map identifier for summary display.
- `mode`: Gameplay mode, such as solo, challenge, or private room.
- `status`: Public-safe game status for history/resume.
- `round_count`: Total configured rounds.
- `current_round_number`: Optional current round for resumable in-progress games.
- `total_score`: Current or final player score.
- `started_at`: Optional started timestamp.
- `completed_at`: Optional completed timestamp.
- `created_at`: Creation timestamp used for ordering.

### Validation Rules

- Summaries must be ordered by stable descending history keys.
- Result sets must be bounded and cursor-paginated.
- In-progress summaries must not reveal hidden round answers, location IDs, answer coordinates, or raw guess coordinates.
- Completed summaries can link to existing authorized results flows rather than duplicating sensitive result detail.

### Relationships

- Derived from game participation by a registered user.
- Complements public stats and future saved history surfaces.

## Game Participation

Represents a registered player's relationship to a gameplay session.

### Fields

- `game_player_id`: Stable participation identifier.
- `user_id`: Registered user identifier when participation belongs to an account.
- `game_id`: Game identifier.
- `status`: Participation status.
- `total_score`: Score credited to the participant.
- `created_at`: Participation creation timestamp.
- `updated_at`: Last participation update timestamp.

### Validation Rules

- Only registered user participations contribute to registered account progress.
- Eligible completed participation contributes at most once to stats.
- Guest-only participation is not linked to account progress in this phase.

### Relationships

- Belongs to one game.
- Can produce one saved progress summary.
- Feeds public stats when registered and completed.

## State Transitions

### Profile Update

```text
current profile -> validate requested changes -> persist valid changes -> return refreshed safe profile
current profile -> validate requested changes -> reject invalid changes -> preserve current profile
```

### Game Progress Contribution

```text
game in progress -> completed once -> contributes to stats and saved progress
completed game -> repeated processing -> no duplicate contribution
```

### Public Stats Read

```text
user exists with completed games -> aggregate public-safe stats
user exists without completed games -> return zero-state public-safe stats
user missing/unavailable -> return not-found style outcome without private account details
```
