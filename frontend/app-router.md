# App Router

## Concepts

The App Router maps folders to route segments. `layout.tsx` persists across child routes, `page.tsx` renders a URL, `loading.tsx` creates a Suspense boundary, `error.tsx` handles client-side recoverable errors, `not-found.tsx` handles 404 states, and `route.ts` owns HTTP methods for that segment. Route groups organize without changing the URL.

Why this exists: routing is part of the rendering architecture. Layout persistence, streaming, metadata, and nested boundaries are performance and UX tools.

## Best Practices

- Keep route files thin and compose feature modules.
- Use route groups for product areas such as `(marketing)` and `(app)`.
- Use `loading.tsx` for segment-level loading and Suspense for granular loading.
- Use `error.tsx` only where recovery needs client interactivity.
- Use `not-found.tsx` and `notFound()` for missing resources.
- Put public HTTP surfaces under `app/api/*/route.ts`.
- Type `params` and `searchParams` as promises.

## Anti-Patterns

- Building nested routing with React Router.
- Placing `route.ts` next to `page.tsx` in the same segment.
- Putting product logic directly in `page.tsx`.
- Reading `searchParams` in layouts and expecting them to update.
- Using layouts as authorization enforcement for all mutations.
- Adding `"use client"` to route files to access navigation hooks.

## Common Mistakes

- Forgetting that layouts do not rerender for every query string change.
- Making loading UI generic and unhelpful.
- Placing data fetching in a Client Component because route params are needed.
- Confusing route groups with URL paths.
- Letting one root layout own every provider even when route groups need different shells.

## Production Examples

```text
app/
  [locale]/
    layout.tsx
    (marketing)/
      page.tsx
    (app)/
      layout.tsx
      dashboard/
        page.tsx
        loading.tsx
        error.tsx
      projects/
        [id]/
          page.tsx
          not-found.tsx
```

```tsx
// app/[locale]/(app)/projects/[id]/page.tsx
import { notFound } from 'next/navigation'
import { getProject } from '@/features/projects/data'

type Props = {
  params: Promise<{ id: string }>
}

export default async function ProjectPage({ params }: Props) {
  const { id } = await params
  const project = await getProject(id)

  if (!project) notFound()

  return (
    <main>
      <h1>{project.name}</h1>
    </main>
  )
}
```

## Folder Organization

```text
app/[locale]/
  layout.tsx
  (app)/
    layout.tsx
    dashboard/
      page.tsx
      loading.tsx
    settings/
      page.tsx
  (marketing)/
    page.tsx
    pricing/page.tsx
```

Place reusable UI outside `app/` unless it is route-specific and not imported elsewhere.

## TypeScript Examples

```tsx
type SearchParams = {
  page?: string
  query?: string
}

type PageProps = {
  params: Promise<{ locale: string }>
  searchParams: Promise<SearchParams>
}

export default async function SearchPage({ searchParams }: PageProps) {
  const { page = '1', query = '' } = await searchParams
  return <main data-page={page}>{query}</main>
}
```

## Performance Considerations

- Split route groups by shell to reduce global layout work.
- Keep persistent layouts stable and cache layout data when safe.
- Use route segment loading for instant feedback and Suspense for partial rendering.
- Avoid dynamic runtime APIs in high-level layouts unless wrapped and intentional.
- Use parallel routes only when independent route slots improve UX.

## Security Considerations

- Do not treat hidden route groups as security.
- Protect pages, Server Actions, data functions, and Route Handlers independently.
- Avoid leaking resource existence through unauthorized route responses.
- Sanitize and validate route params before database use.
- Keep admin and app route groups explicit for reviewability.

## Accessibility Considerations

- Each page should expose one clear `h1`.
- Route loading states should identify the region being loaded.
- Error boundaries need recovery controls with accessible names.
- `not-found.tsx` should include navigation to a safe route.
- Persistent layouts should not steal focus on every navigation.

