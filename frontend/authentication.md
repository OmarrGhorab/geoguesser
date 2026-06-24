# Authentication

## Concepts

Authentication verifies identity, session management persists that identity across requests, and authorization decides access. In Next.js, authentication work belongs on the server: Server Actions for login/signup forms, HTTP-only cookies for sessions, and server-side session verification for protected data.

Why this exists: auth is security-critical and must not depend on client state.

## Best Practices

- Prefer a mature auth/session library unless requirements justify custom auth.
- Use Server Actions for login, signup, logout, and session-changing forms.
- Store sessions in HTTP-only, secure, same-site cookies.
- Keep tokens out of localStorage, Zustand, and client props.
- Validate credentials on the server.
- Re-render UI after cookie changes through Server Actions.
- Centralize session verification in server-only modules.

## Anti-Patterns

- Storing JWTs in localStorage.
- Storing auth tokens or user permissions in Zustand.
- Fetching `/api/auth/me` in `useEffect` for server-renderable auth state.
- Relying on Client Component redirects for protection.
- Rolling custom password auth casually.
- Treating layouts as the only auth gate.

## Common Mistakes

- Exposing full session objects to Client Components.
- Forgetting logout must clear cookies server-side.
- Failing to rotate or expire sessions.
- Not handling session invalidation after password changes.
- Rendering authenticated shell before verifying session.
- Making auth checks in Proxy only and skipping actions/handlers.

## Production Examples

```ts
// lib/auth/session.ts
import 'server-only'
import { cookies } from 'next/headers'

export type Session = {
  userId: string
  role: 'member' | 'admin'
}

export async function getSession(): Promise<Session | null> {
  const token = (await cookies()).get('session')?.value
  if (!token) return null
  return verifySessionToken(token)
}
```

```ts
// features/auth/actions.ts
'use server'

import { cookies } from 'next/headers'
import { redirect } from 'next/navigation'
import { z } from 'zod'

const loginSchema = z.object({
  email: z.string().email(),
  password: z.string().min(8),
})

export async function login(_state: unknown, formData: FormData) {
  const parsed = loginSchema.safeParse({
    email: formData.get('email'),
    password: formData.get('password'),
  })

  if (!parsed.success) return { ok: false }

  const sessionToken = await authenticateUser(parsed.data)
  ;(await cookies()).set('session', sessionToken, {
    httpOnly: true,
    secure: true,
    sameSite: 'lax',
    path: '/',
  })

  redirect('/dashboard')
}
```

## Folder Organization

```text
lib/auth/
  session.ts
  passwords.ts
features/auth/
  actions.ts
  schemas.ts
  components/login-form.tsx
```

Keep provider integration and session verification server-only.

## TypeScript Examples

```ts
export type PublicUser = {
  id: string
  name: string
  email: string
}

export function toPublicUser(user: { id: string; name: string; email: string }): PublicUser {
  return { id: user.id, name: user.name, email: user.email }
}
```

## Performance Considerations

- Verify session close to data access to avoid unnecessary work.
- Cache non-sensitive public user display data separately from session validity.
- Avoid client auth bootstrapping requests when the server can render state.
- Keep auth providers deep enough to avoid making the whole app client-side.
- Avoid blocking static marketing routes on auth checks.

## Security Considerations

- Use HTTP-only secure cookies.
- Hash passwords with strong current algorithms when managing credentials.
- Use CSRF defenses appropriate to cookie-based sessions.
- Rate limit login and signup attempts.
- Avoid leaking whether an email exists.
- Rotate sessions after privilege changes.

## Accessibility Considerations

- Login forms need labels, autocomplete attributes, and error summaries.
- Auth errors should be clear without exposing sensitive details.
- Redirects should land on pages with clear headings.
- MFA flows need keyboard-friendly inputs and recovery paths.
- Session timeout UI should be announced and not trap focus.

