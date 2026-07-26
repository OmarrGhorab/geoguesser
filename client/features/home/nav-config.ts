/**
 * Top navbar items derived from backend domains wired in
 * `backend/internal/app/routes.go`.
 *
 * Only labels for implemented API surfaces appear here.
 * Paths are locale-prefixed by the navbar (e.g. `/${locale}/play`).
 */
export const AUTHENTICATED_TOP_NAV = [
  {
    id: "play",
    /** Games, matchmaking, and hosted rooms are exposed through one mode picker. */
    backends: ["games", "matchmaking", "rooms"],
    href: "/play",
    hasDropdown: true,
  },
  {
    id: "challenges",
    /** /api/v1/challenges/* — daily, shared, streaks, missions */
    backends: ["challenges"],
    href: "/challenges",
    hasDropdown: false,
  },
  {
    id: "maps",
    /** GET /api/v1/maps, /api/v1/maps/{id} */
    backends: ["maps"],
    href: "/maps",
    hasDropdown: false,
  },
  {
    id: "leaderboards",
    /** GET /api/v1/leaderboards/* */
    backends: ["leaderboards"],
    href: "/leaderboard",
    hasDropdown: false,
  },
  {
    id: "friends",
    /** /api/v1/friends/* social graph */
    backends: ["friends"],
    href: "/friends",
    hasDropdown: false,
  },
] as const;

export type AuthenticatedTopNavId =
  (typeof AUTHENTICATED_TOP_NAV)[number]["id"];

export const AUTHENTICATED_GAME_MODE_GROUPS = [
  { id: "soloAdventures", columns: 3 },
  { id: "quickMatch", columns: 4 },
  { id: "competitive", columns: 3 },
  { id: "withFriends", columns: 1 },
] as const;

export type AuthenticatedGameModeGroupId =
  (typeof AUTHENTICATED_GAME_MODE_GROUPS)[number]["id"];

/** Canonical player-facing modes accepted by the backend today. */
export const AUTHENTICATED_GAME_MODES = [
  {
    id: "classicSolo",
    backendMode: "solo",
    group: "soloAdventures",
    href: "/play?mode=solo",
    asset: "/authenticated-home/game-modes/classic-solo.png",
    tone: "solo",
  },
  {
    id: "practice",
    backendMode: "practice",
    group: "soloAdventures",
    href: "/play?mode=practice",
    asset: "/authenticated-home/game-modes/practice.png",
    tone: "solo",
  },
  {
    id: "dailyChallenge",
    backendMode: "daily",
    group: "soloAdventures",
    href: "/daily-mission",
    asset: "/authenticated-home/game-modes/daily-challenge.png",
    tone: "solo",
  },
  {
    id: "quickPlay",
    backendMode: "quick_play",
    group: "quickMatch",
    href: "/play?mode=quick_play",
    asset: "/authenticated-home/game-modes/quick-play.png",
    tone: "casual",
    recommended: true,
  },
  {
    id: "casualSolo",
    backendMode: "casual_solo",
    group: "quickMatch",
    href: "/matchmaking?mode=casual_solo",
    asset: "/authenticated-home/game-modes/casual-solo.png",
    tone: "casual",
  },
  {
    id: "casualDuos",
    backendMode: "casual_duo",
    group: "quickMatch",
    href: "/matchmaking?mode=casual_duo",
    asset: "/authenticated-home/game-modes/casual-duos.png",
    tone: "casual",
  },
  {
    id: "casualSquads",
    backendMode: "casual_squad",
    group: "quickMatch",
    href: "/matchmaking?mode=casual_squad",
    asset: "/authenticated-home/game-modes/casual-squads.png",
    tone: "casual",
  },
  {
    id: "rankedSolo",
    backendMode: "ranked_solo",
    group: "competitive",
    href: "/matchmaking?mode=ranked_solo",
    asset: "/authenticated-home/game-modes/ranked-solo.png",
    tone: "ranked",
  },
  {
    id: "rankedDuos",
    backendMode: "ranked_duo",
    group: "competitive",
    href: "/matchmaking?mode=ranked_duo",
    asset: "/authenticated-home/game-modes/ranked-duos.png",
    tone: "ranked",
  },
  {
    id: "rankedSquads",
    backendMode: "ranked_squad",
    group: "competitive",
    href: "/matchmaking?mode=ranked_squad",
    asset: "/authenticated-home/game-modes/ranked-squads.png",
    tone: "ranked",
  },
  {
    id: "partyLobby",
    backendMode: "party_lobby",
    group: "withFriends",
    href: "/rooms?mode=party_lobby",
    asset: "/authenticated-home/game-modes/party-lobby.png",
    tone: "party",
  },
] as const;

export type AuthenticatedGameModeId =
  (typeof AUTHENTICATED_GAME_MODES)[number]["id"];

/** Left sidebar items mapped to backend profile/social surfaces. */
export const AUTHENTICATED_SIDE_NAV = [
  { id: "home", href: "/", backend: "home" },
  { id: "profile", href: "/profile", backend: "profiles" },
  { id: "stats", href: "/profile", backend: "profiles" },
  { id: "friends", href: "/friends", backend: "friends" },
  /** Missions claim under challenges — closest to “badges” */
  { id: "missions", href: "/missions", backend: "challenges" },
  { id: "settings", href: "/profile", backend: "profiles" },
] as const;

export type AuthenticatedSideNavId =
  (typeof AUTHENTICATED_SIDE_NAV)[number]["id"];
