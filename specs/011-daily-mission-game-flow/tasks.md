# Tasks: Daily Mission Game Flow

- [x] T001 Add regression tests for completed attempt resume and safe panorama
  media projection.
- [x] T002 Fix completed-attempt status preservation and add `panorama_id` to
  the current-round contract and OpenAPI.
- [x] T003 Add frontend mission/game Zod schemas, server-only reads, and
  CSRF/idempotency-aware Server Actions.
- [x] T004 Bind the calendar and Play/Resume/View Results states to current
  daily metadata.
- [x] T005 Add the localized game route with lazy Google Street View, guess map,
  five-round transitions, per-round result, and persisted final breakdown.
- [x] T006 Add English/Arabic copy and focused frontend tests.
- [x] T007 Run backend/frontend gates, rebuild Docker, and verify the live flow.
- [x] T008 Repair legacy unplayed attempts with missing media, replace the
  indefinite loader with a retry state, and migrate Google Maps loading and
  markers away from browser warnings.
- [x] T009 Replace opaque gameplay/result URLs with semantic localized routes
  and keep the old game route as a compatibility redirect.
- [x] T010 Build the map-first completed score and five-round breakdown UI from
  the existing owned game-results endpoint, including English, Arabic, RTL,
  loading/error fallbacks, metadata, and focused frontend tests.
- [x] T011 Replace the completed-today mission landing with the supplied split
  design, shared marked-map summary, real derived stats, localized copy, and
  mission artwork.
- [x] T012 Embed the reusable detailed Results/Map panel beneath the completed
  summary and make View Results scroll to it without another request.
- [x] T013 Refine the completed summary and detailed panel against the supplied
  design asset with its compact shell, themed route map, asymmetric content
  grid, segmented tabs, player row, and dense five-round score table.
