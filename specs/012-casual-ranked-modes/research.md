# Research: Casual And Ranked Team Modes

## 1. Package Ownership

**Decision**: Add `parties`, `matchplay`, and `competitive` packages; generalize existing `matchmaking`, `games`, `realtime`, and `uploads` through narrow interfaces.

**Rationale**: Party membership, active-match collaboration, and seasonal progression have different authorization, persistence, and lifecycle rules. Keeping them separate prevents realtime transport from owning game policy and prevents matchmaking from becoming the owner of chat or ratings.

**Alternatives considered**: Put everything in `matchmaking` (rejected as a mixed-responsibility package); reuse private `rooms` as team parties/matches (rejected because matchmade games have no host/code/settings lifecycle and ranked history must not inherit room semantics).

## 2. Party Persistence And Membership

**Decision**: Persist Duo/Squad parties, members, invites, readiness, and version in PostgreSQL. Require accepted-friend invitations, reject either-direction blocks, use 15-minute invites, require all members ready before queueing, transfer leadership to the earliest remaining member, and retain the party after a match for requeue. Solo uses an implicit one-user queue ticket rather than a durable party.

**Rationale**: Durable membership provides refresh/reconnect recovery and database uniqueness for one active party per user. Friend-only invites and block checks align with the existing social graph and reduce unsolicited party abuse.

**Alternatives considered**: Redis-only parties (lost on restart and weak uniqueness); room codes (outside the requested premade invite flow); random fill (explicitly outside v1).

## 3. Queue Model And Compatibility

**Decision**: Replace player-pair claims with atomic team-ticket claims. Queue keys are the six playlist/format combinations. Each ticket contains an immutable roster snapshot and rating average; per-user pointers enforce exclusivity. Ranked search begins at ±100 team-average rating, expands by 50 every 30 seconds, and caps at ±400. `ranked_standard` remains an accepted deprecated alias for Ranked Solo and legacy rows remain readable.

**Rationale**: Ticket-level claims keep premade rosters atomic and allow exactly equal teams while preserving the current Redis lease/claim/durable-first recovery pattern.

**Alternatives considered**: Enqueue every party member separately (can split teams); PostgreSQL polling queue (more contention at 10,000 searchers); separate queue implementation for each size (duplicated correctness logic).

## 4. Match And Game Authority

**Decision**: `matches` records playlist/format/team result; `game_players.team_slot` and `match_players.team_slot` identify the two teams. `games` remains authoritative for rounds and guesses. Casual uses `timer_seconds = NULL`; Ranked uses 60 seconds. Multiplayer submit responses never reveal the answer until the shared round closes.

**Rationale**: Existing game transactions already guarantee one guess and shared advancement. Adding team slots and hooks is smaller and safer than a parallel game engine, while delayed reveal closes an existing multiplayer information leak.

**Alternatives considered**: Separate team score tables (derived totals would drift); client-calculated totals/bonuses (not competitive-safe); reveal to early submitters (allows team answer leakage).

## 5. Speed Bonus

**Decision**: Store accuracy and speed separately; total guess score is `accuracy + floor(accuracy × 0.05 × remaining_seconds / 60)`, with the bonus capped at 250 and clamped to zero at/after the deadline. Server time and persisted round boundaries are the only timing inputs.

**Rationale**: The bonus rewards confident speed but remains proportional to accuracy, so fast random guesses cannot outrank accurate play. Separate fields make results explainable and preserve the original 5,000-point accuracy curve.

**Alternatives considered**: Flat time bonus (rewards random submissions); direct rating bonus (creates rating inflation and perverse incentives); first-submitter bonus (network-sensitive).

## 6. Rating, Divisions, And Placements

**Decision**: Use a transparent team-average Elo update with `K=32`: `expected = 1 / (1 + 10^((opponent_average - own_average)/400))` and `base_delta = round(32 × (actual - expected))`, where actual is 1/0.5/0. Every non-abandoning teammate receives the same base delta. New profiles start at hidden rating 800 and reveal after five matches. An abandoner receives the team loss delta plus `-15`; rating floors at zero.

**Rationale**: The formula uses only the permitted inputs—outcome and opponent strength—produces symmetric changes, and guarantees equal team changes. Rating 800 starts placements near the Trailblazer I/Navigator boundary without awarding a visible rank prematurely.

**Alternatives considered**: Glicko-2/TrueSkill (better uncertainty modeling but harder to explain and would add inputs beyond the v1 contract); individual performance modifiers (encourage selfish team play); fixed win/loss points (ignore opponent strength).

## 7. Rank Mapping And World Legend

**Decision**: Ratings map in 100-point divisions from Scout III at 0 through Geo Master II at 1900–1999; Geo Master I begins at 2000 and has no upper bound. World Legend is computed from the active season's ordered eligible standings: rating descending, wins descending, rating-reached time ascending, user ID ascending only as a deterministic final key. Exactly positions 1–500 receive it when 500 qualify.

**Rationale**: Fixed thresholds make every standard promotion predictable, while read-time top-500 membership prevents stale stored badges. The extra user-ID key never changes the specified competitive tie-break unless all visible facts are identical.

**Alternatives considered**: A stored top-500 flag updated asynchronously (stale and race-prone); percentile ranks (not stable or transparent); separate top 500 per format (would grant the eighth rank to more than 500 users globally).

## 8. Seasons And Reset

**Decision**: Seasons last 84 days by default and roll over under a PostgreSQL advisory lock. At closure, active standings freeze with final position, ending rank, and peak rank. The next season requires five placements again and starts at `round(800 + 0.5 × (previous_rating - 800))`, floored at zero. Configuration exposes duration and fixed formula constants, and the client shows the rule/end time before ranked queue entry.

**Rationale**: A 50% soft reset compresses extremes without erasing skill, while immutable per-season rows preserve history and top-500 snapshots.

**Alternatives considered**: Hard reset to 800 (too destructive); no reset (stale leaderboard); manual-only rollover (operationally fragile).

## 9. Realtime Transport And Recovery

**Decision**: Extend the backend WebSocket transport with party and match channels, one-time 30-second connection tickets bound to user/channel, targeted audiences, bounded outbound queues, monotonic Redis versions, and reference-counted Redis Pub/Sub subscriptions per locally active channel. The browser sends the ticket as a `ticket.{opaque}` WebSocket subprotocol alongside `geoguess.v1`, never in the URL. Slow consumers are closed and recover from an HTTP snapshot; clients reconnect with exponential backoff plus jitter and fetch a new ticket.

**Rationale**: Auth cookies are scoped to the Next BFF origin, so direct browser WebSockets need an opaque one-use credential. Targeted fanout is required for team privacy and multi-instance delivery; versioned snapshots repair missed events without durable storage for every camera movement.

**Alternatives considered**: Query-string access JWTs (long-lived secret leakage); Next.js WebSocket proxy (not a portable Route Handler capability); polling at 1 second (excess load and poor match latency); broadcast-to-all with client filtering (privacy failure).

## 10. Markers And Spectating

**Decision**: Store the current proposed marker and current panorama/image view state in Redis with round-bounded TTLs. Marker updates are limited to 2/s; view updates to 4/s and only materially changed state. The server computes recipients: markers are team-only; submitted Solo players may receive only the unsubmitted opponent's scene state; submitted team players may receive only active teammates' scene state. Map/cursor/guess state is never part of a view event.

**Rationale**: These are disposable collaboration hints, not history. Server-side audience selection enforces the information boundary even with a modified client.

**Alternatives considered**: Persist every update (unbounded writes); WebRTC screen sharing (large security/operational scope); client-side recipient filtering (not secure).

## 11. Chat, Attachments, And Retention

**Decision**: Persist team text messages and moderation facts in PostgreSQL; paginate by sequence; publish compact events after commit. Text is 500 characters, limited to 10 messages/10 seconds and 60/minute. Team-chat uploads are purpose- and match-bound, max 5 MB, JPEG/PNG/WebP only. On completion the server verifies magic bytes and dimensions (≤20 MP), decodes, resizes the longest edge to ≤2048 px, re-encodes as JPEG, stores the sanitized private derivative, and deletes the raw object before it can be attached. Add `golang.org/x/image` for WebP decode/resize. This is technical file sanitization, not automated semantic/NSFW moderation.

**Rationale**: Serving only a decoded/re-encoded derivative removes active metadata/polyglot risk and does not require a new external moderation vendor. Mute/report and authorized review cover harmful content in v1.

**Alternatives considered**: Serve original files (unsafe); accept remote URLs (tracking/content substitution); external moderation API (new vendor, policy, latency, and cost not requested); disallow images (does not meet the feature).

## 12. Chat Access And Cleanup

**Decision**: Normal message/history/attachment access is allowed during the match and for a fixed 15-minute result session after terminal completion; the client does not expose a chat-history route. Backend retains unreported chat and sanitized images for 30 days after match completion. A report retains the referenced message/image for 180 days from the report or until an explicit legal hold ends. Rejected/pending raw uploads are deleted within 24 hours. A graceful 15-minute cleanup worker deletes expired rows and objects in bounded batches.

**Rationale**: Short-lived normal access matches the requested in-game collaboration while a bounded moderation window supports abuse handling and storage control.

**Alternatives considered**: Permanent history (privacy/storage burden); immediate deletion (no report investigation); client-only mute/report state (not enforceable/recoverable).

## 13. Frontend Data Flow

**Decision**: Use localized Server Components for initial play, match, and competitive reads via `server-only` uncached fetch modules. Use Server Actions for party/queue commands and same-origin Route Handlers for latency-sensitive guesses, chat, upload completion, and realtime-ticket issuance. Hydrate a focused match Client Component for Google Maps, authoritative countdown rendering, WebSocket state, chat, markers, and spectating. Extract reusable map/panorama primitives from the daily mission feature instead of copying them. Do not store canonical server state in Zustand.

**Rationale**: This follows the installed Next.js 16 guidance, keeps secrets/cookies server-side, minimizes initial JavaScript, and still supports browser-only APIs.

**Alternatives considered**: Fully client-rendered application (larger bundle and duplicated auth); Server Actions for high-frequency marker/view traffic (wrong transport); new client server-state library (unnecessary).

## 14. Rollout And Failure Policy

**Decision**: Ship schema and dormant services first, then enable Casual, Ranked, and team-chat images independently. PostgreSQL wins over Redis after formation; Redis failure blocks new queues/live collaboration but not durable results. Storage failure disables image actions while text remains available. Season/config invalidity disables Ranked and fails startup validation when Ranked is enabled. Rollback uses feature flags and prior code compatibility; migration down is not used after competitive/chat data exists.

**Rationale**: Independent flags limit blast radius and preserve completed match/rating history during dependency failures.

**Alternatives considered**: All-at-once cutover (high operational risk); destructive rollback (data loss); make R2 a global readiness dependency (unnecessarily disables non-image play).
