# Navbar Game Modes Implementation Note

## Goal

Replace the separate Singleplayer, Multiplayer, and Party navbar entries with one
localized Play mega-menu based on the approved
[`navbar-game-modes-redesign.png`](./navbar-game-modes-redesign.png).

The menu exposes the eleven player-facing backend modes and omits legacy aliases
(`private_room`, `ranked`, and `ranked_standard`). The source-of-truth contracts
are `backend/internal/games/state.go`, `backend/internal/matchmaking/model.go`, and
the `GameMode` schema in `backend/openapi/openapi.yaml`.

## Mode Routing

| Player-facing mode | Backend mode | Frontend destination |
| --- | --- | --- |
| Classic Solo | `solo` | `/play?mode=solo` |
| Practice | `practice` | `/play?mode=practice` |
| Daily Challenge | `daily` | `/daily-mission` |
| Quick Play | `quick_play` | `/play?mode=quick_play` |
| Casual Solo | `casual_solo` | `/matchmaking?mode=casual_solo` |
| Casual Duos | `casual_duo` | `/matchmaking?mode=casual_duo` |
| Casual Squads | `casual_squad` | `/matchmaking?mode=casual_squad` |
| Ranked Solo | `ranked_solo` | `/matchmaking?mode=ranked_solo` |
| Ranked Duos | `ranked_duo` | `/matchmaking?mode=ranked_duo` |
| Ranked Squads | `ranked_squad` | `/matchmaking?mode=ranked_squad` |
| Party Lobby | `party_lobby` | `/rooms?mode=party_lobby` |

## UX and Quality Gates

- Closed, open, filtered, and empty-search states are defined.
- Base UI Popover provides focus return, outside-click dismissal, and Escape close.
- All visible and accessible copy is localized in English and Arabic.
- Logical spacing and Base UI positioning support LTR and RTL layouts.
- The interactive client boundary is limited to the game-mode popover.
- The eleven local 72×72 PNG icons total 36,391 bytes and require no runtime API
  request.
- Verification: ESLint, TypeScript, 93 unit tests, production build, English
  browser interaction, Arabic RTL browser inspection, and browser console checks.
