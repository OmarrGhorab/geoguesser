# Performance Review

## Review Scope

Inspect bundle size, client JS, dynamic imports, lazy loading, Suspense, streaming, rerenders, unstable callbacks, images, fonts, excessive state, and Core Web Vitals.

## Blockers

- Large route marked `"use client"` without necessity.
- Initial server-renderable data fetched after hydration.
- Unoptimized LCP image.
- Excessive client state causing broad rerenders in critical interactions.
- Heavy library added to the client bundle for simple behavior.

## What To Check

- Server Components are used for data-heavy UI.
- Client islands are small.
- Suspense boundaries stream slow sections.
- Cache Components are used for stable expensive reads.
- Independent async work is parallelized.
- `next/image` includes meaningful `sizes`; `priority` is reserved for true LCP images.
- `next/font` is used for fonts.
- Motion respects reduced motion and avoids layout thrash.
- Zustand selectors avoid broad rerenders.

## Severity Guidance

- `High`: major LCP/INP/CLS or bundle regression in important route.
- `Medium`: avoidable hydration, rerenders, missing streaming, unoptimized media.
- `Low`: micro-optimization or cleanup.

## Example Finding

```text
High - app/[locale]/reports/page.tsx:1
The reports page is a Client Component and imports the charting library before any data is loaded.
Why it matters: The route ships a large JS bundle and blocks useful content behind hydration.
Recommended fix: render report summary and data table as Server Components, dynamically import only the interactive chart island.
```

