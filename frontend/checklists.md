# Checklists

## Concepts

Use checklists to enforce the skill's opinions during planning, implementation, review, and release. They are not bureaucracy; they catch the predictable mistakes in Next.js 16+ work.

Why this exists: production quality comes from repeatedly applying the same boundary, performance, security, localization, and accessibility checks.

## Best Practices

- Run the smallest checklist that matches the task.
- Treat any banned technology as a stop sign unless the user explicitly requires an exception.
- Verify server/client boundaries before coding.
- Verify auth, validation, and cache invalidation before shipping.
- Run lint, typecheck, tests, and build when feasible.
- Review UI in desktop, mobile, LTR, RTL, loading, error, and empty states.

## Anti-Patterns

- Starting with a Client Component by habit.
- Skipping validation because TypeScript passed.
- Skipping accessibility checks for internal tools.
- Adding dependencies before checking platform APIs.
- Shipping forms without Server Action error states.
- Releasing without testing cache invalidation.

## Common Mistakes

- Forgetting localized strings in aria labels.
- Forgetting route handler auth.
- Forgetting to update tags after mutations.
- Forgetting loading states for streamed sections.
- Forgetting mobile text wrapping.
- Forgetting to check current official docs for changed Next.js APIs.

## Production Examples

Implementation checklist:

- Server Component by default.
- Client boundary justified.
- Data read is server-side.
- Mutation uses Server Action unless external HTTP consumer exists.
- Zod validation exists at boundary.
- Authorization exists at operation.
- Cache tags and invalidation are defined.
- UI strings use next-intl.
- Form labels and errors are accessible.
- Tests cover success, failure, and unauthorized cases.

Review checklist:

- No Axios, Redux Toolkit, React Query, React Router, Pages Router, JavaScript, or avoidable CSS Modules.
- No auth tokens in localStorage or Zustand.
- No server data cache in Zustand.
- No internal API call from Server Component.
- No broad `"use client"` boundary.
- No hardcoded user-facing strings.

## Folder Organization

```text
frontend/checklists.md
features/*/
  actions.ts
  data.ts
  schemas.ts
  components/
```

Keep task checklists in this skill; keep project-specific release checklists in the project if needed.

## TypeScript Examples

```ts
type ReviewResult = {
  passed: boolean
  blockers: string[]
  warnings: string[]
}

export function createReviewResult(): ReviewResult {
  return { passed: true, blockers: [], warnings: [] }
}
```

## Performance Considerations

- Check bundle impact for new client dependencies.
- Check Suspense placement.
- Check LCP image handling.
- Check hydration scope.
- Check cache tags for precision.

## Security Considerations

- Check auth and authorization at every server boundary.
- Check secrets remain server-only.
- Check input validation.
- Check cache privacy.
- Check dependency risk.

## Accessibility Considerations

- Check semantic landmarks and heading order.
- Check labels, descriptions, and error messages.
- Check keyboard behavior.
- Check focus management.
- Check color contrast, reduced motion, RTL, and zoom.

