# Forms

## Concepts

Forms should use native HTML and Server Actions whenever possible. React Hook Form is for client-side form interactivity that native forms cannot cover well, such as complex conditional fields, field arrays, and immediate client validation. Zod validates on the server regardless of client validation.

Why this exists: native forms plus Server Actions provide progressive enhancement, simpler mutations, less JavaScript, and reliable server validation.

## Best Practices

- Use `<form action={serverAction}>` for standard mutations.
- Use `useActionState` for pending state and server-returned errors.
- Use React Hook Form with Zod when forms require rich client interaction.
- Keep server validation authoritative.
- Localize labels, help text, placeholders, and errors with `next-intl`.
- Use semantic labels, fieldsets, and descriptions.
- Keep form actions idempotent or protected against double-submit where needed.

## Anti-Patterns

- Posting internal forms to Route Handlers by default.
- Managing every input in React state.
- Validating only on the client.
- Storing form state globally in Zustand.
- Hardcoding error messages.
- Disabling submit forever without accessible feedback.

## Common Mistakes

- Missing `name` attributes, so `FormData` is empty.
- Forgetting labels or relying on placeholder text.
- Returning error objects that do not match field names.
- Using React Hook Form for a two-field form that could be native.
- Not preserving values after server validation errors.
- Using hidden inputs for trusted data without server-side verification.

## Production Examples

```tsx
// Server Component form
import { saveProfile } from '@/features/profile/actions'

export function ProfileForm() {
  return (
    <form action={saveProfile}>
      <label htmlFor="display-name">Display name</label>
      <input id="display-name" name="displayName" required minLength={2} />
      <button type="submit">Save</button>
    </form>
  )
}
```

```tsx
// Client form with React Hook Form for complex interaction
'use client'

import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

const schema = z.object({
  email: z.string().email(),
  role: z.enum(['member', 'admin']),
})

type Values = z.infer<typeof schema>

export function InviteUserForm() {
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: { email: '', role: 'member' },
  })

  return (
    <form onSubmit={form.handleSubmit(() => form.reset())}>
      <label htmlFor="email">Email</label>
      <input id="email" {...form.register('email')} />
      {form.formState.errors.email ? (
        <p role="alert">{form.formState.errors.email.message}</p>
      ) : null}
      <button type="submit">Invite</button>
    </form>
  )
}
```

## Folder Organization

```text
features/profile/
  actions.ts
  schemas.ts
  components/
    profile-form.tsx
```

Keep schemas shared by server action and client form only when both need the same validation contract.

## TypeScript Examples

```ts
export type FieldErrors<TField extends string> = Partial<Record<TField, string[]>>

export type FormState<TField extends string> = {
  ok: boolean
  errors: FieldErrors<TField>
}
```

## Performance Considerations

- Native forms avoid unnecessary hydration.
- React Hook Form reduces rerenders compared with controlled inputs.
- Load complex form islands only where needed.
- Avoid large validation schemas in broad client bundles.
- Use server actions to avoid extra API client code.

## Security Considerations

- Validate and authorize on the server.
- Treat all form fields, including hidden fields, as untrusted.
- Protect sensitive actions against CSRF according to auth/session architecture.
- Rate limit expensive form submissions.
- Do not include secrets in default values or hidden inputs.

## Accessibility Considerations

- Every input needs a programmatic label.
- Error messages should be associated with fields and announced.
- Use `fieldset` and `legend` for grouped controls.
- Preserve keyboard submission behavior.
- Avoid placeholder-only instructions.

