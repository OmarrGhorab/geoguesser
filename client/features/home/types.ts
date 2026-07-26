import type {
  AuthenticatedGameModeGroupId,
  AuthenticatedGameModeId,
} from "@/features/home/nav-config";

export type AuthenticatedNavCopy = {
  play: string;
  challenges: string;
  maps: string;
  leaderboards: string;
  friends: string;
  home: string;
  profile: string;
  stats: string;
  missions: string;
  settings: string;
};

export type AuthenticatedGameModesCopy = {
  title: string;
  searchLabel: string;
  searchPlaceholder: string;
  empty: string;
  recommended: string;
  viewAll: string;
  escapeHint: string;
  groups: Record<AuthenticatedGameModeGroupId, string>;
  items: Record<
    AuthenticatedGameModeId,
    { title: string; description: string }
  >;
};

export type AuthenticatedPremiumCopy = {
  title: string;
  body: string;
  benefits: string[];
  price: string;
  cta: string;
  sidebarTitle: string;
  sidebarBody: string;
};

export type AuthenticatedAriaCopy = {
  primary: string;
  dashboard: string;
  search: string;
  menu: string;
};

/** Localized chrome + labels only — no usernames, scores, or map entities. */
export type AuthenticatedHomeCopy = {
  brand: string;
  /** Template with {name}, e.g. "Welcome back, {name}!" */
  welcome: string;
  question: string;
  languageLabel: string;
  nav: AuthenticatedNavCopy;
  gameModes: AuthenticatedGameModesCopy;
  premium: AuthenticatedPremiumCopy;
  recommended: string;
  seeAll: string;
  play: Record<string, { title: string; description: string; cta: string }>;
  modes: string;
  modeRows: Array<{ title: string; description: string }>;
  daily: {
    title: string;
    streak: string;
    ends: string;
    cta: string;
    /** Template with {count} */
    playersToday: string;
    days: string[];
  };
  stats: {
    title: string;
    played: string;
    average: string;
    best: string;
    cta: string;
  };
  aria: AuthenticatedAriaCopy;
};

export type AuthenticatedChromeCopy = {
  brand: string;
  nav: AuthenticatedNavCopy;
  gameModes: AuthenticatedGameModesCopy;
  premium: Pick<
    AuthenticatedPremiumCopy,
    "title" | "body" | "cta" | "sidebarTitle" | "sidebarBody"
  >;
  languageLabel: string;
  aria: AuthenticatedAriaCopy;
  /** Display name for navbar (from viewer data, not i18n). */
  viewerName: string;
  viewerAvatarUrl: string | null;
};

export type AuthenticatedSidebarItem =
  "home" | "profile" | "stats" | "friends" | "missions" | "settings";

export type { AuthenticatedHomeData } from "@/features/home/schemas";
