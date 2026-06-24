# Localization Review

## Review Scope

Verify next-intl usage, no hardcoded strings, RTL support, pluralization, formatting, localized metadata, and localized validation/errors.

## Blockers

- Hardcoded user-facing strings in localized routes.
- No RTL support for supported RTL locales.
- Metadata not localized.
- Pluralization done with string concatenation.
- Dates, numbers, or currencies formatted manually.

## What To Check

- Server Components use server translation helpers where possible.
- Client Components receive only required messages.
- All labels, placeholders, aria labels, alt text, toasts, and errors are localized.
- ICU messages handle plurals and variables.
- `lang` and `dir` are correct.
- CSS uses logical properties where direction matters.
- Cache tags or keys include locale when localized content differs.

## Severity Guidance

- `High`: app cannot support required locale/RTL, hardcoded critical flow text.
- `Medium`: metadata/errors not localized, bad pluralization, overhydrated messages.
- `Low`: minor copy extraction.

## Example Finding

```text
Medium - app/[locale]/settings/page.tsx:12
The page title and description are hardcoded in English.
Why it matters: This bypasses next-intl and creates untranslated UI for every non-English locale.
Recommended fix: load the Settings namespace with getTranslations and move the strings into messages files.
```

