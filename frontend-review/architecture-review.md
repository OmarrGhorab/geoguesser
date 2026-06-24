# Architecture Review

## Review Scope

Review feature organization, separation of concerns, server/client boundaries, business logic placement, abstractions, coupling, cohesion, and scalability.

## Blockers

- Business logic embedded in JSX or Client Components when it belongs on the server.
- App-wide Client Component shells without strong need.
- Internal Route Handlers called from Server Components.
- Feature behavior spread across unrelated folders with no ownership.
- Shared utilities that mix server-only and client-safe code.
- Abstractions that hide authorization, validation, or data ownership.

## What To Check

- `app/` contains route composition, not product logic.
- `features/*` owns actions, data, schemas, types, and feature UI.
- Server-only code is protected with `import 'server-only'`.
- Client Components are leaf-oriented and interactive only.
- Cross-feature dependencies are intentional and one-directional.
- DTOs prevent raw database records from leaking into UI.
- Abstractions have at least two real use cases and do not hide risk.

## Severity Guidance

- `High`: wrong boundary, business logic in client, broad coupling, or architecture that creates security/performance risk.
- `Medium`: unclear ownership, unnecessary abstraction, moderate duplication.
- `Low`: naming or folder polish that affects navigation but not behavior.

## Example Finding

```text
High - features/billing/components/billing-panel.tsx:1
The entire billing panel is marked "use client" even though only the plan selector needs browser state.
Why it matters: Pricing, invoice data, and entitlement logic now hydrate on the client and can pull server-only dependencies into the client graph.
Recommended fix: Keep the panel as a Server Component and extract a small PlanSelector client island.
```

