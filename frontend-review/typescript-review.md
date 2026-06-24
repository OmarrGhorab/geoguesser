# TypeScript Review

## Review Scope

Reject weak types and unsafe boundaries. Review inference, generics, utility types, discriminated unions, reusable models, duplicated types, assertions, implicit `any`, and `unknown` handling.

## Blockers

- `any` in application code without a documented, narrow escape hatch.
- Unsafe assertions on untrusted data.
- JavaScript files in production paths.
- Form, API, route, or external response data typed without runtime validation.
- Non-serializable props passed to Client Components.

## What To Check

- `unknown` is parsed or narrowed before use.
- Zod-derived types are used for validated data.
- Discriminated unions model action and UI states.
- Duplicated types are consolidated only when they represent the same contract.
- Generics make code safer, not harder to understand.
- Interfaces are used only where extension/merging is intended; type aliases are acceptable for props and unions.
- Route props reflect current async `params` and `searchParams` conventions.

## Severity Guidance

- `High`: unsafe input typing, `any` masking data/security bugs, wrong route types causing runtime bugs.
- `Medium`: weak action state typing, duplicated models, overbroad assertions.
- `Low`: naming or utility type cleanup.

## Example Finding

```text
Medium - features/invite/actions.ts:18
The action casts FormData values with `as string` before validation.
Why it matters: FormData can contain null or File values; the cast hides invalid inputs from TypeScript and can produce broken writes.
Recommended fix: pass raw values into a Zod schema and use the parsed result.
```

