# Phase 6 Frontend Architecture

## Purpose

This document defines the frontend architecture before implementing pages. It maps the GeoGuess product, backend API contract, and Next.js 16+ App Router model into a maintainable client structure.

The frontend lives under:

```text
client/
```

## Current Client State

The current client is minimal:

```text
client/
  app/
    layout.tsx
    page.tsx
    globals.css
  components/
  lib/
  public/
```

Current `client/package.json` includes Next.js, React, TypeScript, Tailwind CSS v4, and ESLint. The target architecture also requires adding:

- `next-intl`
- `zustand`
- `react-hook-form`
- `zod`
- `@hookform/resolvers`
- `motion`
- `shadcn/ui` dependencies such as Radix primitives, `class-variance-authority`, `clsx`, `tailwind-merge`, and `lucide-react`
- `pnpm` as the intended package manager

Do not implement pages until this structure is accepted.

## Architecture Principles

- Use Next.js 16+ App Router.
- Use Server Components by default.
- Use Client Components only for map interaction, realtime room UI, timers, dialogs, local controls, and complex forms.
- Use Server Actions for in-app mutations.
- Use native `fetch()` from server-only data modules to the Go API.
- Never fetch server-renderable data inside `useEffect`.
- Never store server data in Zustand.
- Never store auth tokens in Zustand, localStorage, or client props.
- Localize every user-facing string with `next-intl`.
- Support RTL from the route and layout level.
- Use shadcn/ui as the UI primitive foundation.
- Extend shadcn through composition, not generated-file edits.
- Use Tailwind CSS v4 tokens and CSS variables.

## Proposed Frontend Tree

```text
client/
  app/
    [locale]/
      layout.tsx
      loading.tsx
      error.tsx
      not-found.tsx

      (marketing)/
        page.tsx
        about/
          page.tsx
        pricing/
          page.tsx

      (auth)/
        login/
          page.tsx
        register/
          page.tsx

      (game)/
        play/
          page.tsx
        games/
          [gameId]/
            page.tsx
            loading.tsx
            results/
              page.tsx
        rooms/
          page.tsx
          [roomCode]/
            page.tsx
            loading.tsx
        matchmaking/
          page.tsx

      (app)/
        profile/
          page.tsx
        leaderboard/
          page.tsx
        maps/
          page.tsx
          [mapId]/
            page.tsx
        friends/
          page.tsx
        achievements/
          page.tsx
        billing/
          page.tsx

    globals.css
    favicon.ico

  components/
    ui/
    layout/
    feedback/
    seo/

  features/
    auth/
      actions.ts
      data.ts
      schemas.ts
      types.ts
      components/
    profile/
      actions.ts
      data.ts
      schemas.ts
      types.ts
      components/
    maps/
      data.ts
      schemas.ts
      types.ts
      components/
    game/
      actions.ts
      data.ts
      schemas.ts
      types.ts
      components/
      stores/
    rooms/
      actions.ts
      data.ts
      realtime.ts
      schemas.ts
      types.ts
      components/
      stores/
    matchmaking/
      actions.ts
      data.ts
      schemas.ts
      types.ts
      components/
    leaderboard/
      data.ts
      schemas.ts
      types.ts
      components/
    friends/
      actions.ts
      data.ts
      schemas.ts
      types.ts
      components/
    achievements/
      data.ts
      schemas.ts
      types.ts
      components/
    billing/
      actions.ts
      data.ts
      schemas.ts
      types.ts
      components/
    ads/
      data.ts
      types.ts
      components/

  lib/
    api/
      client.ts
      errors.ts
      schemas.ts
    auth/
      session.ts
    i18n/
      routing.ts
      request.ts
      direction.ts
    env.ts
    utils.ts

  stores/
    theme-store.ts
    modal-store.ts
    sidebar-store.ts
    preferences-store.ts

  messages/
    en.json
    ar.json

  public/
```

## Route Map

All user-facing routes should be locale-prefixed:

```text
/:locale
/:locale/login
/:locale/register
/:locale/play
/:locale/games/:gameId
/:locale/games/:gameId/results
/:locale/rooms
/:locale/rooms/:roomCode
/:locale/matchmaking
/:locale/profile
/:locale/leaderboard
/:locale/maps
/:locale/maps/:mapId
/:locale/friends
/:locale/achievements
/:locale/billing
/:locale/about
/:locale/pricing
```

### Route Groups

| Group | URL Effect | Purpose |
| --- | --- | --- |
| `[locale]` | Adds locale prefix | Owns locale validation, `lang`, `dir`, message loading. |
| `(marketing)` | None | Public landing, about, pricing, SEO-focused pages. |
| `(auth)` | None | Login and registration shell. |
| `(game)` | None | Full-screen gameplay routes with minimal chrome. |
| `(app)` | None | Authenticated app shell for profile, maps, leaderboards, friends, billing. |

Route groups do not provide security. Pages, Server Actions, and data functions must still authorize.

## Layout Plan

### Root Locale Layout

File:

```text
app/[locale]/layout.tsx
```

Responsibilities:

- Validate locale.
- Load `next-intl` messages.
- Set `<html lang>` and `dir`.
- Include global providers as deep as possible.
- Include global CSS.
- Provide metadata defaults.

Do not turn this layout into a broad Client Component.

### Marketing Layout

File:

```text
app/[locale]/(marketing)/layout.tsx
```

Responsibilities:

- Public navigation.
- Footer.
- SEO-friendly static shell.
- Cached map/game marketing summaries where safe.

### Auth Layout

File:

```text
app/[locale]/(auth)/layout.tsx
```

Responsibilities:

- Minimal centered auth shell.
- Redirect already-authenticated users from login/register when appropriate.
- Avoid loading the full app navigation.

### Game Layout

File:

```text
app/[locale]/(game)/layout.tsx
```

Responsibilities:

- Full-viewport gameplay surface.
- Minimal header or overlay controls.
- Keep map/panorama Client Components scoped below this layout.
- Avoid heavy global providers.

### App Layout

File:

```text
app/[locale]/(app)/layout.tsx
```

Responsibilities:

- Authenticated app shell.
- Header/sidebar.
- Profile/menu controls.
- Suspense boundaries around session-dependent UI.
- Server-side session checks for protected pages.

## Page Ownership

Route files should stay thin.

Example:

```text
app/[locale]/(game)/games/[gameId]/page.tsx
```

Should:

- Parse `params` as a Promise.
- Call `features/game/data.ts`.
- Compose Server Components and Client islands.
- Render loading/error/not-found boundaries.

Should not:

- Contain scoring rules.
- Open WebSocket connections directly.
- Build large interactive UI inline.
- Call Route Handlers for internal data.

## Server Components And Client Islands

### Server Component Default

Use Server Components for:

- Pages and layouts.
- Map list pages.
- Leaderboards.
- Profile summary.
- Game result screens.
- Static/cached marketing content.
- Data formatting where locale is known.

### Client Component Islands

Use Client Components for:

- Guess map with pin placement.
- Panorama/street-view controls.
- Round timer.
- Room lobby realtime player list.
- WebSocket connection manager.
- Dialogs, popovers, menus, tabs, tooltips.
- Complex forms needing React Hook Form.
- Motion-powered transitions.

Client islands should receive small DTOs, not raw API responses.

## Feature Folder Rules

Each feature follows this shape:

```text
features/{feature}/
  actions.ts
  data.ts
  schemas.ts
  types.ts
  components/
  stores/
```

### `data.ts`

Server-only read functions.

Rules:

- Include `import 'server-only'`.
- Use native `fetch()`.
- Validate responses with Zod.
- Use Cache Components only when data is safe to cache.
- Never import from Client Components.

### `actions.ts`

Server Actions for app mutations.

Rules:

- Start file with `'use server'`.
- Validate input with Zod.
- Call Go API using server-side fetch.
- Authorize inside action or rely on backend plus session check where appropriate.
- Invalidate precise cache tags after mutations.
- Return serializable form state.

### `schemas.ts`

Zod schemas for:

- Form input.
- Search params.
- API responses.
- Action state.

### `types.ts`

Feature-level DTO and UI types.

Prefer deriving from Zod when practical.

### `components/`

Feature UI components.

Rules:

- Server Components by default.
- Add `"use client"` only to the leaf components requiring browser interaction.
- Localize labels and text in the component that owns the UI.

### `stores/`

Feature-local Zustand stores only for UI state.

Example:

```text
features/game/stores/guess-map-store.ts
features/rooms/stores/room-panel-store.ts
```

Do not store game state, room state, leaderboard rows, user session, guesses, or API results in Zustand.

## Shared Components

### `components/ui/`

shadcn/ui generated primitives:

- Button
- Dialog
- AlertDialog
- Tooltip
- DropdownMenu
- Tabs
- Form primitives
- Input
- Select
- Toast/Sonner if chosen

Rules:

- Do not put product-specific components here.
- Do not modify generated components for one feature.
- Compose wrappers in feature folders.

### `components/layout/`

Shared layout pieces:

- AppHeader
- AppSidebar
- MarketingHeader
- LocaleSwitcher
- UserMenu
- MobileNavigation

### `components/feedback/`

Reusable states:

- EmptyState
- ErrorState
- LoadingSkeleton
- ResultBadge
- ScoreDelta

### `components/seo/`

SEO helpers:

- StructuredData
- OpenGraph helpers
- Breadcrumb JSON-LD later

## Zustand Store Plan

Global stores belong in `client/stores`.

Allowed global stores:

| Store | Purpose |
| --- | --- |
| `theme-store.ts` | Non-sensitive theme preference and system/manual mode. |
| `modal-store.ts` | Global modal state for UI-only dialogs. |
| `sidebar-store.ts` | App sidebar collapsed/open state. |
| `preferences-store.ts` | Non-sensitive UI preferences such as units, reduced visual noise, map UI toggles. |

Feature-local stores:

| Store | Purpose |
| --- | --- |
| `features/game/stores/guess-map-store.ts` | Current unsent map pin, zoom, pan mode, local map UI state. |
| `features/game/stores/round-ui-store.ts` | Local timer display mode, result panel open state. |
| `features/rooms/stores/room-ui-store.ts` | Lobby panel state, invite dialog visibility, local ready UI. |

Forbidden in Zustand:

- Auth tokens.
- Refresh tokens.
- User permissions.
- Game records from the API.
- Current room server state.
- Leaderboard data.
- Cache data.
- Payment/subscription state.

## Forms Plan

Use native forms plus Server Actions by default.

### Native Server Action Forms

Use for:

- Login.
- Register.
- Logout.
- Create room.
- Join room.
- Start solo game.
- Enter matchmaking.
- Update profile simple fields.

Pattern:

```text
features/auth/actions.ts
features/auth/schemas.ts
features/auth/components/login-form.tsx
```

### React Hook Form + Zod Forms

Use only when the UI needs rich client behavior:

- Room settings with conditional timer/max-player controls.
- Profile editor with avatar preview.
- Future custom map creation.
- Billing checkout plan selection if dynamic.

Server validation remains authoritative even when client validation exists.

### Form Accessibility Rules

- Every input has a label.
- Error text is associated with fields.
- Form-level errors use `role="alert"` or `aria-live`.
- Buttons show pending state without losing accessible names.
- No placeholder-only labels.

## Localization Plan

Use `next-intl`.

### Locales

Initial locales:

```text
en
ar
```

Why include Arabic early:

- The user timezone/context is Egypt.
- It forces RTL-safe layout decisions from the beginning.

### Files

```text
messages/
  en.json
  ar.json
lib/i18n/
  routing.ts
  request.ts
  direction.ts
app/[locale]/layout.tsx
```

### Rules

- Never hardcode user-facing strings.
- Localize metadata titles/descriptions.
- Localize form labels, placeholders, descriptions, errors, aria labels, empty states, and loading states.
- Use ICU messages for pluralization and dynamic score text.
- Validate locale params against an allowlist.
- Set `dir="rtl"` for Arabic.
- Use CSS logical properties where route/UI direction matters.

### Message Namespace Plan

```text
Common
Navigation
Marketing
Auth
Profile
Maps
Game
Rooms
Matchmaking
Leaderboard
Friends
Achievements
Billing
Errors
Forms
Accessibility
```

## API Data Layer

Use server-only API helpers:

```text
lib/api/client.ts
lib/api/errors.ts
lib/api/schemas.ts
```

Responsibilities:

- Build backend API URLs from validated environment config.
- Forward cookies from Server Components and Server Actions when needed.
- Use native `fetch()`.
- Validate JSON with Zod.
- Normalize OpenAPI error envelopes.
- Apply request timeouts with `AbortSignal`.

Feature `data.ts` modules call `lib/api/client.ts`; pages call feature data functions.

Do not call `app/api` Route Handlers from Server Components for internal data.

## Cache And Streaming Plan

Use Cache Components deliberately.

Cache candidates:

- Public map list.
- Public map metadata.
- Public leaderboard snapshots with short life.
- Marketing/static content.

Do not cache:

- Current round state.
- Current room state.
- Auth session data beyond request memoization.
- Billing entitlements without explicit invalidation.
- Hidden location coordinates in client-readable caches.

Use Suspense for:

- Current user shell.
- Leaderboards.
- Game results.
- Room state.
- Matchmaking status.

Use `loading.tsx` for route-level loading and component-level skeletons for important partial data.

## Realtime Frontend Plan

Realtime belongs under:

```text
features/rooms/realtime.ts
features/rooms/components/room-events-provider.tsx
features/game/components/realtime-round-status.tsx
```

Rules:

- WebSocket/SSE setup must live in Client Components.
- Server commands still go through Server Actions or backend HTTP endpoints.
- Realtime events update local UI state, not authoritative server data.
- On reconnect, refetch room/game state from the server.
- Do not trust realtime payloads for final scoring.

## Route-To-Feature Mapping

| Route | Feature Modules |
| --- | --- |
| `/:locale` | `features/maps`, `features/leaderboard`, marketing components |
| `/:locale/login` | `features/auth` |
| `/:locale/register` | `features/auth` |
| `/:locale/play` | `features/game`, `features/maps` |
| `/:locale/games/:gameId` | `features/game` |
| `/:locale/games/:gameId/results` | `features/game`, `features/leaderboard`, `features/achievements` |
| `/:locale/rooms` | `features/rooms`, `features/maps` |
| `/:locale/rooms/:roomCode` | `features/rooms`, `features/game`, `features/realtime` |
| `/:locale/matchmaking` | `features/matchmaking`, `features/rooms` |
| `/:locale/profile` | `features/profile`, `features/auth` |
| `/:locale/leaderboard` | `features/leaderboard` |
| `/:locale/maps` | `features/maps` |
| `/:locale/maps/:mapId` | `features/maps`, `features/leaderboard` |
| `/:locale/friends` | `features/friends` |
| `/:locale/achievements` | `features/achievements` |
| `/:locale/billing` | `features/billing` |

## MVP Frontend Implementation Order

1. Add frontend dependencies and tooling alignment:
   - pnpm
   - next-intl
   - shadcn/ui
   - Radix primitives
   - lucide-react
   - Zustand
   - React Hook Form
   - Zod
   - Motion

2. Set up app shell:
   - `[locale]` route segment.
   - message files.
   - locale routing.
   - `lang` and `dir`.
   - global design tokens.

3. Set up API layer:
   - environment validation.
   - fetch helper.
   - API error normalization.
   - OpenAPI-aligned schemas.

4. Build auth UI:
   - login.
   - register.
   - current session display.

5. Build map/game start:
   - map list.
   - solo start form.
   - game route shell.

6. Build core gameplay:
   - round media viewer.
   - guess map Client Component.
   - submit guess action.
   - result panel.

7. Build game results:
   - final score.
   - round breakdown.
   - replay/share actions.

8. Build private rooms:
   - create room.
   - join room.
   - lobby.
   - realtime player list.

9. Build leaderboard/profile:
   - public leaderboard.
   - profile stats.

10. Defer:
   - billing.
   - ads.
   - friends.
   - achievements.
   - matchmaking if not in MVP.

## Security Rules

- Keep backend base URL and secrets server-side.
- Use HTTP-only cookies for auth.
- Do not store tokens in localStorage or Zustand.
- Validate every Server Action input with Zod.
- Treat Server Actions as public POST endpoints.
- Recheck authorization in backend and action layer where relevant.
- Do not expose true coordinates before guess submission or timeout.
- Do not render unauthorized data and hide it with CSS.

## Performance Rules

- Keep gameplay Client Components small and isolated.
- Avoid making route layouts Client Components.
- Use Server Components for data-heavy pages.
- Stream current room/game state behind Suspense.
- Cache stable public map and leaderboard reads.
- Avoid duplicate backend fetches.
- Load heavy map/panorama libraries only on game routes.
- Respect reduced motion for animations.

## Accessibility Rules

- Every page has a clear `h1`.
- Game controls are keyboard reachable where practical.
- Guess submission and timers need accessible status text.
- Do not communicate score solely through color.
- Dialogs, menus, popovers, and tooltips use Radix/shadcn primitives.
- Localize aria labels and alt text.
- Test RTL for navigation, map controls, and room lobby.

## Anti-Patterns To Reject

- React Router.
- Pages Router.
- Axios.
- React Query.
- Redux Toolkit.
- Client-side `useEffect` fetching for server-renderable data.
- Storing API data in Zustand.
- Hardcoded visible strings.
- One giant Client Component for the gameplay page.
- Editing shadcn generated components for one feature.
- Putting all product components in `components/`.

## Open Questions

- Will the MVP include Arabic at launch or only prepare for it?
- Which map/panorama library will be used for the guessing map and imagery?
- Will game routes use WebSockets immediately or start with polling/SSE?
- Should guests be allowed to create rooms or only join them?
- Should billing routes exist in navigation before monetization launches?

## Phase 6 Exit Criteria

Phase 6 is ready for implementation when:

- Route map is accepted.
- Layout groups are accepted.
- Feature folder boundaries are accepted.
- Zustand store rules are accepted.
- Forms strategy is accepted.
- Localization strategy is accepted.
- MVP frontend implementation order is accepted.
