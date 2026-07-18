# Data Model: Party Lobby And Practice Modes

## Game

The existing game row accepts two new modes.

| Field | Party Lobby | Practice |
|---|---|---|
| `mode` | Canonical `party_lobby`; legacy `private_room` readable | `practice` |
| `status` | Existing pending/active/completed/cancelled lifecycle | Pending → active → completed/abandoned |
| `map_id` | Host-selected | Owner-selected |
| `round_count` | Fixed configured total, 1–10 | Number of materialized rounds, starting at 1 and increasing |
| `timer_seconds` | Default 180, range 10–600 | Null |
| `total_score` | Player totals are authoritative standings | Owner's accumulated accepted score |

Party Lobby and legacy Private Room are multiplayer modes with no team slots or competitive season. Practice is single-player and open-ended by mode semantics. Terminal Practice games cannot create additional rounds.

## Round

- `(game_id, round_number)` remains unique.
- Party Lobby rounds use shared `starts_at` and `ends_at`.
- Practice rounds use sequential positive numbers, null `ends_at`, and become active one at a time.
- Practice creates the next row only after the current row is completed.

## Game Player

- Party Lobby has 2–50 active players, each with null `team_slot` and an individual `total_score`.
- Practice has exactly one owner-matching player with null `team_slot`.
- Registered and guest ownership follows existing session rules.

## Guess

- Unique per round/player.
- `accuracy_score` and `score` are 0–5,000.
- `speed_bonus` is always zero for both new modes.
- Practice reveals its answer immediately.
- Party Lobby delays reveal until all eligible guesses arrive or the deadline closes.

## Room And Room Player

Existing hosted-room entities become the Party Lobby authority.

- `rooms.max_players`: 2–50, default 50 when omitted.
- `rooms.round_count`: 1–10, default 5 when omitted.
- `rooms.timer_seconds`: 10–600, default 180 when omitted.
- Existing lobby, active, completed, expired, and cancelled states remain.

## Party Lobby Standing Projection

| Field | Meaning |
|---|---|
| `player_id` | Stable participant ID |
| `display_name` | Participant snapshot name |
| `total_score` | Sum of accepted round scores |
| `total_distance_meters` | Sum of accepted guess distances |
| `rounds_scored` | Accepted guess count |
| `placement` | Deterministic ordinal ordering |
| `tied` | Another standing has equal score and distance |

Ordering: score descending, distance ascending, joined-at ascending, player ID ascending.

## Practice History Page

- Cursor binds to the game and last round number.
- Default page size 20; maximum 100.
- Each item includes round metadata, accepted guess when present, and the authorized revealed answer.
- Response includes `next_cursor` and `has_more`; it never embeds all open-ended history by default.

## State Transitions

```text
Party Lobby: room lobby → room active → room completed
                    ↘ expired/cancelled

Practice game: pending → active → completed (explicit end)
                            ↘ abandoned

Practice round: active → completed → next active round created on request
```

## Migration

Migration `00020_party_lobby_practice_modes.sql` updates game-mode constraints to admit `party_lobby` and `practice` while retaining every legacy mode. Down is allowed only when no rows use the new modes; otherwise it raises an explicit irreversible-data error.
