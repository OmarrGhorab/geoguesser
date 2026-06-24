# Coding Standards

## Concepts

Coding standards keep the codebase predictable across server/client boundaries, UI patterns, validation, localization, and performance. The standard is strict because Next.js apps are full-stack systems where small choices can affect security and user experience.

Why this exists: consistency lets teams review faster and lets AI agents generate code that fits production expectations.

## Best Practices

- Use TypeScript, ESLint, Prettier, and pnpm.
- Prefer named exports for feature components and utilities.
- Keep functions small and purpose-specific.
- Use async Server Components for reads.
- Use Server Actions for app writes.
- Use native `fetch()`.
- Localize every user-facing string.
- Keep comments rare and useful.
- Preserve existing repo conventions when they do not violate this skill.

## Anti-Patterns

- JavaScript files.
- Axios, Redux Toolkit, React Query, React Router, Pages Router.
- Hardcoded visible strings.
- Client Components by default.
- Catching errors only to log and continue unsafely.
- Premature abstraction.
- Inconsistent formatting or package managers.

## Common Mistakes

- Ignoring lint warnings that point to hook or accessibility bugs.
- Mixing server and client exports in one file.
- Creating generic `utils.ts` catchalls.
- Using `any` for action state.
- Adding dependencies for small platform features.
- Editing generated shadcn code for product-specific styling.

## Production Examples

```ts
// Good: explicit server-only data function
import 'server-only'

export async function getCurrentTeam() {
  const userId = await requireUserId()
  return db.team.findFirst({ where: { members: { some: { userId } } } })
}
```

```tsx
// Good: small client boundary
'use client'

export function DisclosureButton({
  expanded,
  onToggle,
}: {
  expanded: boolean
  onToggle: () => void
}) {
  return (
    <button type="button" aria-expanded={expanded} onClick={onToggle}>
      Details
    </button>
  )
}
```

## Folder Organization

```text
features/*/
  actions.ts
  data.ts
  schemas.ts
  components/
components/ui/
lib/
stores/
```

File names should be lowercase kebab-case except framework conventions such as `page.tsx`.

## TypeScript Examples

```ts
type Result<TData, TError extends string> =
  | { ok: true; data: TData }
  | { ok: false; error: TError }
```

```ts
export function assertNever(value: never): never {
  throw new Error(`Unexpected value: ${value}`)
}
```

## Performance Considerations

- Keep imports narrow.
- Do not make broad providers or wrappers client-side without need.
- Avoid unnecessary memoization; measure first.
- Use stable dimensions for UI elements.
- Prefer server-side formatting and rendering where possible.

## Security Considerations

- Default to server-only for privileged modules.
- Validate and authorize before mutation.
- Avoid logging sensitive values.
- Do not expose internal errors to users.
- Review dependency additions carefully.

## Accessibility Considerations

- Component APIs should require labels for icon-only controls.
- Preserve focus styles.
- Use semantic HTML.
- Include loading, empty, error, disabled, and success states.
- Do not bury accessibility props behind generic spread patterns that can be overwritten accidentally.

