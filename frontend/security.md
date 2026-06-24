# Security

## Concepts

Security in Next.js frontend architecture means protecting data across server/client boundaries, authenticating sessions, authorizing operations, validating all untrusted input, controlling caches, and avoiding secret exposure. Server Components and Server Actions improve security only when used deliberately.

Why this exists: a frontend repo is often a full-stack app. UI code can expose data, mutate state, set cookies, and call privileged services.

## Best Practices

- Keep secrets server-only.
- Use `import 'server-only'` in privileged modules.
- Validate with Zod at trust boundaries.
- Authorize every action, handler, and sensitive read.
- Use HTTP-only secure cookies for sessions.
- Return DTOs to clients.
- Use security headers.
- Sanitize or avoid rendering HTML.
- Review cache scopes for private data.

## Anti-Patterns

- Auth tokens in localStorage, sessionStorage, Zustand, or client props.
- Trusting hidden form fields.
- Trusting client-side role checks.
- Exposing raw database rows.
- Rendering untrusted HTML.
- Caching personalized data as shared content.
- Calling internal APIs to hide server logic.

## Common Mistakes

- Forgetting Server Actions are reachable by POST.
- Checking auth but not authorization.
- Leaking environment variables with `NEXT_PUBLIC_`.
- Logging secrets or tokens.
- Returning distinct auth errors that enable account enumeration.
- Forgetting webhook signature verification.

## Production Examples

```ts
// lib/env.ts
import { z } from 'zod'

const envSchema = z.object({
  DATABASE_URL: z.string().url(),
  SESSION_SECRET: z.string().min(32),
})

export const env = envSchema.parse(process.env)
```

```ts
// lib/data-access.ts
import 'server-only'
import { getSession } from '@/lib/auth/session'

export async function requireUserId() {
  const session = await getSession()
  if (!session) throw new Error('Unauthorized')
  return session.userId
}
```

## Folder Organization

```text
lib/
  env.ts
  auth/
  authz/
  db/
features/*/
  actions.ts
  data.ts
```

Privileged modules belong outside client component graphs.

## TypeScript Examples

```ts
export type SafeError =
  | { code: 'unauthorized'; message: 'Unauthorized' }
  | { code: 'invalid_request'; message: 'Invalid request' }
  | { code: 'not_found'; message: 'Not found' }
```

```ts
export function toSafeError(error: unknown): SafeError {
  if (error instanceof ZodError) {
    return { code: 'invalid_request', message: 'Invalid request' }
  }
  return { code: 'not_found', message: 'Not found' }
}
```

## Performance Considerations

- Security checks should be indexed and cheap, but never skipped for speed.
- Validate once at boundaries, then pass trusted typed data.
- Avoid fetching sensitive records before authorization.
- Do not cache secrets or private data in shared caches.
- Use server-side direct calls instead of HTTP proxies for internal operations.

## Security Considerations

- Deny by default.
- Use least privilege for service tokens.
- Keep dependencies current.
- Rate limit abuse-prone actions and handlers.
- Use CSP where feasible.
- Audit every server/client boundary during review.

## Accessibility Considerations

- Security UX must remain accessible: login, MFA, password reset, and error recovery.
- Do not rely on visual-only warnings for destructive actions.
- Make session timeout warnings screen-reader accessible.
- Keep auth forms compatible with password managers.
- Avoid inaccessible CAPTCHA-only flows.

