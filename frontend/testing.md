# Testing

## Concepts

Testing verifies server-first behavior, mutations, routing, validation, accessibility, and user flows. Use tests at the right layer: unit tests for pure logic, integration tests for actions/data boundaries, component tests for UI states, and E2E tests for production workflows.

Why this exists: Next.js apps mix server and client concerns. Tests must catch boundary mistakes, not just component snapshots.

## Best Practices

- Test Server Actions for validation, authorization, mutation, and invalidation.
- Test Route Handlers with real `Request` objects.
- Test accessibility states with Testing Library and automated a11y checks.
- Use Playwright for critical flows.
- Mock external services at boundaries.
- Run typecheck, lint, format, unit, and E2E gates in CI.
- Test localized and RTL routes.

## Anti-Patterns

- Snapshot-only coverage.
- Testing implementation details of shadcn/Radix primitives.
- Mocking authorization away in mutation tests.
- E2E tests that depend on external production services.
- Ignoring Server Components because they are async.
- Skipping accessibility and keyboard tests.

## Common Mistakes

- Not testing failed Server Action states.
- Not testing unauthenticated and unauthorized users.
- Not testing cache invalidation after writes.
- Using client-only test patterns for server modules.
- Forgetting route params are async.
- Not checking form labels and error associations.

## Production Examples

```ts
// features/projects/actions.test.ts
import { createProject } from './actions'

test('returns validation errors for invalid project names', async () => {
  const formData = new FormData()
  formData.set('name', 'x')

  const result = await createProject({ ok: false, errors: {} }, formData)

  expect(result.ok).toBe(false)
  expect(result.errors.name).toBeDefined()
})
```

```ts
// app/api/health/route.test.ts
import { GET } from './route'

test('returns ok', async () => {
  const response = await GET()
  await expect(response.json()).resolves.toEqual({ ok: true })
})
```

## Folder Organization

```text
features/projects/
  actions.test.ts
  data.test.ts
tests/
  e2e/
  accessibility/
```

Co-locate focused tests with feature code; keep cross-route E2E tests under `tests`.

## TypeScript Examples

```ts
type TestUser = {
  id: string
  role: 'member' | 'admin'
}

export function createTestUser(overrides: Partial<TestUser> = {}): TestUser {
  return { id: 'user_1', role: 'member', ...overrides }
}
```

## Performance Considerations

- Keep unit tests fast and deterministic.
- Use E2E for critical paths, not every branch.
- Avoid starting full browsers for pure server logic.
- Run expensive suites in CI stages.
- Include performance budgets or Lighthouse checks for key pages when feasible.

## Security Considerations

- Test auth bypass attempts.
- Test tenant isolation.
- Test invalid payloads and malformed JSON.
- Test webhook signature rejection.
- Avoid real secrets in tests.
- Reset test data between cases.

## Accessibility Considerations

- Assert controls have accessible names.
- Test keyboard navigation for menus, dialogs, and forms.
- Run automated a11y checks on key pages.
- Test error messages and live regions.
- Test RTL and zoom-sensitive layouts.

