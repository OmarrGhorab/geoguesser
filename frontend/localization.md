# Localization

## Concepts

Use `next-intl` for all localization. Locales should shape routing, message loading, formatting, metadata, validation messages, and layout direction. Never hardcode user-facing strings.

Why this exists: localization is architecture, not copy replacement. It affects URLs, cache keys, SEO, layout, accessibility, dates, numbers, pluralization, and RTL.

## Best Practices

- Use a top-level `[locale]` segment unless the product explicitly requires domain or cookie-only routing.
- Keep messages in `messages/{locale}.json`.
- Use server translation helpers in Server Components.
- Use `NextIntlClientProvider` only where client translations are needed.
- Set `<html lang>` and `dir`.
- Use locale-aware formatting for dates, numbers, currency, and relative time.
- Use logical CSS and test RTL.

## Anti-Patterns

- Hardcoding visible strings in JSX.
- Translating only pages but not forms, errors, metadata, and aria labels.
- Passing all messages to every client component.
- Concatenating translated strings instead of using ICU messages.
- Using physical `left` and `right` styling everywhere.
- Treating English text as stable IDs.

## Common Mistakes

- Forgetting metadata localization.
- Forgetting validation error localization.
- Hydrating huge message catalogs on every page.
- Not localizing route labels and navigation.
- Using icons that imply direction without mirroring in RTL.
- Not validating unsupported locales.

## Production Examples

```tsx
// app/[locale]/layout.tsx
import { NextIntlClientProvider } from 'next-intl'
import { getMessages } from 'next-intl/server'

const rtlLocales = new Set(['ar', 'he', 'fa', 'ur'])

export default async function LocaleLayout({
  children,
  params,
}: {
  children: React.ReactNode
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  const messages = await getMessages()
  const dir = rtlLocales.has(locale) ? 'rtl' : 'ltr'

  return (
    <html lang={locale} dir={dir}>
      <body>
        <NextIntlClientProvider messages={messages}>
          {children}
        </NextIntlClientProvider>
      </body>
    </html>
  )
}
```

```tsx
import { getTranslations } from 'next-intl/server'

export default async function SettingsPage() {
  const t = await getTranslations('Settings')
  return (
    <main>
      <h1>{t('title')}</h1>
      <p>{t('description')}</p>
    </main>
  )
}
```

## Folder Organization

```text
app/[locale]/
  layout.tsx
  page.tsx
i18n/
  request.ts
  routing.ts
messages/
  en.json
  ar.json
```

Keep message namespaces aligned with features or routes.

## TypeScript Examples

```ts
export const locales = ['en', 'ar'] as const
export type Locale = (typeof locales)[number]

export function isLocale(value: string): value is Locale {
  return locales.includes(value as Locale)
}
```

```ts
export function getDirection(locale: Locale): 'ltr' | 'rtl' {
  return locale === 'ar' ? 'rtl' : 'ltr'
}
```

## Performance Considerations

- Load only the messages needed by the route or client island.
- Prefer server translations to avoid sending large catalogs.
- Include locale in cache keys or tags when content differs by locale.
- Avoid formatting large lists on the client when the server can do it.
- Test font loading for all scripts.

## Security Considerations

- Validate locale params against an allowlist.
- Do not use translation strings as trusted HTML.
- Sanitize rich text inputs before translation rendering.
- Avoid leaking unpublished localized content via broad caches.
- Keep locale redirects constrained to known routes.

## Accessibility Considerations

- Set `lang` and `dir` on the document.
- Localize `aria-label`, `alt`, titles, and form errors.
- Ensure reading order is correct in RTL.
- Do not use icons alone where direction or meaning changes by locale.
- Use locale-aware names for navigation landmarks when needed.

