# Research: Party Lobby And Practice Modes

## Canonical Party Lobby Naming

**Decision**: Write new hosted-room games as `party_lobby`; continue accepting `private_room` as a legacy multiplayer alias.

**Rationale**: The existing room subsystem already implements join-by-code, host controls, up to 50 players, timed shared rounds, reconnection, and realtime state. Reusing it avoids a second lobby authority while the canonical mode makes the product distinction from premade matchmaking parties explicit.

**Alternatives considered**: Keep writing `private_room` and expose only a display alias; rejected because persisted mode analytics and public contracts would never distinguish the new product mode. Backfill every old row; rejected because old active games and older clients should remain stable.

## Party Lobby Defaults And Standings

**Decision**: Default omitted room settings to five rounds, 180 seconds, and 50 maximum players; preserve host override ranges of 1–10 rounds, 10–600 seconds, and 2–50 players. Rank players by total score descending, total distance ascending, join time, then player ID, while separately exposing ties for equal score and distance.

**Rationale**: These limits match existing safe room bounds, the requested three-minute default, and deterministic reload behavior. Score plus distance represents gameplay performance; stable fallbacks prevent pagination/order drift.

**Alternatives considered**: Unlimited lobby size; rejected because realtime fanout and snapshots need a bounded v1 capacity. Rank equal scores by submission speed; rejected because the user requested ordinary unranked scoring and default three-minute rounds, not an undisclosed speed tiebreak.

## Open-Ended Practice Persistence

**Decision**: Create Practice with one round and treat `games.round_count` as the number of rounds materialized so far. After the current round is completed, a transaction locks the game/current round, creates `round_count + 1` under the existing unique `(game_id, round_number)` invariant, updates `round_count`, and returns an existing newly active round on replay.

**Rationale**: Infinite future locations cannot be preselected. Incremental materialization provides durable sequential history without a new parallel game model and makes concurrent next requests deterministic.

**Alternatives considered**: Pre-create a large number of rounds; rejected because any fixed number violates the product behavior and wastes locations/storage. Create one new game per round; rejected because accumulated session history, score, ownership, and reload behavior become fragmented.

## Practice Lifecycle And History

**Decision**: Practice remains `active` after each accepted guess, exposes an explicit next-round command and explicit end command, and adds cursor-paginated round history with default 20/max 100. Ending sets the game terminal but retains history.

**Rationale**: Client-driven advancement fulfills no-timer/no-worker behavior. Pagination is required because an open-ended session cannot safely reuse an unbounded terminal-results payload.

**Alternatives considered**: Automatically create the next round during guess submission; rejected because it couples answer display to location-provider success and makes user-driven stopping unclear. Reuse unbounded `/results`; rejected because response cost grows indefinitely.

## Dependencies And Failure Isolation

**Decision**: Party Lobby continues to use Redis/realtime as the existing room feature does. Practice routes call only session, game persistence, map/location selection, media resolution, and scoring dependencies; no realtime ticket, Redis presence, queue, chat, or lifecycle-worker calls are introduced.

**Rationale**: This makes Practice a reliable low-dependency loop and directly satisfies the no-realtime requirement.

**Alternatives considered**: Publish Practice game events for consistency; rejected because it adds an unnecessary realtime requirement and contradicts the requested mode.

## Progression Neutrality

**Decision**: Both modes are excluded from competitive finalization, matchmade statistics, daily challenge completion, missions, public leaderboards, and wins/losses.

**Rationale**: Party Lobby is explicitly unranked and Practice must not become a stats or rewards farming path.

**Alternatives considered**: Count Practice accuracy toward general profile totals; rejected because current hooks may trigger missions/statistics and would blur learning activity with normal completed games.
