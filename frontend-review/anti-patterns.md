# Anti-Pattern Detection

## Automatically Detect

- Large Client Components.
- Deep prop drilling.
- God components.
- God hooks.
- Nested ternaries.
- Overuse of Context.
- Business logic inside JSX.
- Inline functions causing measurable rerender issues.
- Multiple responsibilities.
- Repeated fetches.
- Duplicate validation.
- State duplication.
- Duplicated constants.
- Massive files.
- Dead code.
- Commented-out code.
- Unused imports.
- Unused state.
- Unused props.
- Memory leaks.
- Missing cleanup.
- Hydration mismatches.

## Severity Defaults

- `Critical`: anti-pattern creates security exposure or broken production flow.
- `High`: anti-pattern harms architecture, performance, accessibility, or state correctness.
- `Medium`: anti-pattern creates maintainability risk or likely regression.
- `Low`: localized cleanup.
- `Suggestion`: optional simplification.

## Review Notes

Do not flag a pattern mechanically when context makes it acceptable. Explain why this instance is harmful.

Example:

```text
Medium - features/search/components/results.tsx:64
The component duplicates filter state in local state and URL searchParams.
Why it matters: Back/forward navigation and shared URLs can show stale filters because there are two sources of truth.
Recommended fix: derive filter state from searchParams and use router navigation to update it.
```

