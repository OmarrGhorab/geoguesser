# Next.js 16+

## Concepts

Next.js 16+ is not just React with routing. It combines the App Router, React Server Components, Server Actions, Route Handlers, Cache Components, streaming, metadata, image/font optimization, and deployment primitives. Current APIs can differ from older training data, so inspect official docs or local `node_modules/next/dist/docs/` before making code changes in a real project.

Why this exists: Next.js releases can change conventions, defaults, and APIs. Production work must follow the installed version, not memory.

## Best Practices

- Use App Router only.
- Use TypeScript only.
- Use `next/link`, `next/navigation`, Metadata API, `next/image`, `next/font`, and native `fetch()`.
- Enable Cache Components when the project is ready for explicit caching.
- Prefer `proxy.ts` over old middleware conventions in Next.js 16+ projects.
- Use `pnpm` for package commands unless the existing repository has a different lockfile and the user tells you to preserve it.
- Read deprecation notices before using config flags or APIs.

## Anti-Patterns

- Recommending Pages Router APIs such as `getServerSideProps`, `getStaticProps`, `_app`, `_document`, or API Routes.
- Adding React Router.
- Treating `fetch` caching behavior as if it were older App Router implicit caching.
- Recommending `experimental.ppr` when Cache Components replaced that model.
- Using JavaScript examples in production docs or code.

## Common Mistakes

- Forgetting that `params` and `searchParams` are promises in current App Router examples.
- Assuming `fetch` requests are cached by default under Cache Components.
- Using outdated `middleware.ts` naming in a Next.js 16 migration without checking docs.
- Adding a package for a capability that Next.js already provides.
- Mixing Pages Router and App Router mental models in one answer.

## Production Examples

```ts
// next.config.ts
import type { NextConfig } from 'next'

const nextConfig: NextConfig = {
  cacheComponents: true,
  reactCompiler: true,
}

export default nextConfig
```

```tsx
// app/[locale]/layout.tsx
import type { Metadata } from 'next'
import { NextIntlClientProvider } from 'next-intl'
import { getMessages } from 'next-intl/server'

export const metadata: Metadata = {
  title: {
    default: 'Acme',
    template: '%s | Acme',
  },
}

export default async function LocaleLayout({
  children,
  params,
}: {
  children: React.ReactNode
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  const messages = await getMessages()

  return (
    <html lang={locale}>
      <body>
        <NextIntlClientProvider messages={messages}>
          {children}
        </NextIntlClientProvider>
      </body>
    </html>
  )
}
```

## Folder Organization

```text
app/
  [locale]/
    layout.tsx
    page.tsx
    loading.tsx
    error.tsx
    not-found.tsx
  api/
    webhooks/stripe/route.ts
next.config.ts
instrumentation.ts
proxy.ts
```

Use file conventions intentionally. Every special file has framework behavior; do not create them as generic components.

## TypeScript Examples

```tsx
import type { Metadata } from 'next'

type Props = {
  params: Promise<{ locale: string; slug: string }>
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { slug } = await params
  return { title: slug }
}

export default async function Page({ params }: Props) {
  const { slug } = await params
  return <main>{slug}</main>
}
```

## Performance Considerations

- Use Server Components to reduce client JavaScript.
- Use Suspense and loading files to stream meaningful UI early.
- Use Cache Components for stable reads and UI.
- Use `next/image` with correct sizes and priority only for true LCP images.
- Avoid client providers and global scripts that affect every route.

## Security Considerations

- Treat Server Actions and Route Handlers as externally reachable.
- Keep environment variables server-side unless intentionally prefixed for public exposure.
- Prefer server-only modules for DAL, auth, billing, and privileged service calls.
- Use secure headers through platform config or `next.config.ts`.
- Review Next.js security docs before custom auth or data access.

## Accessibility Considerations

- Use framework navigation without breaking browser semantics.
- Keep route pages structured around `main` and headings.
- Use accessible loading and error states for route boundaries.
- Ensure metadata titles describe the page after localization.
- Preserve focus and announcements during client transitions.

