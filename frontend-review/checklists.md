# Checklists

## Universal PR Checklist

- No banned technologies.
- Server Components are default.
- `"use client"` boundaries are justified and small.
- Server Actions validate and authorize.
- Route Handlers are used only for external HTTP surfaces or protocol needs.
- Native `fetch()` is used with error handling.
- Cached data has tags and invalidation.
- User-facing strings use next-intl.
- Forms have labels, errors, loading, disabled, and success states.
- Security-sensitive data stays server-only.
- Tests cover changed risk.

## Architecture Checklist

- Feature ownership is clear.
- Business logic is server-side where appropriate.
- No raw database models leak to clients.
- Abstractions are justified.
- Coupling is directional and reviewable.

## Accessibility Checklist

- Semantic HTML.
- Keyboard navigation.
- Visible focus.
- Programmatic labels.
- Error announcements.
- Dialog/menu focus management.
- WCAG AA contrast.
- Reduced motion.
- RTL and zoom resilience.

## Security Checklist

- No tokens in browser-readable storage.
- Auth and authz at every mutation and sensitive read.
- Zod validation at trust boundaries.
- Webhook signatures verified.
- Secrets not exposed in client bundle.
- Cache privacy reviewed.
- Errors do not leak internals.

## Performance Checklist

- Minimal client JS.
- No avoidable effect-based fetching.
- Suspense/streaming used for slow work.
- Images and fonts optimized.
- Rerender scope controlled.
- Heavy libraries isolated or dynamic.
- Layout shift avoided.

