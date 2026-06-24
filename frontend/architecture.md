# Architecture

## Concepts

Architecture starts with the App Router server-first execution model. Routes, layouts, and pages are Server Components unless a file declares `"use client"`. Server Components read data, compose UI, and protect secrets. Client Components provide browser interactivity. Server Actions mutate app data. Route Handlers expose HTTP endpoints. Cache Components make static, cached, and dynamic work coexist through `use cache`, Suspense, and streaming.

Why this exists: Next.js performance and security come from doing the default work on the server, sending less JavaScript, and letting the framework coordinate rendering, navigation, caching, and invalidation.

Accept exceptions when a requirement is truly browser-only, protocol-level, or owned by an external consumer.

## Best Practices

- Use Server Components first and make client boundaries as small as possible.
- Organize by feature, not by technical layer alone.
- Keep business logic in server-only modules under `features/*` or `lib/*`.
- Treat Server Actions and Route Handlers as public trust boundaries.
- Use a Data Access Layer for production apps with `server-only` imports.
- Use DTOs to return only fields the UI needs.
- Prefer composition over inheritance, global providers, or shared mutable singletons.
- Keep `app/` focused on routing, layouts, loading, errors, and metadata.

## Anti-Patterns

- SPA-first architecture with client routers and client caches.
- Global data stores for request data.
- API proxy layers called by Server Components.
- Feature code spread randomly across `components/`, `hooks/`, and `utils/`.
- Layouts that own auth state but leave leaf operations unprotected.
- One giant Client Component per page.

## Common Mistakes

- Treating a route segment as the only ownership boundary. Features often span pages, actions, data, schemas, and UI.
- Reusing database entity types directly in client props.
- Putting providers at the root when they only serve one route group.
- Creating abstractions before the second real use case.
- Forgetting that a module imported by a Client Component enters the client graph.

## Production Examples

```text
Request -> app/[locale]/dashboard/page.tsx
  -> Server Component fetches summary through features/dashboard/data.ts
  -> Suspense streams dynamic project list
  -> Client form island calls Server Action
  -> Server Action validates, authorizes, mutates, updateTag('projects')
  -> Next.js returns updated UI and data
```

```tsx
// app/[locale]/dashboard/page.tsx
import { Suspense } from 'react'
import { getDashboardSummary } from '@/features/dashboard/data'
import { ProjectList } from '@/features/projects/components/project-list'
import { ProjectListSkeleton } from '@/features/projects/components/project-list-skeleton'

export default async function DashboardPage() {
  const summary = await getDashboardSummary()

  return (
    <main>
      <h1>{summary.title}</h1>
      <Suspense fallback={<ProjectListSkeleton />}>
        <ProjectList />
      </Suspense>
    </main>
  )
}
```

## Folder Organization

```text
app/
  [locale]/
    (marketing)/
    (app)/
      dashboard/page.tsx
      settings/page.tsx
features/
  dashboard/
    data.ts
    components/
  projects/
    actions.ts
    data.ts
    schemas.ts
    components/
lib/
  auth/
  authz/
  db/
  env.ts
components/
  ui/
messages/
```

Use `app/` for route composition. Use `features/` for product behavior. Use `lib/` for cross-cutting infrastructure.

## TypeScript Examples

```ts
// features/projects/dto.ts
export type ProjectListItem = {
  id: string
  name: string
  href: string
  updatedLabel: string
}
```

```ts
// features/projects/data.ts
import 'server-only'
import { cacheLife, cacheTag } from 'next/cache'
import { db } from '@/lib/db'

export async function getProjectList(): Promise<ProjectListItem[]> {
  'use cache'
  cacheLife('minutes')
  cacheTag('projects')

  const projects = await db.project.findMany({
    select: { id: true, name: true, updatedAt: true },
    orderBy: { updatedAt: 'desc' },
  })

  return projects.map((project) => ({
    id: project.id,
    name: project.name,
    href: `/projects/${project.id}`,
    updatedLabel: project.updatedAt.toISOString(),
  }))
}
```

## Performance Considerations

- Architecture should make the cheap path automatic: static shell, cached stable reads, streamed dynamic holes.
- Avoid root-level providers and client wrappers because they increase hydration cost across the app.
- Prefer colocated Suspense boundaries around slow subsections.
- Keep shared layout data stable or cached so navigation stays fast.
- Avoid internal HTTP hops from server code.

## Security Considerations

- Put secrets, database calls, and authorization in server-only modules.
- Do not expose database models, provider tokens, or raw auth sessions to Client Components.
- Validate and authorize at every write and external read.
- Use DTOs as the boundary between privileged data and UI props.
- Make route groups and layouts convenient, not security-critical.

## Accessibility Considerations

- Architecture must preserve semantic landmarks: `main`, `nav`, `aside`, `header`, `footer`.
- Route transitions must keep focus behavior predictable.
- Streaming fallbacks must communicate useful loading states without trapping keyboard users.
- Feature components should own labels and descriptions close to controls.
- Localization and RTL support must be part of route architecture, not added later.

