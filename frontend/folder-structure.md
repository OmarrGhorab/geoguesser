# Folder Structure

## Concepts

Use App Router route files for routing and feature modules for product behavior. The folder structure should make server/client boundaries, ownership, localization, UI primitives, and data access obvious.

Why this exists: good structure prevents accidental client bundles, duplicated fetches, scattered mutations, and unclear ownership.

## Best Practices

- Keep `app/` route-focused.
- Use `features/` for product domains.
- Use `components/ui/` for shadcn primitives.
- Use `lib/` for cross-cutting server infrastructure.
- Use `stores/` only for global client UI state.
- Use `messages/` for next-intl catalogs.
- Keep tests near feature code where possible.

## Anti-Patterns

- Dumping everything into `components/`.
- Importing feature internals across unrelated features.
- Putting server data code in `app/api` just to call it internally.
- Creating one global `types.ts`.
- Mixing client stores with server data modules.
- Keeping generated shadcn components inside feature folders.

## Common Mistakes

- Barrel exports that hide server/client boundaries.
- Shared utilities that import server-only code and then get used by clients.
- Feature folders without schemas or actions, causing logic to drift.
- Route files becoming hundreds of lines long.
- Message files not aligned with feature namespaces.
- Test folders too far from the code under test.

## Production Examples

```text
app/
  [locale]/
    layout.tsx
    (app)/
      dashboard/page.tsx
      projects/[id]/page.tsx
    (marketing)/
      page.tsx
  api/
    webhooks/billing/route.ts
components/
  ui/
features/
  projects/
    actions.ts
    data.ts
    schemas.ts
    types.ts
    components/
lib/
  auth/
  authz/
  db/
  env.ts
stores/
messages/
tests/
```

## Folder Organization

Prefer this canonical shape unless the existing repo has a strong convention:

```text
features/{feature}/
  actions.ts
  data.ts
  schemas.ts
  types.ts
  components/
  stores/        # feature-local UI stores only
```

Keep `data.ts` server-only. Keep Client Components marked explicitly.

## TypeScript Examples

```ts
// features/projects/index.ts
export type { ProjectListItem } from './types'
export { getProjectList } from './data'
```

Avoid exporting Client Components and server functions from the same barrel if it risks boundary confusion.

## Performance Considerations

- Structure should prevent accidental heavy imports into client graphs.
- Feature-local code splitting follows routes naturally.
- Avoid broad barrels that pull large modules.
- Put expensive client widgets behind explicit client islands.
- Keep server-only data code out of UI primitives.

## Security Considerations

- Use `server-only` in `data.ts`, auth, authz, and db modules.
- Keep secrets under `lib` server modules, never `components`.
- Make Route Handlers transport layers, not the source of privileged logic.
- Avoid cross-feature imports that bypass policy helpers.
- Keep env validation centralized.

## Accessibility Considerations

- Shared UI primitives should encode accessible defaults.
- Feature components should own domain-specific labels and errors.
- Route folders should include loading/error/not-found states where users need them.
- Localization files should include aria and alt text.
- Tests should include accessibility coverage for shared primitives and critical flows.

