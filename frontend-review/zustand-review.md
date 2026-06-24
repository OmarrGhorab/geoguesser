# Zustand Review

## Review Scope

Ensure Zustand is used only for lightweight global client UI state: theme, sidebar, modal, preferences, command menu, and other ephemeral UI coordination.

## Blockers

- Zustand used as an API cache.
- Zustand used for server state or database records.
- Auth tokens, sessions, or permissions stored in Zustand.
- Business logic or mutations embedded in stores.
- Store imported by Server Components.

## What To Check

- Local state is preferred for component-local interaction.
- Selectors are used to avoid broad rerenders.
- Stores are small and scoped.
- Persistence is limited to non-sensitive preferences.
- URL state is not duplicated in a store.
- Server state remains in Server Components, data functions, and cache tags.

## Severity Guidance

- `Critical`: token/session storage in Zustand.
- `High`: server state cache, cross-tenant data, or permission decisions in store.
- `Medium`: oversized store, excessive rerenders, duplicated URL state.
- `Low`: naming or selector cleanup.

## Example Finding

```text
Critical - stores/auth-store.ts:8
The store persists an accessToken in localStorage through Zustand middleware.
Why it matters: Browser-readable tokens are exposed to XSS and cannot be protected with HTTP-only cookie controls.
Recommended fix: move session storage to secure HTTP-only cookies and read session state on the server.
```

