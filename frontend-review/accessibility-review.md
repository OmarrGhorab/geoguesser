# Accessibility Review

## Review Scope

Review semantic HTML, keyboard navigation, labels, ARIA, focus management, contrast, reduced motion, screen readers, dialogs, forms, menus, and WCAG AA compliance.

## Blockers

- Critical flow unusable with keyboard.
- Missing labels on required inputs.
- Dialog/menu focus trap broken.
- Icon-only controls without accessible names.
- Content or controls hidden from assistive tech incorrectly.
- Motion that ignores reduced-motion preferences in critical UI.

## What To Check

- Use semantic elements before ARIA.
- Buttons are actions; links are navigation.
- Heading order is logical.
- Forms have labels and associated errors.
- Focus states are visible.
- Focus is managed in dialogs and route-level recovery UI.
- Color contrast meets WCAG AA.
- Reduced motion is honored.
- RTL and zoom do not break layout.

## Severity Guidance

- `Critical`: blocks core task for keyboard/screen reader users.
- `High`: violates WCAG AA in important workflow.
- `Medium`: incomplete announcements, poor focus, noncritical form issues.
- `Low`: minor semantic improvements.

## Example Finding

```text
High - features/navigation/mobile-menu.tsx:27
The menu trigger is a clickable div with no keyboard support or accessible name.
Why it matters: Keyboard and screen reader users cannot reliably open navigation.
Recommended fix: use a button or shadcn/Radix menu trigger with aria-expanded and a localized label.
```

