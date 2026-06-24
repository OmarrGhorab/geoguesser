# Authorization

## Concepts

Authorization decides whether an authenticated or anonymous actor can access a route, read data, or perform a mutation. It must run on the server at every sensitive operation: Server Components, data functions, Server Actions, and Route Handlers.

Why this exists: UI visibility is not security. Server Actions and Route Handlers are reachable directly, and data functions can be called from multiple routes.

## Best Practices

- Authorize inside every Server Action and Route Handler.
- Authorize before reading sensitive data.
- Centralize reusable policies in `lib/authz`.
- Check tenant, role, ownership, and resource state.
- Return not found for resources that should not reveal existence.
- Keep Client Component permission state advisory only.
- Test authorization failures.

## Anti-Patterns

- Hiding buttons and assuming the mutation is protected.
- Checking auth only in layouts or Proxy.
- Passing role strings to the client and trusting them later.
- Mixing authorization logic into presentation components.
- Returning broad admin data to filter on the client.
- Using Zustand for permission decisions.

## Common Mistakes

- Checking only authentication, not ownership.
- Not rechecking after binding IDs to Server Actions.
- Forgetting webhooks and Route Handlers.
- Over-caching permission-sensitive data.
- Returning detailed forbidden messages for unauthorized resources.
- Not accounting for organization membership changes.

## Production Examples

```ts
// lib/authz/projects.ts
import 'server-only'
import { notFound } from 'next/navigation'
import { getSession } from '@/lib/auth/session'
import { db } from '@/lib/db'

export async function assertCanReadProject(projectId: string) {
  const session = await getSession()
  if (!session) notFound()

  const membership = await db.projectMember.findFirst({
    where: { projectId, userId: session.userId },
    select: { role: true },
  })

  if (!membership) notFound()
  return membership
}
```

```ts
'use server'

import { assertCanManageProject } from '@/lib/authz/projects'

export async function deleteProject(projectId: string) {
  await assertCanManageProject(projectId)
  await db.project.delete({ where: { id: projectId } })
}
```

## Folder Organization

```text
lib/authz/
  projects.ts
  teams.ts
features/projects/
  actions.ts
  data.ts
```

Policies belong in server-only modules and are imported by pages, actions, and handlers.

## TypeScript Examples

```ts
export type ProjectRole = 'viewer' | 'editor' | 'owner'

const roleRank: Record<ProjectRole, number> = {
  viewer: 1,
  editor: 2,
  owner: 3,
}

export function hasAtLeastRole(actual: ProjectRole, required: ProjectRole) {
  return roleRank[actual] >= roleRank[required]
}
```

## Performance Considerations

- Avoid repeated authorization queries by structuring data access functions around a single policy check.
- Cache only public or safely scoped permission metadata.
- Use database indexes for membership lookups.
- Do not overfetch sensitive data before authorization.
- Keep policy helpers focused and cheap.

## Security Considerations

- Authorization must be server-side and per operation.
- Treat client-provided IDs as untrusted.
- Use deny-by-default policies.
- Audit broad admin and cross-tenant paths.
- Avoid time-of-check/time-of-use gaps for critical operations.

## Accessibility Considerations

- Disabled controls should explain why an action is unavailable when appropriate.
- Do not render focusable controls that always fail authorization.
- Error pages for forbidden/not-found states need clear navigation.
- Avoid exposing sensitive resource names in denied messages.
- Keep permission changes understandable to screen reader users.

