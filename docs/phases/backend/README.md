# Backend Implementation Phases

This track turns the product feature list into backend delivery phases based on the approved database, API, backend architecture, and technical specification documents.

## Recommended Order

1. [Phase 1 - Platform Foundation](./phase-01-platform-foundation.md)
2. [Phase 2 - Identity And Player Sessions](./phase-02-identity-and-player-sessions.md)
3. [Phase 3 - Maps Locations And Media](./phase-03-maps-locations-and-media.md)
4. [Phase 4 - Solo Game Loop](./phase-04-solo-game-loop.md)
5. [Phase 5 - Results History And Game Retrieval](./phase-05-results-history-and-game-retrieval.md)
6. [Phase 6 - Private Rooms Realtime And Reconnection](./phase-06-private-rooms-realtime-and-reconnection.md)
7. [Phase 7 - Profiles Stats And Persistent Progress](./phase-07-profiles-stats-and-persistent-progress.md)
8. [Phase 8 - Leaderboards Daily Seeds And Competitive Read Models](./phase-08-leaderboards-daily-seeds-and-competitive-read-models.md)
9. [Phase 9 - Matchmaking And Ranked Foundations](./phase-09-matchmaking-and-ranked-foundations.md)
10. [Phase 10 - Friends Social Graph And Access Controls](./phase-10-friends-social-graph-and-access-controls.md)
11. [Phase 11 - Content Ingestion And Admin Map Operations](./phase-11-content-ingestion-and-admin-map-operations.md)
12. [Phase 12 - Workers Observability Security And Future Extensions](./phase-12-workers-observability-security-and-future-extensions.md)

## Mapping Notes

- The old `client/docs/phases/` files remain a feature-oriented reference.
- This backend track follows actual backend ownership in `internal/*`, PostgreSQL, Redis, OpenAPI, and worker responsibilities.
- Several late-product ideas such as duels, streak modes, quiz modes, and full user-generated map publishing still need more backend specification than currently exists in the root docs. Those are called out as design gaps instead of being treated as implementation-ready.

## Delivery Rules

- Keep Go package boundaries from `docs/phase-5-backend-architecture.md`.
- Keep PostgreSQL as the durable source of truth and Redis for ephemeral coordination only.
- Update `backend/openapi/openapi.yaml` and migrations in the same phase that introduces the relevant behavior.
- Every phase must include tests at the closest useful level and preserve hidden-coordinate security rules.
