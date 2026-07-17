/**
 * Structural contract for authenticated home main + right rail
 * (home-loggedin.png). Used by components and unit tests so section
 * order cannot drift silently from the design.
 */

export const PLAY_MODE_KEYS = [
  "singleplayer",
  "multiplayer",
  "party",
  "quiz",
] as const;

export const MAIN_SECTION_ORDER = [
  "welcome",
  "playModes",
  "premiumBanner",
  "recommended",
  "gameModes",
] as const;

export const RIGHT_RAIL_SECTION_ORDER = [
  "dailyChallenge",
  "yourStats",
  "friendsOnline",
] as const;

export const RECOMMENDED_MAP_KEYS = [
  "world",
  "famous",
  "usa",
  "europe",
] as const;

export const GAME_MODE_KEYS = ["battle", "duels", "team"] as const;

export const DAILY_WEEK_DAYS = [13, 14, 15, 16, 17, 18, 19] as const;
export const DAILY_TODAY = 14;

/** English landmark strings used by e2e/smoke (locale `en`). */
export const EN_LANDMARKS = {
  welcome: "Welcome back, Radiant!",
  playModes: ["Singleplayer", "Multiplayer", "Party", "Quiz"] as const,
  premium: "Subscribe to play without limits!",
  recommended: "Recommended for you",
  gameModes: "Game Modes",
  daily: "Daily Challenge",
  stats: "Your Stats",
  friends: "Friends Online",
} as const;

export function assertMainSectionContract() {
  return {
    playModeCount: PLAY_MODE_KEYS.length,
    mainSections: [...MAIN_SECTION_ORDER],
    mapCount: RECOMMENDED_MAP_KEYS.length,
    gameModeCount: GAME_MODE_KEYS.length,
  };
}

export function assertRightRailContract() {
  return {
    railSections: [...RIGHT_RAIL_SECTION_ORDER],
    weekDays: [...DAILY_WEEK_DAYS],
    today: DAILY_TODAY,
  };
}
