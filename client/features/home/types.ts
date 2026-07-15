export type AuthenticatedNavCopy = {
  /** Top bar — backend-backed */
  singleplayer: string;
  multiplayer: string;
  party: string;
  challenges: string;
  maps: string;
  leaderboards: string;
  friends: string;
  /** Left sidebar */
  home: string;
  profile: string;
  stats: string;
  missions: string;
  settings: string;
};

export type AuthenticatedProfileCopy = {
  name: string;
  level: string;
  credits: string;
};

export type AuthenticatedPremiumCopy = {
  title: string;
  body: string;
  benefits: string[];
  price: string;
  cta: string;
  /** Compact sidebar promo */
  sidebarTitle: string;
  sidebarBody: string;
};

export type AuthenticatedAriaCopy = {
  primary: string;
  dashboard: string;
  search: string;
  menu: string;
};

/** Shared chrome props used by navbar + sidebar shell. */
export type AuthenticatedChromeCopy = {
  brand: string;
  nav: AuthenticatedNavCopy;
  premium: Pick<
    AuthenticatedPremiumCopy,
    "title" | "body" | "cta" | "sidebarTitle" | "sidebarBody"
  >;
  profile: AuthenticatedProfileCopy;
  languageLabel: string;
  aria: AuthenticatedAriaCopy;
};

export type AuthenticatedHomeCopy = AuthenticatedChromeCopy & {
  welcome: string;
  question: string;
  play: Record<string, { title: string; description: string; cta: string }>;
  premium: AuthenticatedPremiumCopy;
  recommended: string;
  seeAll: string;
  maps: Array<{ title: string; difficulty: string; players: string }>;
  modes: string;
  modeRows: Array<{ title: string; description: string; players: string }>;
  daily: {
    title: string;
    streak: string;
    month: string;
    days: string[];
    ends: string;
    cta: string;
    playersToday: string;
  };
  stats: {
    title: string;
    played: string;
    streak: string;
    best: string;
    cta: string;
  };
  friendsOnline: string;
  friends: Array<{ name: string; status: string }>;
};

export type AuthenticatedSidebarItem =
  | "home"
  | "profile"
  | "stats"
  | "friends"
  | "missions"
  | "settings";
