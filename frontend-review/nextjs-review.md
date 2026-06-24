# Next.js Review

## Review Scope

Verify App Router usage, Server Components by default, narrow `"use client"` boundaries, Server Actions, Route Handlers, async Server Components, Metadata API, loading/error layouts, templates, proxy/middleware conventions, streaming, Suspense, Cache Components, revalidation, cache tags, Partial Prerendering, image optimization, and font optimization.

## Blockers

- Pages Router APIs or React Router in app code.
- Fetching initial server-renderable data inside `useEffect`.
- Unnecessary `"use client"` on pages, layouts, or large shells.
- Server Action without auth/authorization for sensitive mutation.
- Route Handler used for internal app-only form mutation.
- Missing cache invalidation after mutation of cached data.
- Runtime data or uncached async work outside Suspense when Cache Components require a boundary.

## What To Check

- Pages and layouts remain Server Components unless impossible.
- Client Components are used for state, event handlers, browser APIs, effects, or browser-only libraries only.
- Server Actions are async and use `'use server'`.
- Route Handlers are for public APIs, webhooks, external consumers, downloads, or protocol needs.
- `params` and `searchParams` match current async App Router conventions.
- `loading.tsx`, `error.tsx`, and Suspense are placed where they improve UX.
- Metadata uses the Metadata API and is localized when needed.
- `next/image` and `next/font` are used correctly.
- Cache tags and revalidation map to the changed data.

## Severity Guidance

- `Critical`: auth bypass, data leak, or core route broken by server/client misuse.
- `High`: avoidable client rendering, missing mutation invalidation, wrong routing model, or major hydration risk.
- `Medium`: missing loading/error boundary, suboptimal metadata, moderate duplication.
- `Low`: convention drift without immediate risk.

## Example Finding

```text
High - app/[locale]/dashboard/page.tsx:1
The dashboard page is a Client Component solely to call useEffect for project data.
Why it matters: This forfeits Server Component rendering, delays data until hydration, increases client JS, and duplicates Next.js caching behavior.
Recommended fix: Make the page async, fetch projects server-side, and move only filter toggles into a Client Component.
```

