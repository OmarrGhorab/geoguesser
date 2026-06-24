# Caching

## Concepts

Next.js 16 Cache Components make caching explicit. Use `use cache` in async functions or components, `cacheLife` for duration, `cacheTag` for invalidation groups, `updateTag` for immediate expiration after Server Actions, `revalidateTag(tag, 'max')` for stale-while-revalidate, and `revalidatePath` when no tag is available. Suspense marks dynamic holes that stream into a static shell.

Why this exists: production apps need fresh dynamic data and fast static shells on the same route.

## Best Practices

- Enable `cacheComponents: true` in suitable Next.js 16+ apps.
- Cache stable reads at the data-function level.
- Tag every cached read that can be invalidated by a mutation.
- Use `updateTag` in Server Actions for user-visible writes.
- Use `revalidateTag(tag, 'max')` for background refresh.
- Prefer tags over paths.
- Wrap uncached runtime data in Suspense.
- Keep cache keys deterministic and derived from serializable inputs.

## Anti-Patterns

- Assuming all `fetch` calls are cached by default.
- Caching request-specific secrets with public cache directives.
- Using `router.refresh()` as cache invalidation.
- Calling `revalidatePath('/')` for every write.
- Mixing old route segment config habits with Cache Components without checking docs.
- Using Zustand or localStorage as a server data cache.

## Common Mistakes

- Forgetting `cacheLife` for cached functions.
- Forgetting `cacheTag`, making invalidation imprecise.
- Using short-lived caches and expecting them to prerender.
- Reading `cookies()` inside a normal `use cache` scope.
- Placing Suspense too high and losing useful static shell content.
- Using `revalidateTag` without the required profile in Next.js 16+.

## Production Examples

```ts
// features/products/data.ts
import 'server-only'
import { cacheLife, cacheTag } from 'next/cache'
import { db } from '@/lib/db'

export async function getProducts() {
  'use cache'
  cacheLife('hours')
  cacheTag('products')

  return db.product.findMany({
    select: { id: true, name: true, price: true },
  })
}
```

```ts
// features/products/actions.ts
'use server'

import { updateTag } from 'next/cache'
import { db } from '@/lib/db'
import { assertCanManageProducts } from '@/lib/authz'

export async function updateProductName(id: string, name: string) {
  await assertCanManageProducts()
  await db.product.update({ where: { id }, data: { name } })
  updateTag('products')
}
```

## Folder Organization

```text
features/products/
  data.ts       # cached reads with tags
  actions.ts    # mutations that update/revalidate tags
  cache-keys.ts # optional tag helpers for complex domains
```

Use tag helper functions when tags include tenant, user, or resource IDs.

## TypeScript Examples

```ts
export const productTags = {
  all: 'products',
  byId: (id: string) => `products:${id}`,
} as const
```

```ts
import { revalidateTag } from 'next/cache'

export async function refreshCatalog() {
  revalidateTag(productTags.all, 'max')
}
```

## Performance Considerations

- Cache expensive stable reads close to the data source.
- Use Suspense to prevent uncached work from blocking the entire route.
- Avoid over-invalidating broad tags.
- Use longer cache lifetimes plus on-demand revalidation for CMS-like data.
- Keep cached return values compact.

## Security Considerations

- Do not cache user-specific sensitive data in shared caches.
- Include tenant or user identifiers in tags and keys when data is scoped.
- Use private/request-time rendering for personalized data.
- Avoid caching auth decisions beyond their safe lifetime.
- Review `cookies()` and `headers()` usage carefully.

## Accessibility Considerations

- Static shells should include meaningful landmarks and headings.
- Suspense fallbacks should describe what is loading.
- Avoid replacing focused interactive regions during revalidation.
- Prevent layout shifts when cached content refreshes.
- Announce mutation results when cache invalidation changes visible content.

