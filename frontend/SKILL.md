---
name: frontend
description: Production Next.js 16+ frontend engineering skill for building, reviewing, and refactoring App Router applications with TypeScript, Server Components, Server Actions, Route Handlers, native fetch, Cache Components, next-intl, Zustand, Tailwind CSS v4, shadcn/ui, Radix UI, React Hook Form, Zod, Motion, ESLint, Prettier, and pnpm. Use for Next.js architecture, UI implementation, data fetching, mutations, caching, localization, auth, accessibility, security, performance, deployment, testing, and production code standards. This skill is not for generic React, Pages Router, JavaScript, Axios, Redux Toolkit, React Query, React Router, or client-first SPA patterns.
---

# Frontend

## Mission

Build production-grade Next.js 16+ App Router frontends. Treat this as an opinionated engineering standard, not a generic React guide.

Always assume:

- Next.js 16+
- App Router
- TypeScript
- Server Components
- Server Actions
- Route Handlers
- Native `fetch()`
- Cache Components and `use cache`
- `next-intl`
- Zustand
- Tailwind CSS v4
- shadcn/ui and Radix UI
- React Hook Form and Zod
- Motion
- ESLint, Prettier, and pnpm

Never recommend:

- Axios
- Redux Toolkit
- React Query
- React Router
- Pages Router
- JavaScript
- CSS Modules unless an existing codebase explicitly requires them

## Concepts

Next.js 16+ is a server-first full-stack framework. App Router routes are Server Components by default, Server Actions handle in-app mutations, Route Handlers expose HTTP interfaces, and Cache Components make caching explicit with `use cache`, `cacheLife`, `cacheTag`, `updateTag`, `revalidateTag`, and `revalidatePath`.

The default architecture is:

- Render and fetch on the server.
- Add Client Components only for browser-only interactivity.
- Keep business logic, authorization, secrets, and writes on the server.
- Use the platform: native `fetch()`, Web `Request` and `Response`, semantic HTML, progressive enhancement, cookies, and HTTP.
- Let Next.js own routing, streaming, caching, navigation, and deployment behavior.

## How To Use This Skill

Read the relevant topic file before making recommendations or code changes:

- `architecture.md`: system shape, layering, decision rules.
- `nextjs.md`: Next.js 16+ baseline and official docs discipline.
- `app-router.md`: routing, layouts, loading, errors, metadata, dynamic segments.
- `server-components.md`: server/client boundaries and composition.
- `server-actions.md`: mutations, forms, auth checks, cache invalidation.
- `route-handlers.md`: public APIs, webhooks, external consumers.
- `fetch.md`: native fetch patterns and data access.
- `caching.md`: Cache Components, PPR, tags, revalidation.
- `typescript.md`: type strategy and strictness.
- `shadcn.md`: design system foundation.
- `tailwind.md`: Tailwind CSS v4 tokens and CSS variables.
- `zustand.md`: allowed client UI state only.
- `forms.md`: Server Actions plus React Hook Form when client interaction is needed.
- `validation.md`: Zod schemas at trust boundaries.
- `localization.md`: next-intl, messages, routing, RTL.
- `authentication.md`: sessions, cookies, login, signup.
- `authorization.md`: per-operation access checks.
- `seo.md`: Metadata API, structured data, sitemap, OG.
- `performance.md`: Core Web Vitals, JS budget, streaming.
- `accessibility.md`: semantic HTML, WCAG, Radix behavior.
- `security.md`: data safety, secrets, CSRF, headers.
- `testing.md`: unit, integration, E2E, accessibility checks.
- `folder-structure.md`: canonical project organization.
- `coding-standards.md`: code style and review rules.
- `deployment.md`: Vercel, runtime, env, observability.
- `checklists.md`: pre-implementation and release gates.
- `examples/`: copyable reference patterns for common production tasks.

Read only the files needed for the task. For cross-cutting work, start with `architecture.md`, then load the specific area.

## Best Practices

- Start with Server Components. Add `"use client"` only when state, event handlers, browser APIs, effects, Motion runtime animation, React Hook Form interaction, or Zustand subscriptions are required.
- Prefer Server Actions for mutations used by the app UI because they preserve progressive enhancement, run on the server, and can return updated UI and data in one roundtrip.
- Use Route Handlers only for public APIs, webhooks, external integrations, file downloads, health checks, or protocol-level needs.
- Fetch data in Server Components or server-only data modules. Never fetch inside `useEffect` when the data can be loaded on the server.
- Use native `fetch()` and direct database or service clients on the server. Do not wrap server-to-server calls in an internal Route Handler.
- Use Cache Components deliberately. Cache stable data or UI with `use cache`, set lifetime with `cacheLife`, tag with `cacheTag`, and invalidate with `updateTag` for read-your-own-writes or `revalidateTag(tag, 'max')` for background refresh.
- Use Suspense and streaming for uncached or request-time data.
- Keep Client Components small and leaf-oriented.
- Localize every user-facing string with `next-intl`; support RTL with `dir`, logical CSS properties, and locale-aware formatting.
- Validate all untrusted input with Zod on the server.
- Use shadcn/ui through composition. Do not casually edit generated component internals; wrap, compose, or create variants.
- Use Zustand only for global client UI state. Never use it for server data, caches, authentication tokens, or request results.

## Anti-Patterns

- Adding `"use client"` to layouts, pages, or large feature shells.
- Building an SPA inside Next.js with React Router, client data caches, and effect-driven fetching.
- Calling `/api/*` Route Handlers from Server Components instead of calling server logic directly.
- Creating Route Handlers for form submissions that are only used by the app UI.
- Storing auth tokens in `localStorage` or Zustand.
- Duplicating fetched data in Zustand.
- Hardcoding visible strings.
- Creating global mutable stores that can leak between requests.
- Using `router.refresh()` as a substitute for cache invalidation.
- Relying on layout-level auth checks while leaving Server Actions or Route Handlers unprotected.

## Common Mistakes

- Forgetting that `params` and `searchParams` are async in current App Router APIs.
- Passing non-serializable props from Server Components to Client Components.
- Importing server-only modules into Client Components.
- Using `revalidatePath` when a precise tag would avoid unnecessary work.
- Using `revalidateTag` when `updateTag` is required for read-your-own-writes.
- Reading `headers()`, `cookies()`, or uncached data outside Suspense when Cache Components are enabled.
- Treating Server Actions as private because they are not visible in the UI. They are reachable by POST and must authorize.

## Production Examples

```tsx
// app/[locale]/dashboard/page.tsx
import { Suspense } from 'react'
import { getTranslations } from 'next-intl/server'
import { getDashboardSummary } from '@/features/dashboard/data'
import { CreateProjectForm } from '@/features/projects/components/create-project-form'
import { ProjectListSkeleton } from '@/features/projects/components/project-list-skeleton'
import { ProjectList } from '@/features/projects/components/project-list'

export default async function DashboardPage() {
  const t = await getTranslations('Dashboard')
  const summary = await getDashboardSummary()

  return (
    <main>
      <h1>{t('title')}</h1>
      <p>{t('activeProjects', { count: summary.activeProjects })}</p>
      <CreateProjectForm />
      <Suspense fallback={<ProjectListSkeleton />}>
        <ProjectList />
      </Suspense>
    </main>
  )
}
```

```ts
// features/projects/actions.ts
'use server'

import { updateTag } from 'next/cache'
import { z } from 'zod'
import { assertCanCreateProject } from '@/lib/authz'
import { db } from '@/lib/db'

const createProjectSchema = z.object({
  name: z.string().trim().min(2).max(80),
})

export async function createProject(_prev: unknown, formData: FormData) {
  await assertCanCreateProject()

  const parsed = createProjectSchema.safeParse({
    name: formData.get('name'),
  })

  if (!parsed.success) {
    return { ok: false, errors: parsed.error.flatten().fieldErrors }
  }

  await db.project.create({ data: parsed.data })
  updateTag('projects')
  return { ok: true, errors: {} }
}
```

## Folder Organization

Use feature-first organization with route files in `app` and business logic in server-only modules:

```text
app/
  [locale]/
    layout.tsx
    page.tsx
    dashboard/
      page.tsx
features/
  projects/
    actions.ts
    components/
    data.ts
    schemas.ts
    types.ts
lib/
  auth/
  authz/
  db/
  i18n/
  utils.ts
messages/
  en.json
  ar.json
components/
  ui/
```

## TypeScript Examples

Prefer schema-derived types at trust boundaries:

```ts
import { z } from 'zod'

export const projectSchema = z.object({
  id: z.string().uuid(),
  name: z.string().min(2),
  createdAt: z.coerce.date(),
})

export type Project = z.infer<typeof projectSchema>
```

Type route props according to current App Router conventions:

```tsx
type PageProps = {
  params: Promise<{ locale: string; id: string }>
  searchParams: Promise<{ tab?: string }>
}

export default async function Page({ params, searchParams }: PageProps) {
  const [{ id }, { tab }] = await Promise.all([params, searchParams])
  return <main data-tab={tab}>{id}</main>
}
```

## Performance Considerations

- Minimize client JavaScript by keeping `"use client"` boundaries small.
- Stream slow sections with Suspense.
- Cache stable expensive reads with `use cache`.
- Use `next/image`, `next/font`, route-level code splitting, and static shells.
- Avoid waterfalls by starting independent async work before awaiting.
- Inspect bundle size before adding UI or animation packages.

## Security Considerations

- Authorize in every Server Action and Route Handler.
- Keep secrets on the server and mark server-only modules with `import 'server-only'`.
- Prefer HTTP-only, secure, same-site cookies for sessions.
- Validate all request input, form input, search params, cookies, and third-party API responses.
- Avoid exposing raw database records to Client Components; use DTOs.

## Accessibility Considerations

- Use semantic HTML first.
- Preserve keyboard navigation and focus order.
- Use Radix/shadcn primitives for complex interactive widgets.
- Provide labels, descriptions, error text, and `aria-live` regions for forms.
- Honor reduced motion and avoid layout shifts.
- Ensure localized and RTL experiences remain accessible.

## Exceptions

Accept exceptions only when they are explicit, documented, and tied to a real constraint:

- Use a Client Component wrapper for a third-party browser-only component.
- Use a Route Handler for a mutation only when the consumer is external, webhook-based, protocol-specific, or not a React form/app action.
- Use client-side data fetching only for browser-originated data that cannot be known on the server, such as live sensor state or post-hydration-only APIs.
- Use CSS Modules only in an existing codebase that already requires them or for third-party integration constraints.
- Use local storage only for non-sensitive user preferences that can safely be lost, leaked, or stale. Never store auth tokens there.
