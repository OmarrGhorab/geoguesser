# Forms Review

## Review Scope

Review React Hook Form, Zod, reusable fields, accessible labels, loading states, error states, disabled states, submission handling, and Server Action integration.

## Blockers

- Form submission through internal Route Handler when Server Action is appropriate.
- Client-only validation without server validation.
- Missing labels on fields.
- No pending/error state for important mutations.
- Hidden trusted fields used without server verification.

## What To Check

- Native forms and Server Actions are used when possible.
- React Hook Form is justified by complex client interaction.
- Zod schema validates server input.
- Field errors map to field names.
- Inputs have `name`, `id`, labels, autocomplete where relevant, and accessible error text.
- Submit buttons expose pending state.
- Disabled states do not trap users.
- Errors and success messages are localized.

## Severity Guidance

- `High`: inaccessible or insecure form, missing server validation, data loss.
- `Medium`: weak loading/error handling, excessive client form complexity.
- `Low`: reusable field cleanup, minor copy/state polish.

## Example Finding

```text
High - features/profile/components/profile-form.tsx:31
The email input has no label and relies on placeholder text.
Why it matters: Placeholder text is not a reliable accessible name and disappears during input, making the form harder to use with assistive technology.
Recommended fix: add a visible label linked with htmlFor/id and associate error text with aria-describedby.
```

