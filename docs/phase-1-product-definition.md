# Phase 1 Product Definition

## Product Summary

This project is a free-first geography guessing game inspired by GeoGuessr. Players are dropped into a real-world location, inspect visual clues, place a guess on a map, and receive a score based on distance from the true location.

The product should feel fast, social, and accessible without requiring a paid subscription to enjoy the core loop. Monetization should be designed around ads, cosmetics, convenience, private room options, and future subscriptions, not pay-to-win mechanics.

## Product Goals

- Let players quickly start a solo or multiplayer geography guessing round.
- Make the core game playable for free.
- Support repeatable game modes using curated location sets.
- Reward geographic knowledge, observation, and speed.
- Keep the MVP small enough to ship, test, and improve.
- Build foundations for multiplayer, subscriptions, ads, leaderboards, and content expansion.

## Non-Goals For Phase 1

- No full paid subscription implementation.
- No real-money rewards.
- No complex ranked matchmaking.
- No creator marketplace.
- No native mobile app.
- No full Street View replacement engine.
- No user-generated map moderation system beyond basic admin-managed location sets.

## Target Users

- Casual geography players who want a free alternative to paid geography games.
- Friends who want quick private multiplayer rooms.
- Competitive players who want timed challenges and leaderboards later.
- Educators or streamers who want shareable geography rounds in the future.

## Core Gameplay Loop

1. Player chooses a mode: solo challenge, private room, or public quick play.
2. The system creates a game session with a selected map or location pool.
3. A round starts by loading a random location.
4. The player inspects the location and surrounding clues.
5. The player places a pin on the map.
6. The player submits the guess before the timer expires.
7. The system calculates distance and score.
8. The result screen shows the real location, guess location, distance, score, and round summary.
9. The next round starts until the game reaches the configured round count.
10. The final screen shows total score, ranking, and options to replay, share, or start another game.

## Round Start

For MVP, a round starts when:

- A solo player clicks `Start Game`.
- A private room host clicks `Start`.
- A public match has enough players or a matchmaking timeout creates a smaller match.

The backend should create a game session, select round locations server-side, and return only the data needed to render the current round. Exact coordinates must not be exposed to the client before the player submits or the timer expires.

## Location Sources

### MVP

Use an admin-curated location database.

Each location should include:

- UUID
- Latitude and longitude
- Country
- Region/state when available
- City or locality when available
- Difficulty level
- Tags such as urban, rural, road, landmark, coast, mountain, desert
- Image or panorama provider reference
- Attribution metadata
- Active/inactive status

### Provider Options

The MVP can support one provider first:

- Static images from a curated dataset.
- Street-level imagery from a third-party provider if licensing permits.
- Mapillary-style public imagery if usage and attribution rules are acceptable.

The product must not assume free unlimited Google Street View usage unless licensing and billing are explicitly accepted.

### Future

- Community map packs.
- Country-specific maps.
- Daily challenge location sets.
- Sponsored or educational maps.
- Difficulty-balanced ranked pools.

## Scoring

### MVP Formula

Use distance-based scoring with a maximum of 5,000 points per round.

Recommended formula:

```text
score = round(maxScore * e^(-distanceKm / decayFactorKm))
```

Default values:

```text
maxScore = 5000
decayFactorKm = 1492
```

This rewards close guesses strongly while still giving partial credit for broad regional knowledge.

### Exact Guess Bonus

If the guess is within a small threshold, award full score:

```text
if distanceMeters <= 25:
  score = 5000
```

### Timed Modes

Timed modes may add a speed bonus only after the base game feels fair. Speed bonuses should never dominate location accuracy.

### Anti-Cheat Rule

Coordinates, provider metadata, and hidden identifiers must not leak exact location data before scoring.

## Game Modes

### MVP

- Solo Classic: 5 rounds, timer optional, total score at the end.
- Private Room: host creates a room code and friends join.
- Quick Play: public room with simple matchmaking if time allows.

### Future

- Daily Challenge
- Ranked Duel
- Battle Royale
- Country Streak
- No-Move Mode
- No-Pan/No-Zoom Mode
- Team Mode
- Custom Maps

## Multiplayer Rooms

### Room Creation

A player creates a room with:

- Room code
- Host player ID
- Selected map/location pool
- Round count
- Timer per round
- Privacy mode: public or private
- Max players

For MVP, private rooms should use short join codes. Authentication should be optional for joining but required for persistent profiles, saved stats, and moderation tools.

### Room Lifecycle

1. Host creates room.
2. Players join lobby.
3. Host starts game.
4. Server selects locations.
5. Each round starts for all connected players.
6. Players submit guesses independently.
7. Round ends when all players submit or the timer expires.
8. Results are broadcast to the room.
9. Game ends after final round.

### Host Controls

MVP host controls:

- Start game
- Remove player
- Change round count before start
- Change timer before start

Future host controls:

- Transfer host
- Restart room
- Custom map selection
- Spectator mode
- Team assignment

## Matchmaking

### MVP

Use simple queue-based matchmaking:

- Player selects `Quick Play`.
- Server places player into a queue by mode and region.
- If a compatible room exists, join it.
- If no room exists within a short timeout, create a new public room.
- Start when minimum players join or timeout expires.

Initial defaults:

```text
minPlayers = 2
maxPlayers = 8
matchmakingTimeoutSeconds = 20
```

### Future

- Skill-based matchmaking.
- Ranked matchmaking.
- Region-aware latency matching.
- Party matchmaking.
- Bot fallback for low-population queues.

## Disconnect Handling

### During Lobby

- If a non-host disconnects, remove them after a grace period.
- If the host disconnects, transfer host to the earliest joined active player.

### During Round

- Keep player state for a short reconnection window.
- If the player reconnects before the timer ends, allow them to continue.
- If they do not reconnect, mark the round as no guess and award 0 points.

Recommended reconnect window:

```text
reconnectGraceSeconds = 30
```

### During Results

- Preserve completed guesses and scores.
- Reconnected players should receive the latest room state.

## Subscriptions And Ads

Subscriptions are not part of Phase 1 implementation, but the product should avoid decisions that make monetization hard later.

### Free Users

Free users should be able to:

- Play solo games.
- Join private rooms.
- Use quick play.
- Earn basic stats.

Potential free limitations:

- Ads between games or after several rounds.
- Limited number of custom private rooms per day.
- Limited access to premium maps later.

### Subscribers

Future subscribers may receive:

- Ad-free gameplay.
- More private room customization.
- Premium map pools.
- Advanced statistics.
- Cosmetics or profile customization.
- Higher daily limits.

### Monetization Rules

- Do not sell competitive advantage.
- Do not make scoring easier for paid users.
- Avoid interrupting active timed rounds with ads.
- Ads should appear between games, in lobbies, or after final results.
- Children and school usage may require stricter ad policies.

## MVP Feature List

### Must Have

- Solo 5-round game.
- Server-selected locations.
- Guess submission.
- Distance calculation.
- Score calculation.
- Round result screen.
- Final result screen.
- Basic location database.
- Basic responsive UI.
- Error states for failed location loads.
- Basic anti-cheat protection by hiding true coordinates.

### Should Have

- Private multiplayer rooms.
- Room code joining.
- Round timer.
- Reconnect handling.
- Basic user profiles.
- Basic game history.
- Admin seed/import flow for locations.

### Could Have

- Public quick play.
- Daily challenge.
- Leaderboard.
- Shareable results.
- Map filters by country or difficulty.
- Ads placeholder integration.

### Won't Have In MVP

- Subscription checkout.
- Ranked ladder.
- Community map creation.
- Native mobile apps.
- Full moderation console.
- Complex achievements.

## User Stories

### Solo Player

- As a player, I want to start a game quickly so I can play without setup.
- As a player, I want to inspect a location and place a map guess so I can test my geography knowledge.
- As a player, I want to see the correct location and my distance so I can learn from each round.
- As a player, I want a final score so I can compare attempts.

### Multiplayer Player

- As a player, I want to create a private room so I can play with friends.
- As a player, I want to join with a room code so I do not need a complicated invite flow.
- As a player, I want everyone to play the same locations so the match feels fair.
- As a player, I want disconnected players to have a chance to return so a temporary network issue does not ruin the game.

### Host

- As a host, I want to configure round count and timer before the game starts.
- As a host, I want to remove disruptive players from a lobby.

### Returning User

- As a returning user, I want my game history saved so I can track improvement.
- As a returning user, I want profile stats so I have a reason to come back.

### Admin

- As an admin, I want to add and disable locations so the playable pool stays healthy.
- As an admin, I want to tag locations so maps can be filtered by difficulty and theme.

## Product Requirements

### Gameplay

- A game consists of configurable rounds, defaulting to 5.
- Each round has exactly one true location.
- Players can submit at most one guess per round.
- The server is the source of truth for score calculation.
- Round results are shown only after submission or timeout.
- Final results aggregate all round scores.

### Locations

- Locations must be selected server-side.
- In multiplayer, all players receive the same location sequence.
- In solo, locations should not repeat within a game.
- In future daily challenge mode, all players receive the same daily sequence.

### Accounts

- MVP can allow guest play.
- Registered users are required for persistent stats.
- Authentication should use secure HTTP-only cookies when implemented.

### Multiplayer

- Rooms require unique join codes.
- Room state must be synchronized from the server.
- Guess submissions must be idempotent.
- Late submissions after the round ends must be rejected.

### Ads

- Ads must never block active guessing input.
- Ad state must be separate from score state.
- Users with future ad-free entitlement should bypass ad placements.

## Non-Functional Requirements

### Performance

- Initial game page should load quickly on mid-range mobile devices.
- Round transition should feel near-instant after assets are ready.
- Location media should be lazy-loaded and cached where licensing permits.
- Map interactions should remain smooth on mobile.
- Server score calculation should complete within tens of milliseconds.

### Scalability

- The backend should support stateless app instances.
- Game and room state should be designed for Redis or another shared store.
- WebSocket or realtime transport should support horizontal scaling later.
- Location selection should avoid expensive random full-table scans at scale.

### Security

- Exact coordinates must not be sent before the guess is locked.
- Server must validate all guesses, timers, room IDs, and player IDs.
- Guest identities must not be trusted for privileged actions.
- Room codes should be hard to enumerate.
- Rate limit room creation, joins, auth attempts, and guess submissions.
- Do not store authentication tokens in localStorage.

### Privacy

- Avoid collecting unnecessary personal data.
- Keep analytics event payloads minimal.
- Do not send precise user location unless explicitly needed and consented.
- Prepare for account deletion and data export later.

### Reliability

- A player refresh should restore current game state when possible.
- Multiplayer rooms should tolerate short disconnects.
- Failed media loads should show recoverable error states.
- Background cleanup should expire abandoned rooms.

### Accessibility

- The app must support keyboard navigation for menus, forms, and dialogs.
- Interactive controls need visible focus states.
- Color must not be the only way to communicate score or result state.
- Timers should be screen-reader friendly without being overly noisy.
- Map interactions need accessible fallbacks where practical.

### Internationalization

- User-facing strings should be localization-ready from the beginning.
- Layout should support right-to-left languages later.
- Country and region labels should come from localized display names when possible.

## Data Model Draft

### Location

```text
id: UUID
latitude: decimal
longitude: decimal
countryCode: string
region: string | null
locality: string | null
difficulty: enum
tags: string[]
provider: string
providerRef: string
attribution: string | null
isActive: boolean
createdAt: timestamp
updatedAt: timestamp
```

### Game

```text
id: UUID
mode: enum
status: lobby | active | completed | abandoned
roundCount: number
timerSeconds: number | null
locationPoolId: UUID | null
createdByPlayerId: UUID | null
createdAt: timestamp
startedAt: timestamp | null
completedAt: timestamp | null
```

### Round

```text
id: UUID
gameId: UUID
roundNumber: number
locationId: UUID
startsAt: timestamp
endsAt: timestamp | null
```

### Guess

```text
id: UUID
roundId: UUID
playerId: UUID
latitude: decimal
longitude: decimal
distanceMeters: number
score: number
submittedAt: timestamp
```

### Room

```text
id: UUID
gameId: UUID
code: string
visibility: private | public
hostPlayerId: UUID
maxPlayers: number
createdAt: timestamp
expiresAt: timestamp
```

## Key Product Decisions

- Build solo first, because it proves the location, map, scoring, and result loop.
- Add private rooms before ranked matchmaking, because friend play is more valuable than complex competitive systems early.
- Keep scoring server-side, because client-side scoring makes cheating easier.
- Use curated locations first, because provider licensing, quality control, and anti-cheat behavior are easier to manage.
- Keep monetization out of active rounds, because interruptions damage the core experience.

## Open Questions

- Which imagery provider will be used for MVP?
- Will guest users be allowed to create private rooms or only join them?
- Should public quick play be included in MVP or Phase 2?
- What countries or regions should be in the initial location pool?
- Should the game allow moving/panning in MVP, or start with static/panorama-only locations?
- Are ads required before launch, or should the MVP launch without monetization?
- What minimum age or school-safe requirements should the product follow?

## Phase 1 Exit Criteria

Phase 1 is complete when:

- The MVP scope is approved.
- Core gameplay rules are agreed.
- Location source strategy is chosen.
- Scoring formula is accepted.
- Multiplayer room behavior is defined.
- Monetization assumptions are documented.
- Non-functional requirements are understood.
- Phase 2 technical architecture can begin without major product ambiguity.
