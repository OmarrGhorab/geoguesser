# Frontend Examples

## Concepts

These examples are copyable patterns for Next.js 16+ App Router projects. Treat them as starting points, then adapt names, schemas, auth policies, messages, and UI primitives to the target codebase.

Why this exists: examples make the skill operational. They show how the philosophy turns into code without falling back to generic React patterns.

## Best Practices

- Start from Server Components.
- Use Server Actions for app-only mutations.
- Use Route Handlers for external HTTP consumers.
- Validate with Zod at boundaries.
- Authorize on the server.
- Cache stable server data with tags.
- Use Zustand only for client UI state.
- Keep all visible strings localizable in real project code.

## Anti-Patterns

- Copying example strings into production without next-intl.
- Using examples as permission models without adapting auth.
- Turning examples into broad abstractions too early.
- Moving server examples into Client Components.
- Replacing Server Actions with app-only API routes.
- Storing fetched server data in Zustand.

## Common Mistakes

- Forgetting `name` attributes in forms.
- Forgetting `import 'server-only'` in data modules.
- Forgetting `updateTag` or `revalidateTag` after mutations.
- Trusting hidden input values.
- Using example cache tags without tenant or locale scoping where needed.
- Leaving example error messages unlocalized.

## Production Examples

### Server Action Form

```ts
// features/projects/actions.ts
'use server'

import { updateTag } from 'next/cache'
import { z } from 'zod'
import { assertCanCreateProject } from '@/lib/authz/projects'
import { db } from '@/lib/db'

const schema = z.object({
  name: z.string().trim().min(2).max(80),
})

export async function createProject(_state: unknown, formData: FormData) {
  await assertCanCreateProject()

  const parsed = schema.safeParse({ name: formData.get('name') })
  if (!parsed.success) {
    return { ok: false, errors: parsed.error.flatten().fieldErrors }
  }

  await db.project.create({ data: parsed.data })
  updateTag('projects')
  return { ok: true, errors: {} }
}
```

```tsx
// features/projects/components/create-project-form.tsx
'use client'

import { useActionState } from 'react'
import { createProject } from '../actions'

export function CreateProjectForm() {
  const [state, action, pending] = useActionState(createProject, {
    ok: false,
    errors: {},
  })

  return (
    <form action={action}>
      <label htmlFor="name">Project name</label>
      <input id="name" name="name" required minLength={2} />
      {state.errors.name ? <p role="alert">{state.errors.name[0]}</p> : null}
      <button disabled={pending}>{pending ? 'Creating...' : 'Create'}</button>
    </form>
  )
}
```

### Cached Server Data

```ts
// features/projects/data.ts
import 'server-only'
import { cacheLife, cacheTag } from 'next/cache'
import { db } from '@/lib/db'

export async function getProjects() {
  'use cache'
  cacheLife('minutes')
  cacheTag('projects')

  return db.project.findMany({
    select: { id: true, name: true, updatedAt: true },
    orderBy: { updatedAt: 'desc' },
  })
}
```

### Route Handler

```ts
// app/api/webhooks/billing/route.ts
import { verifyBillingSignature } from '@/features/billing/signature'
import { processBillingEvent } from '@/features/billing/webhooks'

export async function POST(request: Request) {
  const body = await request.text()
  const signature = request.headers.get('billing-signature')

  if (!signature || !verifyBillingSignature(body, signature)) {
    return Response.json({ error: 'Invalid signature' }, { status: 401 })
  }

  await processBillingEvent(body)
  return Response.json({ received: true })
}
```

### Zustand UI Store

```ts
// stores/sidebar-store.ts
'use client'

import { create } from 'zustand'

type SidebarStore = {
  open: boolean
  setOpen: (open: boolean) => void
  toggle: () => void
}

export const useSidebarStore = create<SidebarStore>((set) => ({
  open: true,
  setOpen: (open) => set({ open }),
  toggle: () => set((state) => ({ open: !state.open })),
}))
```

## Folder Organization

```text
app/
  api/webhooks/billing/route.ts
features/
  billing/
    signature.ts
    webhooks.ts
  projects/
    actions.ts
    data.ts
    components/create-project-form.tsx
stores/
  sidebar-store.ts
```

Keep examples organized by the production files they imply.

## TypeScript Examples

```ts
type ActionState<TField extends string> = {
  ok: boolean
  errors: Partial<Record<TField, string[]>>
}

type CreateProjectState = ActionState<'name'>
```

```ts
type CacheTagFactory = {
  all: string
  byId: (id: string) => string
}

export const projectTags: CacheTagFactory = {
  all: 'projects',
  byId: (id) => `projects:${id}`,
}
```

## Performance Considerations

- Keep examples server-first to avoid unnecessary hydration.
- Use precise cache tags to avoid broad invalidation.
- Keep client islands small.
- Avoid loading form libraries for simple native forms.
- Keep webhook handlers fast and enqueue slow side effects.

## Security Considerations

- Replace placeholder auth helpers with real project policies.
- Validate all input before mutation or webhook processing.
- Check tenant and ownership in authz helpers.
- Verify signatures for webhooks.
- Never store auth tokens in Zustand or localStorage.

## Accessibility Considerations

- Replace example labels and messages with localized text.
- Keep labels associated with inputs.
- Use `role="alert"` or live regions for form errors.
- Ensure sidebar state is reflected with `aria-expanded` in real controls.
- Preserve keyboard behavior when wrapping shadcn/Radix primitives.

