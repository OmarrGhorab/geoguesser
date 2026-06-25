# GeoGuess Implementation Phases

This folder reorganizes the feature plan into delivery tracks instead of mixed product/frontend slices.

The source of truth for scope remains the root design set in `docs/`:

- `docs/phase-1-product-definition.md`
- `docs/phase-2-system-design.md`
- `docs/phase-3-database-design.md`
- `docs/phase-4-api-design.md`
- `docs/phase-5-backend-architecture.md`
- `docs/phase-6-frontend-architecture.md`
- `docs/phase-8-technical-specifications.md`
- `docs/phase-9-project-setup.md`

## Tracks

- [Backend phases](./backend/README.md)
- Frontend phases: deferred for now. Add after backend sequencing is accepted.

## Why This Split

The earlier `client/docs/phases/` plan is useful as a product progression list, but implementation now needs:

- Backend phases aligned with database, API, Go package, Redis, and worker boundaries.
- Frontend phases aligned later with Next.js route, Server Component, Client Component, and UX dependencies.
- Honest separation between features that are fully designed and features that still need more backend specification.

## Planning Rule

Backend phases should be implemented in this order unless a later phase is pulled forward intentionally with its missing design work filled in first.
