# Deployment

## Concepts

Deploy Next.js 16+ apps with a platform that supports App Router, Server Components, Server Actions, Route Handlers, streaming, Cache Components, image optimization, environment variables, and observability. Vercel is the default recommendation unless project constraints require another supported host.

Why this exists: framework features depend on runtime capabilities. Deployment is part of architecture.

## Best Practices

- Use Vercel for first-class Next.js support when possible.
- Validate environment variables at boot.
- Keep preview, staging, and production environments separate.
- Use `pnpm build`, typecheck, lint, and tests in CI.
- Configure cache, headers, images, and redirects intentionally.
- Monitor Web Vitals, server errors, and action failures.
- Use secure secret management.

## Anti-Patterns

- Deploying to a host that buffers streaming without understanding the UX impact.
- Assuming in-memory cache is durable in serverless environments.
- Committing `.env.local`.
- Disabling typecheck or lint to ship.
- Depending on local filesystem writes at runtime.
- Treating preview deployments as production-secure by default.

## Common Mistakes

- Missing environment variables in preview.
- Incorrect absolute URLs for metadata and webhooks.
- Not setting webhook endpoints per environment.
- Forgetting image remote patterns.
- Not testing Server Actions after deployment.
- Assuming local cache behavior matches multi-instance production.

## Production Examples

```json
{
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "eslint .",
    "format": "prettier --check .",
    "typecheck": "tsc --noEmit"
  }
}
```

```ts
// lib/env.ts
import { z } from 'zod'

const envSchema = z.object({
  NEXT_PUBLIC_APP_URL: z.string().url(),
  DATABASE_URL: z.string().url(),
  SESSION_SECRET: z.string().min(32),
})

export const env = envSchema.parse(process.env)
```

## Folder Organization

```text
.github/workflows/
next.config.ts
instrumentation.ts
lib/env.ts
```

Keep deployment config reviewed and versioned. Keep secrets outside the repo.

## TypeScript Examples

```ts
import type { NextConfig } from 'next'

const nextConfig: NextConfig = {
  cacheComponents: true,
  images: {
    remotePatterns: [{ protocol: 'https', hostname: 'images.example.com' }],
  },
}

export default nextConfig
```

## Performance Considerations

- Verify streaming behavior in the target runtime.
- Use platform image optimization and CDN features.
- Understand cache durability in serverless and multi-region deployments.
- Track Web Vitals in production.
- Keep cold-start-sensitive code lean.

## Security Considerations

- Use platform secret storage.
- Scope environment variables per environment.
- Set secure cookies in production.
- Validate webhook signatures against environment-specific secrets.
- Review headers, CSP, and allowed image/script origins.

## Accessibility Considerations

- Run accessibility tests in CI for critical pages.
- Verify deployed fonts and locale assets load correctly.
- Monitor real-user UX, not just lab metrics.
- Ensure error pages are accessible in production.
- Keep preview URLs available for accessibility review.

