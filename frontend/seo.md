# SEO

## Concepts

Use the Next.js Metadata API, metadata file conventions, structured data, sitemaps, robots, canonical URLs, localized alternates, and semantic HTML. SEO is tied to rendering strategy, performance, localization, and accessibility.

Why this exists: search engines and social crawlers need stable metadata and content, while users need fast, semantic pages.

## Best Practices

- Use `metadata` or `generateMetadata` in route segments.
- Type metadata with `Metadata`.
- Set `metadataBase` for absolute URLs.
- Localize titles, descriptions, Open Graph, and alternates.
- Use semantic headings and landmarks.
- Add JSON-LD for eligible content.
- Use `sitemap.ts` and `robots.ts`.
- Cache metadata data or mark dynamic metadata intentionally under Cache Components.

## Anti-Patterns

- Manually writing head tags in components when Metadata API supports them.
- Hardcoding English metadata.
- Duplicating titles across pages.
- Using client-only rendering for indexable primary content.
- Forgetting canonical URLs.
- Using deprecated metadata fields instead of current viewport APIs.

## Common Mistakes

- Relative Open Graph image URLs without `metadataBase`.
- Fetching uncached metadata while the page is otherwise prerenderable.
- Mismatched `lang`, canonical, and alternate links.
- Multiple `h1` elements because card titles use hero-level markup.
- Hiding primary content behind hydration.
- Missing descriptive alt text for content images.

## Production Examples

```tsx
import type { Metadata } from 'next'
import { getTranslations } from 'next-intl/server'
import { getArticle } from '@/features/articles/data'

type Props = {
  params: Promise<{ locale: string; slug: string }>
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale, slug } = await params
  const t = await getTranslations({ locale, namespace: 'Article' })
  const article = await getArticle(slug)

  if (!article) return { title: t('notFoundTitle') }

  return {
    title: article.title,
    description: article.description,
    openGraph: {
      title: article.title,
      description: article.description,
      type: 'article',
      images: [{ url: article.ogImageUrl, alt: article.title }],
    },
  }
}
```

## Folder Organization

```text
app/
  layout.tsx
  sitemap.ts
  robots.ts
  [locale]/
    articles/[slug]/page.tsx
features/seo/
  json-ld.tsx
```

Keep SEO helpers typed and route-aware.

## TypeScript Examples

```tsx
type ArticleJsonLdProps = {
  headline: string
  datePublished: string
  authorName: string
  url: string
}

export function ArticleJsonLd(props: ArticleJsonLdProps) {
  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{
        __html: JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'Article',
          headline: props.headline,
          datePublished: props.datePublished,
          author: { '@type': 'Person', name: props.authorName },
          url: props.url,
        }),
      }}
    />
  )
}
```

## Performance Considerations

- Server-render indexable content.
- Use streaming metadata defaults unless a crawler-specific need requires blocking.
- Optimize LCP images.
- Avoid client-only content for SEO-critical pages.
- Cache SEO data that changes infrequently.

## Security Considerations

- Escape or serialize JSON-LD safely with `JSON.stringify`.
- Do not put secrets or draft content in metadata.
- Validate slugs and canonical URLs to avoid open redirects or spam URLs.
- Avoid rendering untrusted HTML descriptions.
- Keep robots rules aligned with private route protection.

## Accessibility Considerations

- Semantic HTML improves both SEO and screen reader navigation.
- Use descriptive link text.
- Provide alt text for meaningful images.
- Keep heading order logical.
- Localize metadata and visible headings consistently.

