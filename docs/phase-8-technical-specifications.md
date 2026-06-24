# Phase 8 Technical Specifications

## Purpose

This document defines implementation-ready technical specifications for the major GeoGuess-style game features after product definition, system design, database design, API design, backend architecture, and frontend architecture.

Phase 7 visual design is intentionally skipped for now. These specs should be stable enough to guide backend, frontend, realtime, testing, and operational implementation without relying on vague feature descriptions.

## Source Documents

- `docs/phase-1-product-definition.md`
- `docs/phase-2-system-design.md`
- `docs/phase-3-database-design.md`
- `docs/phase-4-api-design.md`
- `backend/openapi/openapi.yaml`
- `docs/phase-5-backend-architecture.md`
- `docs/phase-6-frontend-architecture.md`

## Global Engineering Rules

- The Go API is authoritative for game state, scoring, authorization, timers, room state, and results.
- PostgreSQL stores durable facts: users, sessions, maps, locations, games, rounds, guesses, leaderboards, subscriptions, payments, and audit logs.
- Redis stores ephemeral coordination state: active rooms, presence, matchmaking queues, reconnect windows, idempotency keys, hot leaderboards, rate limits, and realtime fanout.
- Exact location latitude and longitude must never be sent to the client before the player has submitted a guess or the round has expired.
- In-app mutations should be initiated by Next.js Server Actions when called from the app UI, then forwarded to the Go API with native `fetch()`.
- Public APIs, payment webhooks, health endpoints, realtime endpoints, and future external consumers use Go Route Handlers/API endpoints directly.
- Client Components are allowed only for browser interactivity such as maps, panorama controls, timers, realtime lobby state, dialogs, and rich form interactions.
- Zustand is UI-state only. It must not store server state, auth tokens, room records, guesses, leaderboards, billing data, or API cache data.
- User-facing text belongs in `next-intl` messages. No hardcoded visible strings.
- Subscriptions, ads, cosmetics, and limits must never create a competitive scoring advantage.

## Spec Conventions

Each feature spec includes:

- Status: `MVP`, `Phase 2`, or `Future`.
- Owner packages: backend and frontend modules responsible for implementation.
- APIs: REST endpoints from the OpenAPI contract or proposed future endpoints.
- Durable data: PostgreSQL tables involved.
- Ephemeral data: Redis keys or state involved.
- State rules: allowed transitions and invariants.
- Security rules: authorization, validation, and abuse controls.
- Performance rules: latency, query, cache, and scaling expectations.
- Accessibility rules: frontend interaction requirements.
- Acceptance criteria: what must be true before the feature is considered complete.
- Test coverage: minimum backend, frontend, integration, and E2E tests.

## Shared Runtime Concepts

### Identity

The app supports both registered users and guests.

- Registered users are stored in `users` and participate through `game_players.user_id`.
- Guests participate through `game_players.guest_identity_hash`.
- A `game_player` is scoped to one game and snapshots display name, role, status, and total score.
- Persistent profile, billing, friends, achievements, and registered leaderboards require a registered user.

### Time

The server owns authoritative time.

- Use a backend clock abstraction in `internal/platform/clock`.
- Round start and end use server timestamps: `starts_at`, `ends_at`, and `revealed_at`.
- Clients may render countdowns locally, but final acceptance of guesses depends only on server time.
- Allow a small network grace window only if explicitly configured by the service. The default should be strict server-time validation.

### Realtime Event Envelope

Realtime events should share one envelope for WebSocket or SSE.

```json
{
  "event_id": "evt_01J...",
  "type": "room.player_joined",
  "room_code": "ABCD12",
  "game_id": "0197...",
  "occurred_at": "2026-06-25T12:00:00Z",
  "version": 7,
  "payload": {}
}
```

Rules:

- `event_id` must be unique for deduplication.
- `version` is a monotonic room-state version from Redis.
- Events are hints for UI updates. The client must refetch authoritative room/game state after reconnect.
- Realtime payloads must not include hidden coordinates before reveal.

### Idempotency

Retryable writes require an `Idempotency-Key` header.

Use idempotency for:

- Create game.
- Start game.
- Create room.
- Start room.
- Submit guess.
- Create checkout.
- Payment webhook processing.

Rules:

- Keys are scoped to actor plus operation plus resource.
- A repeated key with the same request returns the original result.
- A repeated key with a different request returns `409 Conflict`.
- Payment-like operations should use durable PostgreSQL records. Short-lived gameplay operations can use Redis with TTL plus database uniqueness constraints.

### Error Format

All API errors use the Phase 4 error envelope.

```json
{
  "error": {
    "code": "round_closed",
    "message": "The round is already closed.",
    "request_id": "req_01J..."
  }
}
```

Rules:

- Error `code` is stable and machine-readable.
- `message` is safe to show but should still be localized by the frontend where possible.
- Internal errors must not expose SQL, tokens, secrets, payment data, provider metadata, hidden coordinates, stack traces, or infrastructure details.

## Feature 1: Solo Game

Status: MVP.

### Scope

Solo game proves the core loop: create a game, select round locations server-side, show media, submit a guess, calculate score, reveal result, advance rounds, and show final results.

### Owners

- Backend: `internal/games`, `internal/maps`, `internal/locations`, `internal/platform/postgres`, `internal/platform/redis`
- Frontend: `features/game`, `features/maps`, `app/[locale]/(game)/play`, `app/[locale]/(game)/games/[gameId]`

### APIs

- `POST /api/v1/games`
- `GET /api/v1/games/{gameId}`
- `POST /api/v1/games/{gameId}/start`
- `GET /api/v1/games/{gameId}/rounds/current`
- `POST /api/v1/games/{gameId}/rounds/{roundId}/guesses`
- `GET /api/v1/games/{gameId}/results`

### Durable Data

- `games`
- `rounds`
- `game_players`
- `guesses`
- `maps`
- `locations`
- `map_locations`

### Ephemeral Data

Suggested Redis keys:

```text
game:{game_id}:state
game:{game_id}:current_round
idempotency:game:{actor_id}:{key}
rate_limit:guess:{actor_id}
```

### State Rules

Game states:

```text
pending -> active -> completed
pending -> cancelled
active -> abandoned
```

Round states:

```text
pending -> active -> completed
active -> cancelled
```

Rules:

- Creating a solo game selects all round locations in one transaction.
- A solo game has exactly one active `game_player`.
- Rounds are ordered by `round_number`.
- The current round is the lowest-numbered active round, or the next pending round if the previous round has completed.
- A player can submit at most one guess per round.
- A game completes when the final round is completed.

### Location Selection

- Use only active locations in the selected map.
- Do not repeat a location within the same game.
- Do not use `ORDER BY random()` at scale.
- Use `locations.random_key`, indexed filters, and map membership.
- Store selected `rounds.location_id` in PostgreSQL, but do not expose it in current-round DTOs.

### Frontend Behavior

- `/[locale]/play` loads playable maps in a Server Component.
- Starting a game uses a Server Action.
- `/[locale]/games/[gameId]` renders the game shell as a Server Component and places map/panorama UI in Client Components.
- The guess map stores only the unsent local pin in Zustand or local component state.
- After submit, the UI renders the result returned by the backend.

### Security

- Validate `map_id`, `round_count`, `timer_seconds`, and session identity on the backend.
- Enforce map access tiers server-side.
- Do not expose `round.location_id`, true lat/lng, provider hidden metadata, or admin notes before reveal.
- Rate limit game creation and guess submission.

### Performance

- Game creation should complete within a normal request budget. Avoid expensive full-table random scans.
- Current round reads should be cached briefly in Redis, but hidden coordinate data must remain backend-only.
- Heavy map/panorama libraries load only on game routes.

### Accessibility

- The game page must have a semantic `main` and a localized `h1`.
- Guess submit control must be keyboard reachable.
- Result text must include distance and score in text, not color alone.
- Timer updates should not spam screen readers.

### Acceptance Criteria

- A guest or registered player can create a solo game.
- Round DTOs exclude true coordinates before reveal.
- Guess submission returns distance, score, and revealed location.
- Duplicate guess submission is idempotent or rejected consistently.
- Final results include all rounds and total score.

### Tests

- Unit: scoring formula, location selection no-repeat rule, state transitions.
- Handler: create game, current round, submit guess, final results.
- Repository: game creation transaction and round lookup with Testcontainers PostgreSQL.
- Integration: complete 5-round solo game.
- E2E: start game, place pin, submit, view result, finish game.

## Feature 2: Multiplayer Rooms

Status: MVP for private rooms, Phase 2 for public rooms and advanced host controls.

### Scope

Rooms allow a host to create a lobby, invite players by code, configure rules before start, synchronize rounds, collect guesses, and show shared results.

### Owners

- Backend: `internal/rooms`, `internal/games`, `internal/realtime`, `internal/platform/redis`
- Frontend: `features/rooms`, `features/game`, `app/[locale]/(game)/rooms`, `app/[locale]/(game)/rooms/[roomCode]`

### APIs

- `POST /api/v1/rooms`
- `POST /api/v1/rooms/join`
- `GET /api/v1/rooms/{roomCode}`
- `PATCH /api/v1/rooms/{roomCode}/settings`
- `POST /api/v1/rooms/{roomCode}/start`
- `DELETE /api/v1/rooms/{roomCode}/players/{playerId}`
- Realtime: `/realtime/rooms/{roomCode}` or equivalent WebSocket/SSE endpoint.

### Durable Data

- `rooms`
- `room_players`
- `games`
- `game_players`
- `rounds`
- `guesses`

### Ephemeral Data

Suggested Redis keys:

```text
room:{code}:state
room:{code}:presence
room:{code}:ready
room:{code}:version
room:{code}:events
room:{code}:lock
room_code:{code}
```

### Room Lifecycle

```text
lobby -> active -> completed
lobby -> expired
lobby -> cancelled
active -> abandoned
```

Rules:

- A room code must be unique among active room statuses.
- Room codes are uppercase, non-enumerable, and between 6 and 10 characters.
- Room creation creates or reserves the game shell required for gameplay.
- A room in `lobby` can change settings.
- A room in `active` cannot change round count, timer, map, or max players.
- A completed or expired room cannot be joined.
- Expired room cleanup is handled by background jobs.

### Join Room

Flow:

1. Client submits a room code and optional guest display name.
2. Backend normalizes and validates code.
3. Backend loads room by code.
4. Backend rejects if room is not in `lobby`, is full, expired, or the player is blocked/kicked.
5. Backend creates or reuses a `game_player`.
6. Backend writes `room_players`.
7. Backend updates Redis room presence.
8. Backend broadcasts `room.player_joined`.
9. Client renders the current lobby state.

Rules:

- Registered users cannot join the same room twice.
- Guest rejoin is allowed when the guest session matches the previous `guest_identity_hash`.
- Display names are snapshotted on `game_players`.
- Room join should be rate limited by actor and IP.

### Ready State

Status: Phase 2 unless the MVP requires host-ready flow.

Rules:

- Ready state is ephemeral and stored in Redis.
- Host can start when all non-spectator active players are ready, or when host explicitly overrides if allowed.
- A player joining, leaving, disconnecting, or changing settings clears ready state when the change affects fairness.
- Ready state must not be persisted as a permanent gameplay fact.

Suggested events:

```text
room.player_ready
room.player_unready
room.ready_reset
```

### Host Controls

MVP:

- Start room.
- Update settings before start.
- Remove player.

Future:

- Transfer host.
- Restart room.
- Assign teams.
- Spectator mode.
- Custom map selection.

Rules:

- Host authorization is checked in `rooms.Service`, not only middleware.
- If host disconnects in lobby, host can transfer to earliest joined active player after grace window.
- Host removal commands must be audited for registered rooms when moderation tooling exists.

### Frontend Behavior

- Room page loads the initial room state on the server.
- A small Client Component opens realtime connection and updates lobby UI.
- Commands use Server Actions or backend HTTP endpoints.
- Realtime events update visual state, then the client refetches on reconnect or version mismatch.
- Invite dialog copies room code and share URL using a localized accessible control.

### Security

- Room codes must be generated with cryptographic randomness.
- Do not expose hidden location data in room state.
- Rate limit room creation, joining, settings updates, start, and kick.
- Validate max players, timer, round count, and map access on the backend.
- Guests can join MVP rooms, but registered-only features remain protected.

### Performance

- Active room state should be read from Redis, not repeatedly rebuilt from PostgreSQL on every event.
- Use Redis locks for start-room race prevention.
- Realtime broadcasts should be compact and versioned.

### Accessibility

- Lobby player list should announce join/leave changes politely.
- Ready/start buttons need stable accessible names.
- Host-only controls should be hidden or disabled with explanatory localized text.

### Acceptance Criteria

- A host can create a private room and receive a code.
- Another player can join by code.
- Room state is synchronized across connected clients.
- Host can start the game.
- Non-hosts cannot start or change settings.
- Room cannot be joined after start unless reconnect rules allow it.

### Tests

- Unit: room code generation, room state transitions, host permission rules.
- Handler: create, join, settings, start, remove player.
- Redis integration: presence and ready-state behavior.
- E2E: host creates room, player joins, host starts, both receive round start.

## Feature 3: Timer And Round Synchronization

Status: MVP for timed rounds.

### Scope

Timers keep solo and multiplayer rounds fair. The backend owns round start and end timestamps; clients render countdowns for UX only.

### Owners

- Backend: `internal/games`, `internal/rooms`, `internal/realtime`, `internal/platform/clock`
- Frontend: `features/game/components/round-timer`, `features/rooms`

### Durable Data

- `rounds.starts_at`
- `rounds.ends_at`
- `rounds.revealed_at`
- `guesses.submitted_at`

### Ephemeral Data

```text
room:{code}:round:{round_id}:deadline
game:{game_id}:round:{round_id}:state
```

### Rules

- `starts_at` is set by the backend when the round becomes active.
- `ends_at = starts_at + timer_seconds` for timed rounds.
- Untimed rounds have `ends_at = null`.
- Multiplayer rounds start for all players from the same server timestamp.
- A guess is accepted only if the backend receives it before the allowed deadline.
- When all active players submit, the round may end early.
- Late guesses return `422` with `round_closed` or a similar stable code.
- The client can display local countdown drift, but cannot extend the server deadline.

### Round Transition

```text
round.pending
  -> round.active
  -> round.completed
  -> next round.pending or game.completed
```

Round completion occurs when:

- All active players have submitted; or
- Server deadline passes; or
- Host/admin cancels the game.

### Realtime Events

```text
round.started
round.guess_count_changed
round.ended
round.results_revealed
game.completed
```

Rules:

- `round.started` includes `starts_at`, `ends_at`, round number, and media only.
- `round.ended` can include aggregate status but not hidden location unless reveal is allowed.
- `round.results_revealed` includes revealed location and player results.

### Frontend Behavior

- Timer component is a Client Component.
- It receives server timestamps from a Server Component or API response.
- It renders remaining time from local clock but treats backend response as authoritative.
- On reconnect, the client refetches current round state and recalculates remaining display time.

### Security

- Never trust client-submitted timestamps.
- Do not accept guesses for non-current rounds.
- Do not accept guesses from players outside the game.
- Do not expose true location because a timer expired until the backend marks the round revealed.

### Performance

- Avoid per-second server polling for timers.
- Use a single event at round start and optional event at round end.
- Background workers may close expired rounds if no request naturally triggers completion.

### Accessibility

- Render timer as text.
- Use `aria-live="polite"` only for important thresholds, not every second.
- Give visual and non-visual warning near timeout.

### Acceptance Criteria

- Timed rounds start with shared server timestamps.
- Late guesses are rejected consistently.
- All-player-submit ends multiplayer round early.
- Refresh/reconnect shows correct remaining time.

### Tests

- Unit: deadline calculation and late-guess validation.
- Integration: round expires and rejects guesses.
- E2E: two players see synchronized round start and result reveal.

## Feature 4: Reconnection And Disconnect Handling

Status: MVP for private rooms.

### Scope

Players should survive short network interruptions without corrupting room state or scoring.

### Owners

- Backend: `internal/rooms`, `internal/realtime`, `internal/games`
- Frontend: `features/rooms/realtime.ts`, `features/game`

### Ephemeral Data

```text
room:{code}:presence
room:{code}:reconnect:{game_player_id}
ws:{connection_id}
```

### Rules

- Presence is heartbeat-based.
- A disconnected player enters `disconnected` state after missed heartbeat threshold.
- Reconnect grace defaults to 30 seconds.
- During lobby, disconnected non-hosts can be removed after grace.
- During active round, disconnected players keep their slot until grace expires or game ends.
- If a player reconnects before the round deadline, they can continue.
- If a player does not reconnect before the round closes, they receive no guess and 0 points for that round.
- Completed guesses are preserved even if the player disconnects.
- Results and final scores are durable and reloadable from PostgreSQL.

### Reconnect Flow

1. Client loses realtime connection.
2. UI shows reconnecting state.
3. Client reconnects using session cookies.
4. Backend resolves user or guest identity.
5. Backend checks matching room/game player.
6. Backend reattaches presence and sends current room/game state.
7. Client discards stale local realtime state and renders authoritative state.

### Security

- Reconnect requires the same registered user session or guest identity.
- Do not allow taking over another `game_player_id` from client input.
- Guest identity cookies must be signed and random.

### Performance

- Heartbeat interval should be balanced to avoid noisy Redis writes.
- Presence TTL should expire abandoned sessions without manual cleanup.

### Accessibility

- Reconnect state should be announced without trapping focus.
- Gameplay controls should clearly show disabled state while disconnected.

### Acceptance Criteria

- Refreshing during a room returns the player to the same room state.
- Reconnecting during an active round preserves submitted guesses.
- A missed round produces 0 points without blocking the room.
- Host disconnect in lobby transfers or preserves host behavior according to product decision.

### Tests

- Redis integration: heartbeat expiration and reconnect TTL.
- E2E: disconnect one player, reconnect, submit before deadline.
- E2E: disconnect through round end and receive 0.

## Feature 5: Scoring

Status: MVP.

### Scope

Scoring calculates distance and points for each guess and persists historical results.

### Owners

- Backend: `internal/games/scoring.go`
- Frontend: `features/game/components/result-card`, optional duplicated display-only helpers

### Formula

Use the Phase 1 scoring formula.

```text
score = round(maxScore * e^(-distanceKm / decayFactorKm))
```

Defaults:

```text
maxScore = 5000
decayFactorKm = 1492
fullScoreThresholdMeters = 25
```

If:

```text
distanceMeters <= 25
```

Then:

```text
score = 5000
```

### Distance

- Use haversine distance for MVP.
- Calculate distance on the server.
- Store `distance_meters` and `score` on `guesses`.
- Store `games.scoring_version` to preserve historical meaning if formula changes.

### Rules

- Scores are integers between 0 and 5000 per round.
- Total score is the sum of guess scores for the player.
- Missing guess equals 0.
- Speed bonus is deferred. If introduced later, it must be versioned and cannot dominate accuracy.
- Client-side scoring helpers can exist only for display previews or tests, never as authority.

### Idempotency

- Unique constraint: `(round_id, game_player_id)`.
- Optional idempotency key: `(game_player_id, idempotency_key)`.
- Duplicate same request returns same result.
- Duplicate different guess after first accepted guess returns `409` or existing result according to final API decision.

### Anti-Cheat

- Never expose true coordinates before submission or timeout.
- Do not trust client distance or score.
- Reject guesses outside valid lat/lng bounds.
- Reject guesses for players not in the game.
- Reject guesses for closed or non-current rounds.
- Rate limit guess submissions.

### Performance

- Scoring should be CPU-cheap and run inline in request.
- Guess submit transaction should update guess and player total together.
- Avoid N+1 result reads by batching round/guess/player lookups.

### Accessibility

- Result UI must show numeric score, distance, and location text.
- Do not use only color or animation to communicate success.

### Acceptance Criteria

- Server calculates score deterministically.
- 25m or less returns 5000.
- Far guesses return low but non-negative score.
- Stored scores do not change if formula constants change later.

### Tests

- Unit: haversine known distances, full-score threshold, decay formula, rounding.
- Property: score is always within 0 to 5000.
- Integration: submit guess persists score and updates total.

## Feature 6: Leaderboards

Status: Phase 2. Public global/map leaderboards can be MVP-adjacent if desired.

### Scope

Leaderboards rank registered users by completed game results. Guest play can show local result rankings but should not enter durable public leaderboards by default.

### Owners

- Backend: `internal/leaderboards`, `internal/games`, `internal/jobs`
- Frontend: `features/leaderboard`, `app/[locale]/(app)/leaderboard`

### APIs

- `GET /api/v1/leaderboards/global`
- `GET /api/v1/leaderboards/daily`
- `GET /api/v1/leaderboards/maps/{mapId}`
- Proposed future:
  - `GET /api/v1/leaderboards/countries/{countryCode}`
  - `GET /api/v1/leaderboards/friends`
  - `GET /api/v1/leaderboards/seasons/{seasonId}`

### Durable Data

- `leaderboards`
- `leaderboard_entries`
- `games`
- `game_players`
- `guesses`
- `users`
- `user_profiles`
- `friendships` for friends scope

### Ephemeral Data

```text
leaderboard:{kind}:{scope}:hot
leaderboard:{kind}:{scope}:page:{cursor_hash}
```

### Leaderboard Types

#### Global

- Scope: all eligible completed games.
- Eligibility: registered users only by default.
- Ranking: best completed game score or season score, depending on selected product rule.
- Tie breaker: earlier `recorded_at`, then stable `user_id`.

#### Country

- Scope: games played on maps or locations matching country, or users from profile country.
- Product decision required before implementation.
- Recommended: map/location country scope for gameplay fairness, not user nationality.

#### Friends

- Scope: accepted friends of the current registered user plus self.
- Requires authentication.
- Never expose private blocked/declined relationship data.

#### Seasonal

- Scope: completed games within season period.
- Season boundaries are UTC by default.
- Seasonal rules must snapshot map pool and scoring version.

### Update Strategy

1. Game completes in PostgreSQL transaction.
2. Service emits leaderboard candidate update.
3. Hot Redis sorted set is updated.
4. Background job materializes or rebuilds PostgreSQL snapshots.
5. Reads use Redis for hot pages when available and PostgreSQL as source of truth.

### Pagination

- Use cursor pagination.
- Cursor includes score, rank/order value, and stable ID.
- Limit max is 100.
- Sorting and filtering must be allowlisted.

### Security

- Do not include guests in public durable leaderboards unless explicitly approved.
- Do not expose hidden email or account data.
- Protect friends leaderboard with authentication and friendship checks.
- Rate limit leaderboard reads if abused.

### Performance

- Use Redis sorted sets for hot leaderboards.
- Use indexed PostgreSQL reads for snapshots.
- Rebuild jobs must be idempotent and chunked.
- Avoid calculating global ranks from raw guesses on every request.

### Accessibility

- Leaderboard table must have semantic table structure or accessible list semantics.
- Rank changes should not rely only on color.
- Pagination controls must be keyboard accessible.

### Acceptance Criteria

- Completed registered games can update leaderboard candidates.
- Global/map leaderboard returns cursor-paginated rows.
- Redis cache can be rebuilt from PostgreSQL.
- Friends leaderboard excludes non-friends and blocked users.

### Tests

- Unit: ranking and tie breaker logic.
- Repository: leaderboard cursor pagination.
- Integration: game completion updates candidate entry.
- E2E: leaderboard page displays ranks and paginates.

## Feature 7: Matchmaking

Status: Phase 2.

### Scope

Quick play places players into public rooms based on mode and region. MVP can defer or implement simple queue-based matching.

### Owners

- Backend: `internal/matchmaking`, `internal/rooms`, `internal/jobs`
- Frontend: `features/matchmaking`, `app/[locale]/(game)/matchmaking`

### APIs

- `POST /api/v1/matchmaking/queue`
- `DELETE /api/v1/matchmaking/queue`
- `GET /api/v1/matchmaking/status`

### Durable Data

- `matches`
- `match_players`
- `rooms`
- `games`
- `game_players`

### Ephemeral Data

```text
matchmaking:{mode}:{region}
matchmaking:player:{actor_id}
matchmaking:lock:{mode}:{region}
```

### Rules

- Default mode is `quick_play`.
- Default `minPlayers = 2`.
- Default `maxPlayers = 8`.
- Default timeout is 20 seconds.
- Queue entries expire automatically.
- A player can be in only one queue at a time.
- Worker creates a public room when compatible players are available.
- If timeout passes, worker can create a smaller room or keep waiting based on mode config.

### Security

- Rate limit queue enter/leave.
- Do not trust client-selected region blindly if it affects abuse controls.
- Prevent queue duplication across sessions.

### Performance

- Use Redis sorted sets by enqueue timestamp.
- Use locks to prevent multiple workers from matching the same players.
- Avoid PostgreSQL writes until a match/room is formed.

### Acceptance Criteria

- Player can enter, check status, and leave queue.
- Worker forms a room with compatible players.
- Duplicate queue entries are prevented.
- Expired queue entries are cleaned up.

### Tests

- Unit: compatibility selection.
- Redis integration: queue ordering and expiry.
- Worker integration: forms one room without duplicate players.

## Feature 8: Friends

Status: Phase 2.

### Scope

Friends support social play, friends leaderboards, and future invites.

### Owners

- Backend: `internal/friends`, `internal/users`
- Frontend: `features/friends`, `app/[locale]/(app)/friends`

### Proposed APIs

- `GET /api/v1/friends`
- `POST /api/v1/friends/requests`
- `POST /api/v1/friends/requests/{requestId}/accept`
- `POST /api/v1/friends/requests/{requestId}/decline`
- `DELETE /api/v1/friends/{userId}`
- `POST /api/v1/friends/{userId}/block`

### Durable Data

- `friendships`
- `users`
- `user_profiles`

### Rules

- Friendships are symmetric.
- Store `user_a_id` and `user_b_id` sorted to enforce uniqueness.
- Status values: `pending`, `accepted`, `declined`, `blocked`.
- A user cannot friend themselves.
- Blocking prevents requests and visibility where applicable.
- Friends leaderboards only use `accepted` relationships.

### Security

- Registered users only.
- Do not reveal blocked users through search or friend APIs.
- Rate limit friend requests.
- Audit block/unblock if moderation features require it.

### Accessibility

- Friend request actions need clear labels and confirmation for destructive actions.
- Empty states must be localized.

### Acceptance Criteria

- User can send, accept, decline, remove, and block.
- Duplicate friendships are impossible.
- Friends leaderboard sees accepted friends only.

### Tests

- Unit: symmetric pair sorting and status transitions.
- Integration: duplicate request conflict and accept flow.
- E2E: send and accept friend request.

## Feature 9: Ads And Entitlements

Status: Future, but architecture must not block it.

### Scope

Ads monetize free play without interrupting active timed gameplay. Entitlements decide whether a user should see ads or access premium features.

### Owners

- Backend: `internal/ads`, `internal/billing`
- Frontend: `features/ads`, `features/billing`

### APIs

- `GET /api/v1/billing/entitlements`
- Proposed future:
  - `GET /api/v1/ads/placements?context=...`

### Durable Data

- `subscriptions`
- `user_entitlements`
- `payments`

### Ephemeral Data

```text
entitlements:user:{user_id}
ads:placement:{actor_id}:{context}
```

### Placement Rules

Allowed:

- Lobby.
- Between games.
- Final results.
- Non-ranked post-round interstitial only if it does not delay active players.

Disallowed:

- During active timed guessing.
- Blocking map controls.
- Any placement that changes timer fairness, scoring, or competitive outcome.

### Security And Privacy

- Do not send auth tokens, emails, or unnecessary personal data to ad providers.
- Respect child/school-safe requirements when product policy is defined.
- Entitlement decisions are server-owned.

### Acceptance Criteria

- Entitlement service can return `ad_free`.
- Ad placement logic never affects score or timers.
- Ads can be disabled for subscribers without client trust.

### Tests

- Unit: placement allowed/disallowed contexts.
- Integration: entitlement cache invalidation after subscription webhook.

## Feature 10: Payments And Subscriptions

Status: Future.

### Scope

Payments support subscriptions, trials, renewals, cancellation, billing portal, webhook reconciliation, and entitlements such as ad-free play or premium maps.

### Owners

- Backend: `internal/billing`, `internal/platform/payments`, `internal/jobs`
- Frontend: `features/billing`, `app/[locale]/(app)/billing`, `app/[locale]/(marketing)/pricing`

### APIs

- `GET /api/v1/billing/entitlements`
- `POST /api/v1/billing/checkout`
- `POST /api/v1/billing/portal`
- `POST /api/v1/webhooks/payments`

### Durable Data

- `subscriptions`
- `payments`
- `user_entitlements`
- Optional future `payment_webhook_events`
- `audit_logs`

### Subscription States

```text
trialing -> active
trialing -> cancelled
active -> past_due
active -> cancelled
past_due -> active
past_due -> expired
cancelled -> expired
```

### Trial

Rules:

- Trial eligibility is server-side.
- One trial per user per plan family unless explicitly configured.
- Trial start/end timestamps come from payment provider or backend policy.
- Trial entitlement should expire automatically if no active subscription begins.

### Renewal

Rules:

- Provider webhook is source for successful renewal.
- Store provider subscription ID and payment/invoice ID.
- Extend `current_period_end` only after verified webhook.
- Entitlements update in the same transaction as subscription state.

### Cancellation

Rules:

- `cancel_at_period_end` preserves access until period end.
- Immediate cancellation removes future entitlement according to policy.
- Billing portal is preferred for provider-owned payment method changes.

### Webhooks

Rules:

- Verify provider signature before reading event content as trusted.
- Use idempotency by provider event ID.
- Store enough event metadata to replay/reconcile without storing secrets or card data.
- Process in a transaction.
- Return success for already-processed duplicate events.
- Send unexpected failures to Sentry with redacted payloads.

### Security

- Registered users only for checkout and portal.
- Never store raw card data.
- Never log provider secrets, signatures, card data, or full webhook payloads with PII.
- Checkout `success_url` and `cancel_url` must be allowlisted.
- Payment webhooks do not use CSRF, but must use provider signature verification.

### Performance

- Webhooks should complete quickly and defer slow work to jobs if needed.
- Entitlements should be cache-aside in Redis and invalidated on webhook.

### Acceptance Criteria

- User can create checkout session.
- Webhook can create/update subscription and entitlement idempotently.
- Renewals extend entitlement.
- Cancellation changes entitlement according to policy.
- Payment failures do not grant paid access.

### Tests

- Unit: subscription state machine and entitlement derivation.
- Handler: webhook signature failure and success.
- Integration: duplicate webhook event is idempotent.
- E2E: checkout link creation and billing page entitlement display with mocked provider.

## Feature 11: Authentication And Authorization

Status: MVP for auth foundation, Phase 2 for advanced account controls.

### Scope

Authentication uses JWT access tokens, refresh tokens, HTTP-only cookies, CSRF protection, and guest sessions.

### Owners

- Backend: `internal/auth`, `internal/users`, `internal/middleware`
- Frontend: `features/auth`, `lib/auth`, `app/[locale]/(auth)`

### APIs

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `POST /api/v1/auth/refresh`
- `GET /api/v1/auth/me`

### Durable Data

- `users`
- `user_profiles`
- `auth_sessions`

### Rules

- Access token is short-lived.
- Refresh token is long-lived, rotated, and stored only as a hash.
- Cookies are `HttpOnly`, `Secure`, and `SameSite`.
- Unsafe cookie-authenticated requests require CSRF protection.
- Passwords use Argon2id or bcrypt.
- Role-Based Access Control supports `user`, `moderator`, and `admin`.
- Authorization for domain actions belongs in services.

### Guest Sessions

- Guest session cookie is signed and random.
- Persist only guest identity hash when needed.
- Guest sessions cannot access profile, billing, friends, or registered leaderboards.

### Security

- Rate limit login, register, refresh, and password-sensitive actions.
- Use account lockout or brute-force protection after repeated failures.
- Never store tokens in localStorage, Zustand, or client-visible props.
- Never log passwords, tokens, CSRF values, or refresh hashes.

### Acceptance Criteria

- User can register, login, refresh, and logout.
- Auth cookies have correct flags.
- Refresh rotation invalidates old refresh tokens.
- Guest identity works for gameplay without registered access.

### Tests

- Unit: password hashing, token signing/verification, cookie options.
- Integration: login/refresh/logout flow.
- Security: CSRF rejection and rate limiting.

## Feature 12: Maps, Locations, And Media

Status: MVP.

### Scope

Maps define playable location pools. Locations hold hidden coordinates and provider media references.

### Owners

- Backend: `internal/maps`, `internal/locations`
- Frontend: `features/maps`, `features/game`

### APIs

- `GET /api/v1/maps`
- `GET /api/v1/maps/{mapId}`
- `GET /api/v1/locations/{locationId}/media`

### Durable Data

- `maps`
- `locations`
- `map_locations`

### Rules

- Public map list exposes metadata only.
- Location media endpoint returns media URL and attribution only when requester can view it.
- Location coordinates are hidden until reveal.
- Provider attribution must be displayed when required.
- Premium maps require entitlement checks.

### Performance

- Cache public map list with Next.js cache and Redis.
- Cache location media metadata only where licensing permits.
- Avoid random full-table scans for map selection.

### Security

- Do not expose provider references if they reveal location coordinates or hidden metadata.
- Admin import/update routes require admin authorization and audit logs when added.
- Validate provider URLs and signed URL expiry.

### Accessibility

- Media viewer needs alternative text or accessible description where practical.
- Attribution text must be visible and localized if owned by app.

### Acceptance Criteria

- Public maps can be listed and opened.
- Current round can show media without exposing coordinates.
- Inactive locations are never selected.

### Tests

- Unit: access tier decisions and media DTO shaping.
- Integration: map list cache and location selection.

## Feature 13: Profiles And Stats

Status: MVP-adjacent.

### Scope

Profiles provide persistent identity, locale, country, avatar, and public gameplay stats for registered users.

### Owners

- Backend: `internal/profiles`, `internal/users`
- Frontend: `features/profile`, `app/[locale]/(app)/profile`

### APIs

- `GET /api/v1/profile`
- `PATCH /api/v1/profile`
- `GET /api/v1/users/{userId}/stats`

### Durable Data

- `users`
- `user_profiles`
- `games`
- `game_players`
- `guesses`

### Rules

- Profile update requires registered auth.
- Display name length is 2 to 32 characters.
- Locale must be allowlisted.
- Public stats must not expose private account fields.

### Security

- Avatar URLs must be validated or mediated through file storage later.
- Do not expose email in public profile responses unless explicitly allowed.
- Rate limit profile updates.

### Acceptance Criteria

- User can view and update profile.
- Public stats endpoint returns safe aggregate data.
- Locale preference integrates with frontend routing or future redirect logic.

### Tests

- Handler: profile read/update validation.
- Integration: stats aggregate from completed games.

## Feature 14: File Storage And Uploads

Status: Future.

### Scope

File storage supports avatars, future user-created maps, and admin-imported media where licensing allows.

### Owners

- Backend: `internal/platform/storage`, future feature packages
- Frontend: `features/profile`, future `features/maps`

### Proposed APIs

- `POST /api/v1/uploads`
- `POST /api/v1/uploads/{uploadId}/complete`
- `GET /api/v1/files/{fileId}/signed-url`

### Rules

- Validate file size, MIME type, extension, and content where possible.
- Stream uploads instead of buffering large files in memory.
- Use S3-compatible storage for production.
- Use local storage only in development.
- Use signed URLs for private files.
- Never trust client-provided filenames as storage paths.

### Security

- Prevent path traversal.
- Strip dangerous metadata when possible.
- Restrict executable file types.
- Apply antivirus or malware scanning if public uploads become important.

### Acceptance Criteria

- Uploads enforce size and type limits.
- Signed URLs expire.
- Files cannot overwrite another user's objects.

### Tests

- Unit: file validation.
- Integration: local/S3-compatible storage adapter.
- Security: path traversal and oversized upload rejection.

## Feature 15: Email

Status: Future.

### Scope

Email supports verification, password reset, billing notifications, and product communications.

### Owners

- Backend: `internal/platform/email`, `internal/jobs`, `internal/auth`, `internal/billing`

### Rules

- Send emails from background jobs when possible.
- Use templates with localized content.
- Retry transient failures with exponential backoff.
- Do not include secrets directly in URLs without short expiration and hashing.
- Store only hashed reset/verification tokens.

### Security

- Rate limit email-triggering actions.
- Avoid account enumeration in password reset responses.
- Redact email provider errors in user-facing responses.

### Acceptance Criteria

- Email client is interface-driven and mockable.
- Background sending retries transient failures.
- Verification/reset links expire.

### Tests

- Unit: token generation and expiry.
- Integration: mocked SMTP/provider send.
- Job test: retry and dead-letter behavior.

## Feature 16: Background Jobs

Status: Phase 2 foundation.

### Scope

Background jobs handle cleanup, reconciliation, leaderboard rebuilds, session expiry, emails, and payment maintenance.

### Owners

- Backend: `cmd/worker`, `internal/jobs`

### Jobs

- Expire abandoned rooms.
- Remove stale matchmaking entries.
- Rebuild leaderboard snapshots.
- Expire old sessions.
- Reconcile payment events.
- Send emails.
- Clean old idempotency keys if stored durably.

### Rules

- Every job accepts `context.Context`.
- Every job supports graceful shutdown.
- Use retries with exponential backoff for transient failures.
- Use locks or idempotency for singleton work.
- Jobs log structured JSON with job name, run ID, request/correlation ID where available, and outcome.

### Performance

- Process large tables in chunks.
- Use indexed queries.
- Avoid long transactions.

### Acceptance Criteria

- Worker starts and stops gracefully.
- Failed jobs are logged and observable.
- Cleanup jobs are idempotent.

### Tests

- Unit: retry/backoff policy.
- Integration: worker shutdown and lock behavior.
- Repository: chunked cleanup query behavior.

## Feature 17: Observability And Health

Status: MVP foundation.

### Scope

The system must be debuggable in production through logs, metrics, traces, health checks, dashboards, alerts, and Sentry errors.

### Owners

- Backend: `internal/platform/observability`, `internal/health`, `internal/middleware`
- Frontend: Next.js instrumentation later
- Infrastructure: Nginx, Prometheus, Grafana, Sentry, OpenTelemetry collector if used

### APIs

- `GET /api/v1/health`
- `GET /api/v1/ready`
- `GET /api/v1/metrics`

### Requirements

- JSON structured logs through `log/slog`.
- Request ID on every request.
- Correlation ID propagation across Next.js, Go API, Redis/Postgres logs where practical, and traces.
- OpenTelemetry spans for HTTP requests, database calls, Redis calls, payment calls, and major game operations.
- Prometheus metrics for API, rooms, matchmaking, guesses, leaderboards, payments, Redis, PostgreSQL, and workers.
- Grafana dashboards for API health, gameplay health, realtime health, database/Redis health, payments, and background jobs.
- Sentry captures unexpected backend errors with sensitive data redacted.

### Health Semantics

- `/health`: liveness. Returns ok if process can serve.
- `/ready`: readiness. Checks PostgreSQL, Redis, critical config, and required providers for current feature set.
- `/metrics`: Prometheus scrape, restricted by network or auth.

### Core Metrics

```text
http_request_duration_seconds
http_requests_total
active_rooms
active_realtime_connections
matchmaking_queue_length
matchmaking_wait_seconds
guess_submission_duration_seconds
round_start_latency_seconds
leaderboard_rebuild_duration_seconds
payment_webhook_failures_total
postgres_errors_total
redis_errors_total
```

### Security

- `/metrics` must not be public internet accessible.
- Logs and Sentry must redact tokens, cookies, passwords, CSRF tokens, payment secrets, and PII where possible.
- Health endpoints must not leak credentials or internal topology.

### Acceptance Criteria

- Every request has request ID and structured log.
- Traces connect Next.js server calls to Go API where possible.
- Metrics endpoint exposes core metrics.
- Sentry receives unexpected errors without sensitive data.
- Grafana dashboard and alerts are defined before production.

### Tests

- Handler: health and readiness responses.
- Integration: request ID propagation and metrics increment.
- Review: Sentry redaction and no sensitive logs.

## Feature 18: API Documentation And OpenAPI

Status: MVP foundation.

### Scope

OpenAPI 3.1 is the source of truth for REST contracts.

### Owners

- Backend: `backend/openapi/openapi.yaml`, feature handlers
- Frontend: `lib/api/schemas.ts`, feature schemas

### Rules

- Every public endpoint has request schema, response schema, auth rules, examples, tags, and error responses.
- DTOs must not expose raw GORM models.
- Swagger UI can run locally or in protected internal environments only.
- API changes update OpenAPI in the same change.
- Generated types may be introduced later, but the spec remains authoritative.

### Versioning

- Current base path is `/api/v1`.
- Breaking API changes require versioning or compatibility window.
- Additive fields are preferred over breaking response changes.

### Acceptance Criteria

- OpenAPI validates in CI.
- Every implemented endpoint exists in OpenAPI.
- Every schema ref resolves.
- Authentication and errors are documented.

### Tests

- CI: OpenAPI lint/validation.
- Contract: implemented routes match documented routes.
- Frontend: Zod schemas align with OpenAPI responses.

## Feature 19: Security And Abuse Protection

Status: MVP foundation.

### Scope

Security protects accounts, gameplay fairness, payment operations, location secrecy, infrastructure, and user data.

### Owners

- Backend: `internal/middleware`, `internal/auth`, `internal/games`, `internal/rooms`, `internal/billing`
- Frontend: Server Actions and route protection
- Infrastructure: Nginx

### Requirements

- HTTP-only secure auth cookies.
- CSRF protection for unsafe cookie-authenticated requests.
- Rate limiting for auth, room creation, room join, matchmaking, guess submission, profile update, and payment operations.
- Security headers at Nginx and app layer where appropriate.
- HSTS in production.
- CSP compatible with map/media/payment providers.
- CORS locked down.
- Request body size limits.
- Audit logs for admin, auth security events, billing, and moderation actions.
- Brute-force protection and account lockout policy.
- Secure secret management through environment variables or provider secrets.

### Gameplay-Specific Security

- Hidden coordinates stay server-side until reveal.
- Provider metadata that can identify coordinates is hidden until reveal.
- Score calculation is server-only.
- Guess timestamps are server-only.
- Room codes are non-enumerable and rate limited.

### Acceptance Criteria

- Security-sensitive endpoints have rate limits.
- Unsafe methods require CSRF except verified webhooks.
- Secrets never appear in logs.
- Hidden coordinates are absent from pre-reveal responses.

### Tests

- Security tests for CSRF, cookie flags, rate limit, unauthorized access, and hidden coordinate leakage.
- Contract tests for forbidden fields in current round DTOs.

## Feature 20: Frontend UX, Localization, And Accessibility

Status: MVP foundation.

### Scope

The frontend should be production-grade, localized, accessible, and server-first.

### Owners

- Frontend: `app/[locale]`, `features/*`, `components/ui`, `messages`

### Rules

- Use Server Components by default.
- Use Client Components only for interaction.
- Use Server Actions for in-app mutations.
- Use native `fetch()` in server-only data modules.
- Use next-intl for every user-facing string.
- Initial locales: `en` and `ar`.
- Support RTL through `dir`, logical CSS, and layout testing.
- Use shadcn/ui and Radix for accessible primitives.
- Use React Hook Form and Zod only where rich client forms need them.

### Performance

- Keep gameplay client bundle small.
- Load map/panorama libraries only on game routes.
- Stream slow data with Suspense.
- Cache public map and leaderboard data carefully.
- Avoid duplicate backend fetches.

### Accessibility

- Every page has one clear `h1`.
- Forms have labels and accessible errors.
- Dialogs, menus, popovers, tabs, and tooltips use accessible primitives.
- Timers, results, and realtime lobby updates are screen-reader friendly.
- Keyboard users can navigate core gameplay controls where practical.
- Color is never the only result signal.

### Acceptance Criteria

- No hardcoded visible strings in implemented UI.
- RTL layout works for Arabic.
- Core game flow is keyboard-accessible where practical.
- Lighthouse/Core Web Vitals targets are tracked before production.

### Tests

- Component tests for forms and interactive controls.
- E2E for start game, submit guess, room join.
- Accessibility checks with automated tooling and manual keyboard pass.
- RTL visual pass.

## MVP Implementation Readiness Checklist

Before writing MVP code:

- Confirm imagery provider and licensing.
- Confirm whether guests can create private rooms or only join.
- Confirm whether private rooms are MVP or immediately after solo.
- Confirm whether quick play is MVP or Phase 2.
- Confirm whether global leaderboards include only registered users.
- Confirm payment provider before billing implementation.
- Confirm ad provider and school/child-safe policy before ads.
- Confirm WebSocket versus SSE for realtime MVP.

## Phase 8 Exit Criteria

Phase 8 is complete when:

- Feature specs are accepted for solo, multiplayer rooms, timer synchronization, reconnection, scoring, leaderboards, payments, subscriptions, and webhooks.
- Server authority and hidden-coordinate rules are preserved in every relevant spec.
- Backend package ownership is clear.
- Frontend route/feature ownership is clear.
- Redis and PostgreSQL responsibilities are clear.
- Acceptance criteria and test coverage are defined.
- Open questions are ready for prioritization before implementation.
