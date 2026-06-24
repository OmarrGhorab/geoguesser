# Validation Review

## Review Scope

Ensure shared schemas, `z.infer`, refinements, server validation, client validation, and no duplicated validation logic.

## Blockers

- Untrusted input used without Zod validation.
- Duplicated client/server validation rules that can drift.
- Type assertions replacing parsing.
- Third-party API responses trusted directly.
- File uploads accepted without server-side validation.

## What To Check

- FormData, JSON bodies, route params, search params, cookies, headers, and external responses are parsed.
- `safeParse` returns user-correctable errors.
- `parse` is reserved for invariants or trusted internal data.
- `refine` and `superRefine` model cross-field rules.
- Schemas are close to trust boundaries.
- `z.input` and `z.output` are used correctly when coercion exists.

## Severity Guidance

- `Critical`: validation gap enables security issue or data corruption.
- `High`: missing validation on mutation/API boundary.
- `Medium`: duplicated validation, weak schema, poor error mapping.
- `Low`: schema organization cleanup.

## Example Finding

```text
High - app/api/invite/route.ts:14
The handler reads request.json() and inserts the body without schema validation.
Why it matters: External callers can send malformed roles or unexpected fields, causing privilege bugs or data corruption.
Recommended fix: parse the body with a Zod schema and reject invalid requests with a 400 response.
```

