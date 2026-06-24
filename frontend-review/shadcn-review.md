# shadcn Review

## Review Scope

Verify shadcn/ui composition, variants, CVA usage, Radix accessibility, reusable primitives, and generated component integrity.

## Blockers

- Editing generated shadcn components for one feature without justification.
- Replacing Radix behavior with custom inaccessible widgets.
- Removing focus management or accessible labels.
- Building complex widgets from `div` and click handlers.

## What To Check

- Feature-specific styling is done by composition or variants.
- `asChild` and `Slot` behavior is preserved.
- Dialogs, popovers, menus, selects, tabs, and tooltips retain Radix semantics.
- Icon-only buttons have labels.
- Destructive actions use clear confirmation and server authorization.
- CVA variants are coherent and token-based.

## Severity Guidance

- `High`: accessibility regression in primitive, broken focus/keyboard, unsafe destructive UI.
- `Medium`: generated component modified unnecessarily, variant sprawl.
- `Low`: component API naming or styling cleanup.

## Example Finding

```text
High - components/ui/dialog.tsx:42
The DialogContent wrapper removed DialogTitle support and suppresses the accessibility warning.
Why it matters: Screen reader users may enter dialogs without a programmatic title or context.
Recommended fix: restore the shadcn/Radix title pattern and require title/description in feature dialogs.
```

