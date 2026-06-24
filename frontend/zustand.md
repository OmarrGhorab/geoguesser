# Zustand

## Concepts

Use Zustand only for global client UI state: theme UI state, sidebar open state, command menu, modal state, ephemeral preferences, and cross-island UI coordination. Never use Zustand for server state, request data, caches, database records, or authentication tokens.

Why this exists: Next.js already owns server data, caching, revalidation, and navigation. Zustand is useful for client interactivity that cannot live naturally in local component state.

## Best Practices

- Prefer local `useState` before Zustand.
- Create small scoped stores for UI concerns.
- Keep stores client-only.
- Avoid global stores that can leak across requests.
- Use selectors to limit rerenders.
- Persist only non-sensitive preferences when necessary.
- Hydrate persisted UI state carefully to avoid mismatches.

## Anti-Patterns

- Storing fetched server data in Zustand.
- Using Zustand as a React Query replacement.
- Storing JWTs, sessions, or permissions in Zustand.
- Creating one app-wide mega store.
- Mirroring URL state in a store.
- Reading Zustand from Server Components.

## Common Mistakes

- Putting store files in modules imported by Server Components.
- Persisting default values that conflict with server-rendered markup.
- Not using selectors, causing broad rerenders.
- Mixing business mutations into UI stores.
- Treating store state as authoritative for authorization.
- Forgetting to reset modal or wizard state on route changes when needed.

## Production Examples

```ts
// stores/sidebar-store.ts
'use client'

import { create } from 'zustand'

type SidebarState = {
  open: boolean
  setOpen: (open: boolean) => void
  toggle: () => void
}

export const useSidebarStore = create<SidebarState>((set) => ({
  open: true,
  setOpen: (open) => set({ open }),
  toggle: () => set((state) => ({ open: !state.open })),
}))
```

```tsx
'use client'

import { useSidebarStore } from '@/stores/sidebar-store'

export function SidebarToggle() {
  const open = useSidebarStore((state) => state.open)
  const toggle = useSidebarStore((state) => state.toggle)

  return (
    <button type="button" aria-expanded={open} onClick={toggle}>
      Toggle sidebar
    </button>
  )
}
```

## Folder Organization

```text
stores/
  sidebar-store.ts
  modal-store.ts
features/editor/stores/
  toolbar-store.ts
```

Use global `stores/` only for app-wide UI. Feature-specific UI stores belong in the feature.

## TypeScript Examples

```ts
type ModalState =
  | { kind: 'closed' }
  | { kind: 'delete-project'; projectId: string }
  | { kind: 'invite-user'; teamId: string }

type ModalStore = {
  modal: ModalState
  openModal: (modal: Exclude<ModalState, { kind: 'closed' }>) => void
  closeModal: () => void
}
```

## Performance Considerations

- Use selectors per field or action.
- Split unrelated stores.
- Avoid storing large objects or server lists.
- Keep derived data in selectors or server data functions, not duplicated state.
- Use local state for component-only interaction.

## Security Considerations

- Never persist secrets, tokens, sessions, or authorization state.
- Treat client state as user-controlled.
- Recheck permissions on every Server Action and Route Handler.
- Do not hide sensitive data only by closing a modal or sidebar.
- Avoid exposing internal IDs in UI state unless needed.

## Accessibility Considerations

- Store UI state should drive ARIA state such as `aria-expanded`.
- Modal stores must coordinate focus return and escape behavior through Radix/shadcn primitives.
- Persisted preferences should not override user accessibility settings.
- Sidebar collapse must preserve keyboard access.
- Announce global UI state changes when they affect context.

