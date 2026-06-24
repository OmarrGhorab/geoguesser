# Code Quality Review

## Review Scope

Review naming, readability, maintainability, duplication, complexity, function length, component size, hook size, file size, early returns, composition, reusable utilities, magic numbers, magic strings, error handling, and logging.

## Blockers

- God components or hooks with multiple responsibilities.
- Complex nested ternaries or conditionals that hide business logic.
- Duplicated validation, constants, or fetch logic.
- Dead code, commented-out code, unused imports/state/props.
- Error handling that swallows failures silently.

## What To Check

- Names describe domain intent.
- Components are cohesive and reasonably sized.
- Hooks do one thing.
- Early returns simplify control flow.
- Magic numbers/strings are constants, config, or localized messages.
- Errors are handled at the right layer.
- Logs do not include secrets and are useful.
- Utilities are extracted only when reuse is real.

## Severity Guidance

- `High`: code quality hides security/logic bug or makes critical flow unmaintainable.
- `Medium`: significant duplication or complexity.
- `Low`: naming, local readability, dead code cleanup.
- `Suggestion`: optional refactor with clear upside.

## Example Finding

```text
Medium - features/editor/use-editor-state.ts:1
The hook manages selection, persistence, keyboard shortcuts, network saves, and toast messages.
Why it matters: Multiple responsibilities make it hard to test and increase regression risk.
Recommended fix: split persistence into a Server Action/data helper and keep the hook focused on UI selection state.
```

