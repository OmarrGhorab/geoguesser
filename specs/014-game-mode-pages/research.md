# Research: Game Mode Pages

## Decision 1 — Keep eleven canonical product choices

**Decision**: The selectable values are solo, practice, daily, quick_play,
party_lobby, casual_solo, casual_duo, casual_squad, ranked_solo, ranked_duo, and
ranked_squad. private_room, ranked_standard, and ranked are accepted only when
recovering existing records.

**Rationale**: This exactly matches the navigation surface and existing
six-mode matchmaking vocabulary while preventing legacy aliases from becoming
duplicate products.

**Alternatives considered**: Show every database enum; use only four umbrella
pages. Both obscure distinct rules and create ambiguous analytics and URLs.

## Decision 2 — Use one capability-driven gameplay shell

**Decision**: Extract the Daily map, Street View, guess, timer, reveal, controls,
and result primitives into a shared gameplay feature. Domain adapters provide
capabilities and authoritative state for solo games, Practice, Daily, rooms, and
matches.

**Rationale**: All modes share presentation mechanics but differ in timing,
reveal, team, progression, host, and continuation policy. Capabilities preserve
those differences without copying the existing large Daily component.

**Alternatives considered**: Copy Daily per mode; create one giant component
with a mode switch. Copying guarantees drift, while a giant switch couples
unrelated domain lifecycles.

## Decision 3 — Make Quick Play a backend-owned canonical mode

**Decision**: Add a GameModeQuickPlay runtime constant and an idempotent
POST /games/quick-play operation. The server selects QUICK_PLAY_DEFAULT_MAP_ID,
uses five rounds and a 60-second timer, creates and starts the game atomically,
persists quick_play, and returns the resolved configuration.

**Rationale**: quick_play is already admitted by PostgreSQL and OpenAPI but the
service rejects it. A server-owned operation makes one-click behavior
deterministic, validates map eligibility centrally, and preserves canonical mode
analytics.

**Alternatives considered**: Treat it as Casual Solo matchmaking; send a hidden
Solo form preset. Matchmaking changes the session/eligibility model, while a
Solo alias violates the one-choice-to-one-canonical-mode success criterion.

## Decision 4 — Add durable creation receipts

**Decision**: Add a bounded PostgreSQL command_receipts record for game, Quick
Play, and room creation. Scope, actor fingerprint, idempotency key hash, request
hash, resource kind/id, and expiry are stored with uniqueness on
scope/actor/key. Raw guest identifiers and raw idempotency keys are not stored.

**Rationale**: The UI must survive retry, refresh, and double-click without
creating duplicate durable resources. Existing Redis claims and natural state
idempotency are insufficient for creation after cache loss.

**Alternatives considered**: Client-side button disabling only; Redis-only
claims; unrelated nullable columns on games and rooms. None provides one durable
cross-domain retry rule as cleanly.

## Decision 5 — Repair contract drift before page consumption

**Decision**: Narrow ordinary CreateGameRequest to solo and practice, document
Quick Play separately, add party_version to team matchmaking, add Party Lobby
self-leave and shared-round results, and only advertise idempotency behavior that
the service actually enforces.

**Rationale**: Generated or hand-written clients cannot safely integrate against
the current over-broad mode enum and missing required party version. Contract
truth must precede UI work.

**Alternatives considered**: Encode undocumented request properties and
implementation quirks in the frontend. That creates brittle coupling and fails
contract tests.

## Decision 6 — Keep backend domains authoritative

**Decision**: games owns solo/practice/Quick Play round truth; challenges owns
Daily attempt identity and five-of-five progress; rooms owns Party Lobby;
parties owns premade Duo/Squad rosters; matchmaking owns queue/assignment;
matchplay owns active and terminal match snapshots; competitive owns Ranked
progression.

**Rationale**: These boundaries and durable models already exist and have
separate authorization, privacy, and concurrency semantics.

**Alternatives considered**: A new aggregate game-modes backend package. It
would duplicate domain rules and create a second authority.

## Decision 7 — Use explicit localized route families

**Decision**: Use localized play, games, rooms, parties, matchmaking, and matches
families. Keep Daily Mission routes canonical. A generic game route reads the
game first and redirects legacy/daily/multiplayer records to the correct
semantic destination.

**Rationale**: Stable deep links enable refresh recovery, while semantic
families keep setup, lobby, queue, active play, and results understandable.

**Alternatives considered**: One /play page driven entirely by query strings;
one URL per label. The first overloads lifecycle state and the second duplicates
shared flows.

## Decision 8 — Keep pages server-first with narrow client islands

**Decision**: Page and layout modules remain Server Components. Initial
personalized reads use no-store native fetch through server-only modules.
Server Actions handle low-frequency create/join/start/ready/leave/end commands.
Same-origin Route Handlers bridge repeated guesses, chat, upload, and ticket
requests. Maps, timers, WebSockets, dialogs, and chat are leaf Client Components.

**Rationale**: This follows installed Next.js 16 guidance, keeps cookies and CSRF
handling server-side, reduces hydration, and avoids exposing internal backend
access.

**Alternatives considered**: A client-rendered SPA; calling the Go API directly
from Server Components through internal Next Route Handlers. Both add network or
bundle cost and blur security boundaries.

## Decision 9 — Treat realtime events as versioned hints

**Decision**: Hydrate from an authorized HTTP snapshot, apply only the next event
version, ignore duplicate/old events, and refetch after a gap, reconnect, or
degraded channel. Match/party sockets use one-time 30-second subprotocol tickets;
room sockets use validated origins and guest/registered cookie sessions until a
guest-capable ticket path is introduced.

**Rationale**: PostgreSQL is durable truth and Redis/WebSockets are disposable
coordination. Snapshot repair prevents stale UI and hidden-data inference.

**Alternatives considered**: Trust event streams as canonical; put tickets in
query strings. Both weaken recovery or credential safety.

## Decision 10 — Preserve the current Daily five-game program

**Decision**: The page displays games played out of five, resumes the active
attempt, offers the next attempt after completion when fewer than five are
played, and shows the day-complete state after the fifth game.

**Rationale**: DailyGamesPerDay is currently five and metadata already exposes
played/total. This avoids an unrelated backend product-policy change.

**Alternatives considered**: Change the backend to one game per day. That would
alter progression, streak, mission, and existing-user behavior beyond this
navigation/page feature.

## Decision 11 — Separate Party Lobby from premade matchmaking parties

**Decision**: Party Lobby uses room codes, supports guest/account players and
2–50 player free-for-all. Matchmaking parties are registered-only exact Duo or
Squad rosters with invites, readiness, leader control, and optimistic versioning.

**Rationale**: The backend models, permissions, realtime channels, scoring, and
capacity rules are intentionally different despite the shared word party.

**Alternatives considered**: Reuse rooms as matchmaking parties. This would
break team-size, eligibility, rating, and queue invariants.

## Decision 12 — Roll out capabilities, not only pages

**Decision**: Deploy contract/recovery backend work first, then shared gameplay,
solo/Practice/Daily/Quick Play, Party Lobby, Casual Solo, Ranked Solo, and team
modes. CASUAL_MATCHMAKING_ENABLED and RANKED_TEAM_MODES_ENABLED remain rollout
controls; TEAM_CHAT_IMAGES_ENABLED stays independent.

**Rationale**: Existing six-mode backend support is disabled by default and
active durable sessions must remain recoverable even when new entry is disabled.

**Alternatives considered**: Enable all eleven entries in one release. That
increases blast radius and makes queue/realtime regressions difficult to isolate.

## Existing evidence carried forward

- Solo/Daily operations: below 1 second p95.
- Practice and 50-player Party Lobby snapshot/command target: below 300 ms p95.
- Matchmaking join/leave/status: below 1 second p95; assignment visible below
  3 seconds; candidate scans remain bounded.
- Guess acceptance: below 500 ms p95.
- Authorized fanout: below 250 ms; visible UI update within 1 second.
- Practice history: cursor default 20, maximum 100; no unbounded round loading.
- PostgreSQL wins after durable game/match creation; Redis loss must not lose an
  assignment or terminal result.
- Logs use bounded mode, operation, outcome, and degraded-realtime labels only.
  They exclude tokens, room codes, tickets, Redis keys, identities, chat bodies,
  guesses, precise coordinates, and hidden answers.
