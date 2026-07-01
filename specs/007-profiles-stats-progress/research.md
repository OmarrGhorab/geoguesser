# Research: Profiles Stats Progress

## Decision: Use a dedicated profile domain for current-user profile management

**Rationale**: Architecture docs identify `profiles` as the owner for profile read/update flows while `users` owns registered account records and public stats lookup. A dedicated profile service keeps current-user authorization, profile validation, rate-limit behavior, and update DTO shaping separate from public user/stat reads.

**Alternatives considered**: Expanding `internal/users` to own current-profile mutation was simpler short term, but it would blur public user lookups with private current-account behavior and make privacy/security review harder.

## Decision: Keep public stats and game-history summaries privacy-safe and aggregate-only

**Rationale**: The feature spec requires public stats without private account data. Stats should derive from completed eligible game participation and expose only aggregate outcomes and public-safe history summaries. Hidden locations, guess coordinates, email, auth/session fields, and private preferences must never be returned from public stats/history endpoints.

**Alternatives considered**: Returning richer per-round or raw guess data would support more detailed history screens, but it risks answer spoilers and private data leakage. Per-round details can remain behind participant-authorized game result endpoints.

## Decision: Reuse existing durable gameplay facts, adding indexes only if query plans require them

**Rationale**: Existing tables already store registered users, user profiles, games, game players, rounds, and guesses. Phase 07 should not duplicate derived stats into new tables until materialization is needed for scale. Current needs can use bounded aggregate and cursor queries, with Goose migrations for missing indexes only.

**Alternatives considered**: Creating a persistent stats table would make reads faster, but it adds synchronization and double-counting risk before product scale proves the need.

## Decision: Use cursor pagination for account game history and saved progress summaries

**Rationale**: Game history grows over time. Cursor pagination using stable sort keys keeps reads bounded and avoids high-offset scans. The existing history direction already uses created-at and ID ordering; Phase 07 should preserve and harden that shape.

**Alternatives considered**: Offset pagination is easier for clients, but it performs poorly at volume and can duplicate/skip rows as history changes.

## Decision: Keep profile forms progressively enhanced and server-authorized

**Rationale**: Installed Next.js guidance favors Server Components for data fetching and Server Actions/forms for mutations, with authentication and authorization checked in server-side code. The profile route should load minimal safe DTOs on the server and use client-side code only for interactive form state where needed.

**Alternatives considered**: A fully client-fetched profile page would work but would add unnecessary browser JavaScript, duplicate loading logic, and increase the chance of exposing broader API data to client components.

## Decision: Use existing locale routing and message catalogs for profile UI

**Rationale**: The constitution requires English and Arabic parity. Existing app structure already uses localized routes and message catalogs, so profile and public stats copy should be added there and reuse established RTL direction handling.

**Alternatives considered**: Hardcoding profile copy in components would be faster initially but would violate localization requirements and create future translation debt.

## Decision: No new runtime readiness dependency

**Rationale**: Profile and public stats depend on PostgreSQL. Redis may be used only through existing rate-limit middleware. No new external service is required, and avatar upload/moderation remains outside this feature.

**Alternatives considered**: Adding object-storage-backed avatar upload now would expand the phase beyond profile/stats/progress and introduce storage, moderation, and signed URL concerns better handled by a separate media feature.
