# Route Handlers

## Concepts

Route Handlers are `route.ts` files inside `app/` that handle HTTP methods with Web `Request` and `Response` APIs. Use them for public APIs, webhooks, external consumers, file responses, health checks, and protocol-level integrations. They do not participate in layouts or client navigation like pages.

Why this exists: not every server endpoint is a React UI mutation. External systems need stable HTTP contracts.

## Best Practices

- Use Route Handlers only when there is an HTTP consumer outside the React app or a protocol requirement.
- Use native `Request`, `Response`, `NextRequest`, and `NextResponse`.
- Validate request body, params, headers, and signatures.
- Keep handlers thin; call server-only services for business logic.
- Use `RouteContext<'/path/[param]'>` for typed params.
- Cache GET handlers deliberately with Cache Components helpers in extracted functions.
- Return precise status codes and JSON shapes.

## Anti-Patterns

- Creating `/api/*` for every app mutation.
- Calling Route Handlers from Server Components.
- Using Axios inside handlers.
- Putting a `route.ts` beside `page.tsx` in the same segment.
- Trusting webhook payloads without signature verification.
- Returning raw exceptions or stack traces.

## Common Mistakes

- Forgetting that non-GET methods are not cached.
- Trying to use `use cache` directly inside the handler body instead of in a helper.
- Reading `request.body` twice.
- Skipping idempotency for webhooks.
- Returning inconsistent error formats.
- Missing method-specific auth checks.

## Production Examples

```ts
// app/api/projects/[id]/route.ts
import { NextRequest } from 'next/server'
import { z } from 'zod'
import { assertCanReadProject } from '@/lib/authz'
import { getProjectApiDto } from '@/features/projects/data'

const responseSchema = z.object({
  id: z.string(),
  name: z.string(),
})

export async function GET(
  _request: NextRequest,
  context: RouteContext<'/api/projects/[id]'>
) {
  const { id } = await context.params
  await assertCanReadProject(id)

  const project = await getProjectApiDto(id)
  if (!project) return Response.json({ error: 'Not found' }, { status: 404 })

  return Response.json(responseSchema.parse(project))
}
```

```ts
// app/api/webhooks/billing/route.ts
import { verifyBillingSignature } from '@/features/billing/webhooks'

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

## Folder Organization

```text
app/api/
  health/route.ts
  webhooks/
    billing/route.ts
  public/
    projects/[id]/route.ts
features/
  billing/
    webhooks.ts
  projects/
    data.ts
```

Separate transport from domain logic.

## TypeScript Examples

```ts
type ApiError = {
  error: string
  code: 'unauthorized' | 'not_found' | 'invalid_request'
}

function jsonError(body: ApiError, status: number) {
  return Response.json(body, { status })
}
```

## Performance Considerations

- Avoid server-to-server HTTP hops inside the same app.
- Cache public GET data only when freshness requirements allow it.
- Stream large responses when appropriate.
- Keep webhook handlers fast; enqueue slow side effects.
- Avoid importing UI code into handlers.

## Security Considerations

- Treat every handler as public.
- Verify auth, authorization, tenant, and signatures per method.
- Validate input with Zod before use.
- Use idempotency keys for external event processing.
- Avoid exposing internal IDs unless the API contract requires them.
- Set cache and content headers intentionally.

## Accessibility Considerations

- Route Handlers usually do not render UI, but their errors affect accessible UI consumers.
- Return structured errors that forms and live regions can announce.
- For file downloads, set descriptive filenames and content types.
- For public APIs consumed by pages, include enough state for accessible empty/error UI.
- Do not rely on color-only status values in API-driven UI.

