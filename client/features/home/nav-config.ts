/**
 * Top navbar items derived from backend domains wired in
 * `backend/internal/app/routes.go`.
 *
 * Only labels for implemented API surfaces appear here.
 * Paths are locale-prefixed by the navbar (e.g. `/${locale}/play`).
 */
export const AUTHENTICATED_TOP_NAV = [
  {
    id: "singleplayer",
    /** POST /api/v1/games — solo game loop */
    backend: "games",
    href: "/play",
    hasDropdown: true,
  },
  {
    id: "multiplayer",
    /** POST /api/v1/matchmaking/queue + rooms multiplayer */
    backend: "matchmaking",
    href: "/matchmaking",
    hasDropdown: true,
  },
  {
    id: "party",
    /** POST /api/v1/rooms + realtime /realtime/rooms/{code} */
    backend: "rooms",
    href: "/rooms",
    hasDropdown: true,
  },
  {
    id: "challenges",
    /** /api/v1/challenges/* — daily, shared, streaks, missions */
    backend: "challenges",
    href: "/challenges",
    hasDropdown: false,
  },
  {
    id: "maps",
    /** GET /api/v1/maps, /api/v1/maps/{id} */
    backend: "maps",
    href: "/maps",
    hasDropdown: false,
  },
  {
    id: "leaderboards",
    /** GET /api/v1/leaderboards/* */
    backend: "leaderboards",
    href: "/leaderboard",
    hasDropdown: false,
  },
  {
    id: "friends",
    /** /api/v1/friends/* social graph */
    backend: "friends",
    href: "/friends",
    hasDropdown: false,
  },
] as const;

export type AuthenticatedTopNavId = (typeof AUTHENTICATED_TOP_NAV)[number]["id"];

/** Left sidebar items mapped to backend profile/social surfaces. */
export const AUTHENTICATED_SIDE_NAV = [
  { id: "home", href: "/", backend: "home" },
  { id: "profile", href: "/profile", backend: "profiles" },
  { id: "stats", href: "/profile", backend: "profiles" },
  { id: "friends", href: "/friends", backend: "friends" },
  /** Missions claim under challenges — closest to “badges” */
  { id: "missions", href: "/challenges", backend: "challenges" },
  { id: "settings", href: "/profile", backend: "profiles" },
] as const;

export type AuthenticatedSideNavId =
  (typeof AUTHENTICATED_SIDE_NAV)[number]["id"];
