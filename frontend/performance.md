# Performance

## Concepts

Performance in Next.js 16+ comes from server-first rendering, small client bundles, Cache Components, streaming, optimized media/fonts, stable layout, and precise invalidation. Optimize Core Web Vitals: LCP, INP, CLS, and the supporting network/server metrics.

Why this exists: speed is architecture. Client-first choices create hydration, network, and CPU costs that cannot be fixed with micro-optimizations later.

## Best Practices

- Keep pages Server Components by default.
- Use small Client Component islands.
- Use Suspense for slow or uncached sections.
- Cache stable data and UI with `use cache`.
- Avoid waterfalls with parallel async work.
- Use `next/image` and `next/font`.
- Limit third-party scripts.
- Profile bundle size and runtime interaction costs.

## Anti-Patterns

- Adding app-wide client providers by default.
- Fetching initial data in `useEffect`.
- Using heavy animation and chart libraries above the fold.
- Revalidating entire paths for small changes.
- Loading all locale messages on every page.
- Shipping server-only utilities to the client.

## Common Mistakes

- Not setting image `sizes`.
- Marking too much as priority.
- Creating layout shifts with dynamic labels or images.
- Waterfalling sequential awaits.
- Using Suspense fallback that is larger than final content.
- Ignoring INP by adding expensive client state updates.

## Production Examples

```tsx
import { Suspense } from 'react'

export default function Page() {
  const summaryPromise = getSummary()
  const chartPromise = getChartData()

  return (
    <main>
      <Summary data={summaryPromise} />
      <Suspense fallback={<ChartSkeleton />}>
        <Chart data={chartPromise} />
      </Suspense>
    </main>
  )
}
```

```tsx
import Image from 'next/image'

export function HeroImage() {
  return (
    <Image
      src="/hero.jpg"
      alt="Product dashboard"
      width={1600}
      height={900}
      priority
      sizes="100vw"
    />
  )
}
```

## Folder Organization

```text
features/*/components/
  *.tsx
  *-skeleton.tsx
lib/performance/
  web-vitals.ts
```

Keep skeletons near the streamed components they represent.

## TypeScript Examples

```tsx
type AsyncProps<T> = {
  data: Promise<T>
}

export async function Chart({ data }: AsyncProps<ChartPoint[]>) {
  const points = await data
  return <ChartClient points={points} />
}
```

## Performance Considerations

- Treat every `"use client"` as a budget decision.
- Measure before adding memoization; prefer React Compiler where enabled.
- Use route-level code splitting naturally through App Router.
- Avoid unnecessary rerenders with narrow props and Zustand selectors.
- Keep initial HTML useful and stable.

## Security Considerations

- Performance shortcuts must not bypass authorization.
- Do not cache private data in shared caches.
- Avoid exposing secrets through client bundles while optimizing.
- Validate data before caching it.
- Do not disable security headers for marginal speed gains.

## Accessibility Considerations

- Fast pages still need visible focus and semantic structure.
- Skeletons should not confuse screen readers.
- Avoid motion that harms users with vestibular sensitivity.
- Prevent CLS that moves focused controls.
- Keep interactive response fast for keyboard and assistive tech users.

