# Validation

## Concepts

Use Zod for runtime validation at every trust boundary: forms, Server Actions, Route Handlers, search params, route params, cookies, headers, environment variables, and third-party API responses. TypeScript describes expected types; Zod proves data at runtime.

Why this exists: users, browsers, external services, and URLs can send malformed or malicious data.

## Best Practices

- Parse unknown input with Zod before use.
- Use `safeParse` for user-correctable errors.
- Use `parse` for invariant internal data that should fail fast.
- Keep schemas close to the boundary.
- Derive TypeScript types with `z.infer`.
- Use coercion intentionally for `FormData` and URL values.
- Return flattened field errors to forms.

## Anti-Patterns

- Trusting `FormData.get()` or `request.json()` directly.
- Validating only in React Hook Form.
- Using type assertions instead of parsing.
- Creating one enormous global schema file.
- Returning raw Zod issues to end users.
- Hardcoding validation messages outside localization.

## Common Mistakes

- Forgetting `FormData.get()` can return `File` or `null`.
- Using `z.coerce.number()` without range checks.
- Treating empty strings as absent without preprocessing.
- Not validating third-party API responses.
- Reusing create schemas for update schemas without considering partial fields.
- Not mapping errors to localized UI text.

## Production Examples

```ts
import { z } from 'zod'

export const createTeamSchema = z.object({
  name: z.string().trim().min(2).max(80),
  slug: z
    .string()
    .trim()
    .min(3)
    .max(40)
    .regex(/^[a-z0-9-]+$/),
})

export type CreateTeamInput = z.infer<typeof createTeamSchema>
```

```ts
const parsed = createTeamSchema.safeParse({
  name: formData.get('name'),
  slug: formData.get('slug'),
})

if (!parsed.success) {
  return {
    ok: false,
    errors: parsed.error.flatten().fieldErrors,
  }
}
```

## Folder Organization

```text
features/teams/
  schemas.ts
  actions.ts
  route-schemas.ts
lib/env.ts
```

Use `lib/env.ts` for environment validation. Keep feature schemas with features.

## TypeScript Examples

```ts
const searchSchema = z.object({
  query: z.string().trim().optional().default(''),
  page: z.coerce.number().int().min(1).default(1),
})

type SearchParams = z.infer<typeof searchSchema>
```

```ts
export function parseSearchParams(input: Record<string, string | string[] | undefined>): SearchParams {
  return searchSchema.parse(input)
}
```

## Performance Considerations

- Parse once at the boundary, then pass typed data inward.
- Avoid parsing the same payload repeatedly in nested functions.
- Keep client schemas limited to forms that need client validation.
- Prefer server validation for large schemas to reduce JS.
- Use schema composition to avoid duplicated validation code.

## Security Considerations

- Validation is not authorization; do both.
- Validate file metadata and content server-side.
- Validate redirect URLs to prevent open redirects.
- Validate webhook signatures before trusting payloads.
- Never expose detailed internal validation rules that aid abuse.

## Accessibility Considerations

- Convert validation failures into field-specific, localized messages.
- Link errors with `aria-describedby`.
- Use `role="alert"` or `aria-live` for submit errors.
- Preserve user input after validation failure.
- Do not rely on color alone to identify invalid fields.

