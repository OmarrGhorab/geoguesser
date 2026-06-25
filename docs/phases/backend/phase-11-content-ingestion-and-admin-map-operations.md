# Phase 11 - Content Ingestion And Admin Map Operations

Goal: support controlled backend ingestion and management of playable content.

## Scope

- admin-side map and location import tooling
- content validation and de-duplication
- draft, publish, archive, and visibility controls where approved
- storage foundations for future file-backed imports or uploads

## Candidate Packages

- `internal/maps`
- `internal/locations`
- future `internal/platform/storage`

## Durable Data

- `maps`
- `locations`
- `map_locations`
- future audit records

## Rules

- Imported coordinates and provider references must be validated on the backend.
- Admin import routes must be authorized and audited.
- User-generated map publishing is not fully specified yet and should not be treated as MVP-ready backend scope.

## Design Sources

- `docs/phase-1-product-definition.md`
- `docs/phase-3-database-design.md`
- `docs/phase-5-backend-architecture.md`
- `docs/phase-8-technical-specifications.md` features 12 and 14

## Design Gaps To Resolve In Implementation

- The old "map maker" phase includes user publishing and moderation, but the current backend design only clearly supports admin import and future file storage foundations.
- CSV and JSON import contracts are not yet formalized in OpenAPI.

## Done When

- Backend import paths can create valid playable content safely.
- Invalid content is rejected with useful validation errors.
- Auditability exists for privileged content operations.

## Dependencies

- Phase 3
- Phase 12 if file uploads become part of the first implementation slice
