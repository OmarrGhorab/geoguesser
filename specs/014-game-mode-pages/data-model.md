# Data Model: Game Mode Pages

## Overview

The feature composes existing game, challenge, room, party, matchmaking, match,
and progression records. quick_play already satisfies the database game-mode
constraint. One new durable record, Command Receipt, protects create operations
from duplicate effects. Frontend view models are projections and never become
canonical storage.

## Canonical mode registry

| Canonical value | Product label | Authority | Session | Key capabilities |
|---|---|---|---|---|
| solo | Classic Solo | games | guest/account | configurable, fixed rounds, immediate reveal |
| quick_play | Quick Play | games | guest/account | server defaults, fixed rounds, timed, immediate reveal |
| practice | Practice | games | guest/account | untimed, open-ended, immediate reveal, history, neutral |
| daily | Daily Challenge | challenges + games | guest/account | shared daily seed, 5 games/day, timed |
| party_lobby | Party Lobby | rooms + games | guest/account | hosted FFA, readiness, shared reveal, standings |
| casual_solo | Casual Solo | matchmaking + matchplay | account | 1v1, untimed, no rating |
| casual_duo | Casual Duos | parties + matchmaking + matchplay | account | 2v2, untimed, team |
| casual_squad | Casual Squads | parties + matchmaking + matchplay | account | 4v4, untimed, team |
| ranked_solo | Ranked Solo | matchmaking + matchplay + competitive | account | 1v1, 60s, speed bonus, rating |
| ranked_duo | Ranked Duos | parties + matchmaking + matchplay + competitive | account | 2v2, 60s, rating |
| ranked_squad | Ranked Squads | parties + matchmaking + matchplay + competitive | account | 4v4, 60s, rating |

Legacy input normalization is one-way: private_room maps to Party Lobby product
semantics, ranked_standard maps to Ranked Solo, and legacy games.mode ranked is
interpreted through its match/assignment context. New writes never use aliases.

## Existing durable entities

### Game

Existing games row with mode, status, map, round count, timer, scoring version,
totals, and timestamps. Add GameModeQuickPlay to runtime policy. Quick Play uses
the Solo lifecycle but persists quick_play for recovery and analytics.

State transitions:

    pending -> active -> completed
                    \-> abandoned

Quick Play is created directly as active by its atomic start operation. Practice
uses pending/active until explicit end and materializes rounds one at a time.

### Round and Guess

Round retains server-owned media references, start/end/reveal timestamps, and
status. Guess remains unique for a participant/round and durably replayable by
idempotency key. Pre-reveal DTOs exclude true coordinates and provider secrets.

### Daily Challenge and Attempt

Challenge stores the shared daily seed, map, immutable settings, selected
locations, and reset window. Attempt links a player/session to one game and a
daily game number. Metadata projects daily_games_played and
daily_games_total=5. Completion of games 1–4 yields the next pending entry;
completion of game 5 yields the day-complete state.

### Party Lobby

Room stores public code, status, host, settings, capacity, expiry, version, and
linked game. RoomPlayer links membership to the GamePlayer identity. Readiness,
presence, current round, guess counts, and standings are bounded projections.

State transitions:

    lobby -> active -> completed
       \-> cancelled
       \-> expired

Member self-leave changes membership to left. A lobby host cancels the room
before start rather than silently transferring authority; an active host follows
the existing disconnect/reconnect grace and terminal policy.

### Matchmaking Party and Ticket

Party is a registered-only forming/queued/assigned/closed roster with leader,
format, member readiness, and optimistic version. Duo requires exactly two
eligible ready members; Squad requires exactly four. Solo has no durable party.

QueueTicket is Redis coordination while searching. A durable assignment and
match become authoritative after formation.

    not_queued -> searching -> matched
                     \-> not_queued
                     \-> temporarily_unavailable -> snapshot repair

### Match and Match Player

Match links a canonical six-mode game to two teams. Match snapshots expose only
caller-authorized and reveal-safe state. Ranked terminal progression can remain
pending until durable competitive finalization completes.

    matched -> countdown -> active -> terminal
                         \-> forfeit/abandoned -> terminal

### Competitive progression

Existing profile, season, placement/rank, rating transaction, and match history
records remain owned by competitive finalization. Casual and Practice never
apply rating progression.

## New durable entity: Command Receipt

Purpose: make resource-creation retries durable across process and Redis loss.

| Field | Type | Rules |
|---|---|---|
| id | UUID | primary key |
| scope | text | bounded enum: game_create, quick_play_create, room_create |
| actor_kind | text | user or guest |
| actor_fingerprint | text | keyed hash; never a raw guest/session token |
| idempotency_key_hash | text | keyed hash of the 16–128 character client key |
| request_hash | text | canonical request hash for conflict detection |
| resource_kind | text | game or room |
| resource_id | UUID | created durable resource identifier |
| created_at | timestamptz | server time |
| expires_at | timestamptz | bounded retention cleanup |

Constraints and indexes:

- Unique scope, actor_kind, actor_fingerprint, idempotency_key_hash.
- Reusing a key with the same request returns the existing resource.
- Reusing a key with a different request returns idempotency_conflict.
- Resource creation and receipt insertion occur in one PostgreSQL transaction.
- A bounded expires_at index supports cleanup; cleanup never deletes the
  referenced game or room.
- Receipt retention must exceed the maximum UI retry/recovery window and is
  configured/documented; 24 hours is the initial default.

Migration: add the next Goose migration after 00020. Down migration is available
for empty development databases only; production rollback disables new writes
and retains receipts until no active rollout depends on them.

## Frontend projections

### ModeEntry

Fields: canonicalMode, labelKey, descriptionKey, href, icon, group, sessionKind,
teamSize, capabilities, enabled state, and disabledReasonKey.

Validation: exactly eleven entries; no legacy aliases; every href is localized
at render time.

### GameView

Fields: game snapshot, current round, selected guess, submission state, reveal,
result summary, connection state, and mode capabilities. Server snapshots replace
the projection; local state may only represent pending UI interactions.

### PracticeView

Fields: GameView plus cursor-page history, hasMore, next-round pending state, and
end-session pending state. Pages append by cursor and never load all rounds.

### RoomView

Fields: room snapshot/version, current participant role, settings permissions,
readiness, presence, active round/reveal, standings, and connection health.

### MatchmakingView

Fields: selected playlist/format, current party snapshot/version, eligibility,
queue status, search duration, rating window, assignment destination, and
recoverable error. URL selection never overrides an existing authoritative
ticket.

### MatchView

Fields: match snapshot/version, caller/team identity, round phase, locked guess,
team scores, revealed result, chat page, progression state, and connection
health. Opponent private guesses and answers are absent until reveal.

## Validation and privacy invariants

- UUIDs, room codes, cursors, locale, mode, playlist, and format are validated at
  every boundary.
- Map IDs must resolve to active public maps with enough eligible locations.
- Round counts remain 1–10 except Practice's open-ended generated sequence;
  timers remain null or 10–600 seconds.
- Room capacity remains 2–50; queue rosters remain exactly 1, 2, or 4.
- Client-supplied scores, deadlines, roles, versions, destinations, and answers
  are never authoritative.
- Unauthorized room/match/game recovery uses privacy-safe not-found behavior.
- Command receipts, logs, metrics, and client error payloads never expose raw
  session material, hidden coordinates, room secrets, or private identities.
