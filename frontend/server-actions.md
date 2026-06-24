# Server Actions

## Concepts

Server Actions are async Server Functions used for app mutations. They run on the server, can be called from forms or Client Components, receive `FormData` naturally, support progressive enhancement, and integrate with revalidation so Next.js can return updated UI and data in one roundtrip.

Why this exists: in-app writes should stay close to server data, auth, validation, and cache invalidation without requiring a hand-built API endpoint.

## Best Practices

- Prefer Server Actions over Route Handlers for mutations used only by the app UI.
- Put reusable actions in files with top-level `'use server'`.
- Validate with Zod inside the action.
- Authorize inside every action.
- Return serializable state for form errors.
- Use `updateTag` for read-your-own-writes.
- Use `revalidateTag(tag, 'max')` when stale-while-revalidate is acceptable.
- Call `redirect()` after invalidation when navigation is part of the mutation.

## Anti-Patterns

- Treating actions as private because only your form imports them.
- Mutating data from Client Components with `fetch('/api/...')` for internal forms.
- Throwing raw validation errors into the UI.
- Using `router.refresh()` instead of invalidating server caches.
- Binding sensitive identifiers without checking ownership inside the action.
- Storing form state in Zustand when `useActionState` is enough.

## Common Mistakes

- Forgetting `'use server'`.
- Defining a Server Action inside a Client Component.
- Returning non-serializable values.
- Calling `redirect()` before cache invalidation.
- Using `revalidateTag` when the submitting user must immediately see fresh data.
- Forgetting CSRF and origin considerations for sensitive actions.

## Production Examples

```ts
// features/projects/actions.ts
'use server'

import { updateTag } from 'next/cache'
import { redirect } from 'next/navigation'
import { z } from 'zod'
import { assertCanCreateProject } from '@/lib/authz'
import { db } from '@/lib/db'

const schema = z.object({
  name: z.string().trim().min(2).max(80),
})

export type CreateProjectState = {
  ok: boolean
  errors: Record<string, string[] | undefined>
}

export async function createProject(
  _state: CreateProjectState,
  formData: FormData
): Promise<CreateProjectState> {
  await assertCanCreateProject()

  const parsed = schema.safeParse({ name: formData.get('name') })
  if (!parsed.success) {
    return { ok: false, errors: parsed.error.flatten().fieldErrors }
  }

  const project = await db.project.create({ data: parsed.data })
  updateTag('projects')
  redirect(`/projects/${project.id}`)
}
```

```tsx
// features/projects/components/create-project-form.tsx
'use client'

import { useActionState } from 'react'
import { createProject } from '../actions'

const initialState = { ok: false, errors: {} }

export function CreateProjectForm() {
  const [state, action, pending] = useActionState(createProject, initialState)

  return (
    <form action={action}>
      <label htmlFor="project-name">Project name</label>
      <input id="project-name" name="name" required />
      {state.errors.name ? <p role="alert">{state.errors.name[0]}</p> : null}
      <button disabled={pending}>{pending ? 'Creating...' : 'Create'}</button>
    </form>
  )
}
```

## Folder Organization

```text
features/projects/
  actions.ts
  schemas.ts
  components/
    create-project-form.tsx
  data.ts
```

Keep action schemas near the mutation. Share schemas only when the same trust boundary truly repeats.

## TypeScript Examples

```ts
export type ActionState<TFields extends string> = {
  ok: boolean
  message?: string
  errors: Partial<Record<TFields, string[]>>
}

export type CreateProjectFields = 'name' | 'description'
export type CreateProjectState = ActionState<CreateProjectFields>
```

## Performance Considerations

- Server Actions can update UI and data in a single server roundtrip.
- Keep actions focused; do parallel work inside one action when needed.
- Invalidate precise tags to avoid broad rerenders.
- Avoid long-running work in the request path; enqueue background work when appropriate.
- Return small state payloads.

## Security Considerations

- Authenticate and authorize inside each action.
- Validate all input with Zod before mutation.
- Use server-side session cookies, not client tokens.
- Check resource ownership and tenant boundaries.
- Avoid leaking detailed authorization failures to unauthorized users.
- Rate limit expensive or abuse-prone actions.

## Accessibility Considerations

- Prefer real `<form>` elements for progressive enhancement.
- Associate labels with inputs.
- Use `role="alert"` or `aria-live` for validation errors.
- Preserve focus after errors and redirects where possible.
- Disable submit buttons only while pending and keep status text available.

