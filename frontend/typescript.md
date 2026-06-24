# TypeScript

## Concepts

TypeScript is mandatory. It documents contracts across Server Components, Client Components, Server Actions, Route Handlers, data modules, forms, and validation. Static types do not validate runtime input, so pair TypeScript with Zod at trust boundaries.

Why this exists: Next.js apps cross server, client, network, and form boundaries. Strong types make those boundaries explicit.

## Best Practices

- Use strict TypeScript.
- Prefer inferred types from Zod schemas for validated data.
- Export DTO types, not database model types, to UI modules.
- Type route params and search params as promises.
- Use discriminated unions for action states.
- Use `satisfies` for config and constant maps.
- Avoid `any`; use `unknown` until validation.
- Keep shared types small and owned by features.

## Anti-Patterns

- JavaScript files in production app code.
- `as any` to silence framework or schema errors.
- Reusing ORM types as public API.
- Type-only validation without runtime parsing.
- Broad global types that hide feature ownership.
- Optional fields everywhere instead of modeling states.

## Common Mistakes

- Confusing `z.input` and `z.output` when using coercion.
- Typing `FormData.get()` as string without checking null.
- Returning non-serializable types to Client Components.
- Forgetting `import type` for type-only imports where helpful.
- Overusing generics where a concrete type is clearer.
- Not typing action state, which weakens form error handling.

## Production Examples

```ts
import { z } from 'zod'

export const userSettingsSchema = z.object({
  displayName: z.string().trim().min(2).max(80),
  locale: z.enum(['en', 'ar']),
  marketingEmails: z.coerce.boolean(),
})

export type UserSettingsInput = z.input<typeof userSettingsSchema>
export type UserSettings = z.output<typeof userSettingsSchema>
```

```ts
type SaveState =
  | { status: 'idle'; errors: Record<string, never> }
  | { status: 'error'; errors: Record<string, string[]> }
  | { status: 'success'; errors: Record<string, never> }
```

## Folder Organization

```text
features/settings/
  schemas.ts
  types.ts
  actions.ts
  data.ts
```

Put schema-derived types beside schemas. Put UI-only prop types beside components unless reused.

## TypeScript Examples

```tsx
type PageProps = {
  params: Promise<{ locale: string; id: string }>
  searchParams: Promise<{ mode?: 'view' | 'edit' }>
}

export default async function Page({ params, searchParams }: PageProps) {
  const [{ id }, { mode = 'view' }] = await Promise.all([params, searchParams])
  return <main data-mode={mode}>{id}</main>
}
```

```ts
const locales = ['en', 'ar'] as const
export type Locale = (typeof locales)[number]
```

## Performance Considerations

- Strong DTO types prevent overfetching and over-serialization.
- Avoid large shared type barrels that encourage accidental imports.
- Keep type generation in CI so route and environment errors fail early.
- Use narrow prop types to keep Client Component payloads small.
- Avoid runtime schema parsing in hot loops unless input is untrusted.

## Security Considerations

- Treat external data as `unknown`.
- Use Zod for form data, search params, cookies, headers, and third-party responses.
- Never let type assertions replace auth or validation.
- Keep secret-bearing types in server-only modules.
- Make unsafe escape hatches visible and reviewed.

## Accessibility Considerations

- Type form field names and error maps so labels and errors stay connected.
- Type localized message keys where tooling supports it.
- Use explicit prop names for accessible labels and descriptions.
- Model disabled, loading, and error states instead of boolean soup.
- Make component APIs require accessible names for icon-only controls.

