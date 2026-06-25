# Phase 3 - Maps Locations And Media

Goal: expose playable map and location media data without leaking hidden coordinates.

## Scope

- `internal/maps`
- `internal/locations`
- map listing and map detail reads
- location media resolution
- location selection primitives for gameplay
- access-tier enforcement for free versus protected content

## APIs

- `GET /api/v1/maps`
- `GET /api/v1/maps/{mapId}`
- `GET /api/v1/locations/{locationId}/media`

## Durable Data

- `maps`
- `locations`
- `map_locations`

## Rules

- Map responses expose public metadata only.
- Location coordinates stay server-side until reveal.
- Location selection must not use `ORDER BY random()` at scale.
- Only active locations in active maps are selectable.

## Design Sources

- `docs/phase-3-database-design.md`
- `docs/phase-4-api-design.md`
- `docs/phase-5-backend-architecture.md`
- `docs/phase-8-technical-specifications.md` feature 12

## Done When

- Public maps can be listed and inspected.
- Media DTOs exclude coordinate-leaking provider metadata before reveal.
- Selection helpers are ready for solo and room game creation.

## Dependencies

- Phase 1
- Phase 2 for optional-session media access
