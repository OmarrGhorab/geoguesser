# Accessibility

## Concepts

Accessibility is a production requirement. Build with semantic HTML, WCAG-aligned color contrast and keyboard behavior, focus management, labels, error messages, reduced motion, and localized directionality. Radix and shadcn provide strong primitives, but implementation choices can still break accessibility.

Why this exists: accessible UI is more usable, more robust, often more performant, and legally required in many contexts.

## Best Practices

- Use semantic elements before ARIA.
- Use buttons for actions and links for navigation.
- Provide accessible names for all controls.
- Preserve focus indicators.
- Use Radix primitives for complex widgets.
- Support keyboard-only operation.
- Honor `prefers-reduced-motion`.
- Test with screen readers and automated checks.

## Anti-Patterns

- Clickable `div`s.
- Placeholder-only labels.
- Removing outlines without replacement.
- Icon-only controls without labels.
- Color-only status indicators.
- Modals without focus management.
- Client transitions that lose focus context.

## Common Mistakes

- `aria-label` that differs from visible meaning.
- Error text not associated with inputs.
- `disabled` controls with no explanation in complex workflows.
- Menus that cannot be closed with Escape.
- RTL layouts with incorrect reading order.
- Animations that ignore reduced motion.

## Production Examples

```tsx
export function FieldError({ id, message }: { id: string; message?: string }) {
  if (!message) return null
  return (
    <p id={id} role="alert" className="text-sm text-destructive">
      {message}
    </p>
  )
}
```

```tsx
export function EmailField({ error }: { error?: string }) {
  return (
    <div>
      <label htmlFor="email">Email</label>
      <input
        id="email"
        name="email"
        type="email"
        autoComplete="email"
        aria-invalid={Boolean(error)}
        aria-describedby={error ? 'email-error' : undefined}
      />
      <FieldError id="email-error" message={error} />
    </div>
  )
}
```

## Folder Organization

```text
components/ui/
features/*/components/
tests/accessibility/
```

Keep accessible behavior in shared primitives, but verify every feature composition.

## TypeScript Examples

```tsx
type IconButtonProps = {
  label: string
  icon: React.ComponentType<{ className?: string; 'aria-hidden'?: boolean }>
  onClick: () => void
}

export function IconButton({ label, icon: Icon, onClick }: IconButtonProps) {
  return (
    <button type="button" aria-label={label} onClick={onClick}>
      <Icon className="size-4" aria-hidden="true" />
    </button>
  )
}
```

## Performance Considerations

- Semantic HTML reduces JavaScript needed for behavior.
- Native controls are faster and more accessible than custom controls.
- Avoid layout shifts that move focus targets.
- Reduce motion variants can also reduce CPU work.
- Do not load large accessibility polyfills instead of using correct markup.

## Security Considerations

- Accessible error messages should not leak sensitive security details.
- Focus traps must not prevent users from escaping security-critical dialogs.
- Do not hide sensitive data only visually.
- CAPTCHA and MFA must have accessible alternatives.
- Session timeout warnings should be perceivable and operable.

## Accessibility Considerations

- Run keyboard, screen reader, reduced motion, zoom, contrast, and RTL checks.
- Test loading, error, empty, disabled, and success states.
- Ensure every page has landmarks and a logical heading order.
- Ensure forms can be completed without a mouse.
- Ensure live updates are announced when context changes.

