# SEO Review

## Review Scope

Verify Metadata API, canonical URLs, OpenGraph, Twitter Cards, structured data, robots, sitemap, metadata generation, localization, and semantic content.

## Blockers

- SEO-critical content rendered only after client hydration.
- Missing or wrong canonical URL on public indexable pages.
- Metadata hardcoded in the wrong language.
- Structured data built from unsanitized/untrusted content.
- Private routes exposed in sitemap or robots.

## What To Check

- `metadata` or `generateMetadata` uses current Next.js APIs.
- Public pages have unique localized titles and descriptions.
- OpenGraph images have absolute URLs and useful alt text.
- Canonical and alternates match route/localization strategy.
- `sitemap.ts` and `robots.ts` reflect public/private boundaries.
- JSON-LD is safely serialized.
- Heading structure supports page topic.

## Severity Guidance

- `High`: public SEO page not indexable or metadata is misleading/broken.
- `Medium`: incomplete OG/canonical/localized metadata.
- `Low`: structured data or metadata polish.

## Example Finding

```text
Medium - app/[locale]/blog/[slug]/page.tsx:8
generateMetadata returns the same English description for every locale.
Why it matters: Localized pages share incorrect snippets and weaken search relevance.
Recommended fix: load locale-specific article metadata and include localized alternates.
```

