# Implementation Plan: Daily Mission Game Flow

## Summary

Complete the existing daily challenge pipeline rather than introducing a new
domain model. The daily mission landing page reads current metadata on the
server. A Server Action starts or resumes the existing challenge attempt and
redirects to a localized game route. A small Client Component owns Google
Street View, the unsent guess pin, round transitions, and result presentation;
all writes continue through Server Actions to the Go API.

## Constitution Check

- **Architecture**: PASS. Go challenge/game services remain authoritative;
  Next.js uses server-only data modules and Server Actions; only map interaction
  is client-side.
- **Testing**: PASS. Add backend regression coverage for completed-attempt
  status and panorama safety, frontend schema/action/component tests, then run
  all required gates.
- **UX/localization**: PASS. Play, Resume, View Results, loading, error,
  disabled-day, round-result, and final-result states are localized in `en` and
  `ar`.
- **Performance**: PASS. Landing uses one challenge metadata request. Gameplay
  loads Google Maps only on the game route. Each guess uses one write followed
  by bounded game/round or final-result reads. Target API p95 remains under
  300 ms on local fixtures; no polling is introduced.
- **Operations/security**: PASS. Existing registered-session, CSRF, rate limit,
  ownership, unique attempt, scoring, and spoiler protections remain in force.
  Current-round media exposes only the panorama identifier required for display,
  never answer coordinates.

## Data and Contract

No migration is required. Existing constraints provide one attempt per
`(challenge_id, user_id)` and one dated daily challenge. `RoundMedia` gains an
optional `panorama_id`; `url` becomes optional because panorama-backed rounds do
not require a public static URL. OpenAPI and Zod schemas are updated together.

## Complexity Tracking

No new daily-mission table or aggregate endpoint is added. The existing
challenge attempt and game APIs already own the required invariants. A small
Google Maps loader is used instead of adding another client dependency.

Legacy daily challenges can contain locations created before playable-media
selection was enforced. Start/resume repairs only an unplayed attempt by
resampling five playable locations; once any guess exists, repair is rejected
to preserve scoring integrity.

The frontend result redesign reuses `GET /games/{gameId}/results`, whose owned
completed-game projection already contains each guess and revealed answer. The
semantic routes are `/[locale]/daily-mission/play/[gameId]` and
`/[locale]/daily-mission/results/[gameId]`; the previous game URL redirects for
compatibility. The map is one lazy client island and adds no package dependency.
Result routes are private and explicitly non-indexable despite their readable
path. Target result shell render is under 100 ms excluding the existing API and
Google Maps network latency, with one result API request and one Maps loader.

After completion, the daily landing performs one additional owned game-results
read and renders a dedicated completed-today dashboard. The marked-map client
island is shared with the result route; the surrounding summary and promotional
artwork remain server-composed/static. No additional dependency or backend call
is introduced for active or not-yet-played users.

The reusable detailed Results/Map panel is rendered immediately beneath that
completed dashboard on the same daily-mission page. View Results uses an in-page
anchor, so opening the second section causes neither a route transition nor a
second result fetch. The semantic standalone result route remains as a compatible
wrapper around the same panel for direct links.

Historical day selection now performs one date-specific daily metadata read in
addition to the initial current-day read. A completed historical attempt reuses
the same owned result dashboard; a missing or unfinished attempt renders a
localized passed-challenge empty state and never exposes a playable CTA. The
existing `date` query contract is reused, so no backend contract or migration is
required. The empty state adds no client dependency, animation, or network call
beyond that bounded metadata read.

## Verification

- Backend: focused challenge/game/location tests, `go test ./...`, `go vet
  ./...`, build, targeted lint, OpenAPI validation.
- Frontend: focused mission tests, unit suite, lint, typecheck, production build.
- Runtime: rebuilt Docker API, authenticated start/resume, five guesses, and
  stored final result with five round marks; historical completed and unplayed
  dates render their result and passed states respectively.
