# Folder Review

## Review Scope

Ensure feature-based organization, consistent naming, small files, reusable modules, and clear boundaries.

## Blockers

- Route files that contain large product workflows.
- Server-only and client code mixed in shared barrels.
- Feature code scattered across generic folders with no ownership.
- Duplicate modules for the same concept.
- Banned Pages Router structure.

## What To Check

- `app/` owns routing and framework conventions.
- `features/*` owns domain actions, data, schemas, types, and components.
- `components/ui` contains shadcn primitives only.
- `stores/` contains UI-only global stores.
- File names are consistent and descriptive.
- Files remain small enough to review.
- Imports do not bypass feature boundaries.

## Severity Guidance

- `High`: structure causes server/client leaks, security bypasses, or duplicated ownership.
- `Medium`: maintainability or scalability concerns.
- `Low`: naming and local organization polish.

## Example Finding

```text
Medium - app/[locale]/settings/page.tsx
The route file contains data access, validation schema, form UI, and mutation logic.
Why it matters: The route is hard to review and reuse, and future settings changes will couple unrelated concerns.
Recommended fix: move schema/action/data to features/settings and keep page.tsx as route composition.
```

