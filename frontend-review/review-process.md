# Review Process

## Purpose

Run every review as a production PR review. The goal is to catch architecture, performance, accessibility, UX, security, maintainability, localization, SEO, state, and testing problems before merge.

## Process

1. Identify the changed files, route segments, components, actions, handlers, schemas, stores, tests, and config.
2. Read surrounding code before judging a diff hunk.
3. Classify each changed file as Server Component, Client Component, Server Action, Route Handler, data module, UI primitive, form, schema, store, or config.
4. Check the highest-risk boundaries first: auth, authorization, server/client imports, cache invalidation, user input, secrets, and client bundles.
5. Load category review files that match the changed surfaces.
6. Write findings with severity, location, explanation, why it matters, recommended fix, and example solution when useful.
7. Score categories using `scoring.md`.
8. Emit the report using `report-template.md`.

## Strictness Rules

- Do not approve code that uses banned technologies without explicit project requirement.
- Do not approve Server Actions or Route Handlers without validation and authorization where sensitive.
- Do not approve client-side data fetching when a Server Component can fetch the data.
- Do not approve hardcoded user-facing strings in localized apps.
- Do not approve inaccessible forms, dialogs, menus, or icon controls.
- Do not approve tokens in `localStorage`, Zustand, or client-readable storage.

## Finding Quality Bar

A finding must be actionable. Prefer one precise issue over a vague category complaint.

Good:

```text
High - app/projects/actions.ts:24
The delete action trusts projectId from FormData and deletes without checking project membership.
Why it matters: Server Actions are reachable by direct POST; any authenticated user could attempt cross-tenant deletion.
Fix: Load the project membership server-side and require owner role before deleting.
```

Bad:

```text
Security could be better.
```

