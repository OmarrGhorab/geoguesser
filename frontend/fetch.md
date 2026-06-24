# Native fetch()

## Concepts

Use the platform-native `fetch()` API for HTTP requests. In Server Components, `fetch` can be awaited directly and identical requests are memoized during a render pass. Under Cache Components, requests are not cached by default; opt into caching with `use cache` around data functions or UI.

Why this exists: Next.js extends the platform instead of requiring an HTTP client abstraction. Native `fetch` works across Server Components, Server Actions, Route Handlers, and browser code.

## Best Practices

- Use native `fetch()` everywhere.
- Put server fetches in server-only data modules.
- Validate third-party responses with Zod.
- Throw typed errors or return explicit result objects.
- Start independent requests before awaiting to avoid waterfalls.
- Set headers, credentials, and cache behavior intentionally.
- Use direct function calls instead of internal API calls from server code.

## Anti-Patterns

- Installing Axios.
- Fetching server-renderable data in `useEffect`.
- Calling your own Route Handler from a Server Component.
- Duplicating the same request in a page and child component without a reason.
- Trusting `res.json()` as typed data.
- Storing fetched server data in Zustand.

## Common Mistakes

- Forgetting to check `response.ok`.
- Parsing error responses as if they match success schemas.
- Sending secrets to the browser.
- Fetching sequentially when requests are independent.
- Assuming `fetch` is globally cached forever.
- Mixing cache tags and HTTP cache headers without understanding each layer.

## Production Examples

```ts
// lib/http.ts
import { z } from 'zod'

export async function fetchJson<TSchema extends z.ZodType>(
  input: string | URL,
  schema: TSchema,
  init?: RequestInit
): Promise<z.infer<TSchema>> {
  const response = await fetch(input, init)

  if (!response.ok) {
    throw new Error(`Request failed: ${response.status}`)
  }

  return schema.parse(await response.json())
}
```

```ts
// features/github/data.ts
import 'server-only'
import { z } from 'zod'
import { fetchJson } from '@/lib/http'

const repositorySchema = z.object({
  full_name: z.string(),
  stargazers_count: z.number(),
})

export async function getRepository(name: string) {
  return fetchJson(`https://api.github.com/repos/${name}`, repositorySchema, {
    headers: { Accept: 'application/vnd.github+json' },
  })
}
```

## Folder Organization

```text
lib/
  http.ts
features/
  github/
    data.ts
    schemas.ts
```

Keep generic fetch helpers small. Keep service-specific schemas in feature modules.

## TypeScript Examples

```ts
const issueSchema = z.object({
  id: z.number(),
  title: z.string(),
  state: z.enum(['open', 'closed']),
})

type Issue = z.infer<typeof issueSchema>
```

```ts
const [repo, issues] = await Promise.all([
  getRepository('vercel/next.js'),
  getIssues('vercel/next.js'),
])
```

## Performance Considerations

- Avoid waterfalls with `Promise.all` for independent requests.
- Cache stable expensive reads with `use cache`.
- Stream uncached request-time reads behind Suspense.
- Use pagination and field selection for large APIs.
- Avoid client fetches that delay hydration and duplicate server work.

## Security Considerations

- Keep API tokens on the server.
- Validate external data before rendering or storing.
- Avoid proxying arbitrary URLs.
- Redact secrets from logs and errors.
- Apply timeouts or abort signals for unreliable upstreams.

## Accessibility Considerations

- Fetching strategy affects loading and error UI. Provide meaningful fallbacks.
- Preserve content structure while streamed data loads.
- Return enough state for empty results and errors to be announced.
- Avoid UI jumps when late data arrives.
- Localize all fetched labels before display when they are user-facing.

