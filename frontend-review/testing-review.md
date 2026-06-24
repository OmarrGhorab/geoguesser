# Testing Review

## Review Scope

Check Vitest, React Testing Library, Playwright, coverage, meaningful assertions, Server Actions, Route Handlers, accessibility, and production workflows.

## Blockers

- No tests for high-risk mutation, auth, payment, or destructive flows.
- Tests assert implementation details only.
- Missing authorization failure coverage.
- No E2E coverage for critical user flows.
- Accessibility-critical components without keyboard/error state tests.

## What To Check

- Server Actions test validation, authorization, success, and failure.
- Route Handlers test invalid payloads and auth failures.
- Components test user-observable behavior.
- Playwright covers critical journeys.
- Tests cover loading, error, empty, disabled, and success states.
- Localization and RTL are tested for affected UI.
- Assertions are meaningful and not just snapshots.

## Severity Guidance

- `High`: risky code path lacks tests.
- `Medium`: weak tests, missing failure states, insufficient coverage.
- `Low`: test organization or naming cleanup.

## Example Finding

```text
High - features/projects/actions.ts
The new deleteProject action has no test for unauthorized users or cross-project ownership.
Why it matters: This is a destructive mutation and must prove authorization behavior before merge.
Recommended fix: add action tests for owner success, member rejection, and unauthenticated rejection.
```

