import type { Route } from "next";
import type { AppLocale } from "@/lib/i18n/routing";
import type { CanonicalMode } from "./modes";

export {
  dailyMissionPlayHref,
  dailyMissionResultsHref,
} from "@/features/mission/routes";

export function dailyMissionHref(locale: AppLocale): Route {
  return `/${locale}/daily-mission` as Route;
}

export function playHref(locale: AppLocale, mode?: CanonicalMode): Route {
  if (mode) {
    return `/${locale}/play?mode=${encodeURIComponent(mode)}` as Route;
  }
  return `/${locale}/play` as Route;
}

export function gameHref(locale: AppLocale, gameId: string): Route {
  return `/${locale}/games/${encodeURIComponent(gameId)}` as Route;
}

export function gameResultsHref(locale: AppLocale, gameId: string): Route {
  return `/${locale}/games/${encodeURIComponent(gameId)}/results` as Route;
}

export function roomsHref(locale: AppLocale, mode?: CanonicalMode): Route {
  if (mode) {
    return `/${locale}/rooms?mode=${encodeURIComponent(mode)}` as Route;
  }
  return `/${locale}/rooms` as Route;
}

export function roomHref(locale: AppLocale, roomCode: string): Route {
  return `/${locale}/rooms/${encodeURIComponent(roomCode)}` as Route;
}

export function matchmakingHref(
  locale: AppLocale,
  mode?: CanonicalMode,
): Route {
  if (mode) {
    return `/${locale}/matchmaking?mode=${encodeURIComponent(mode)}` as Route;
  }
  return `/${locale}/matchmaking` as Route;
}

export function partyHref(locale: AppLocale, partyId: string): Route {
  return `/${locale}/parties/${encodeURIComponent(partyId)}` as Route;
}

export function matchHref(locale: AppLocale, matchId: string): Route {
  return `/${locale}/matches/${encodeURIComponent(matchId)}` as Route;
}

export function matchResultsHref(locale: AppLocale, matchId: string): Route {
  return `/${locale}/matches/${encodeURIComponent(matchId)}/results` as Route;
}

/**
 * Locale-prefixes a backend-supplied relative destination when it matches a
 * known in-app path. Rejects absolute, protocol-relative, and unknown paths
 * so assignment redirects cannot open external URLs.
 */
export function localizeBackendDestination(
  locale: AppLocale,
  destination: string,
): Route | null {
  if (typeof destination !== "string" || destination.length === 0) {
    return null;
  }

  // Absolute / scheme / protocol-relative / backslash tricks
  if (
    !destination.startsWith("/") ||
    destination.startsWith("//") ||
    destination.includes("://") ||
    destination.includes("\\") ||
    destination.includes("@")
  ) {
    return null;
  }

  let parsed: URL;
  try {
    parsed = new URL(destination, "http://app.local");
  } catch {
    return null;
  }

  if (parsed.origin !== "http://app.local") {
    return null;
  }

  const path = parsed.pathname;
  if (path.includes("..") || path.includes("//")) {
    return null;
  }

  // Strip an accidental leading locale segment if a client re-localizes.
  const localeStripped = stripLeadingLocale(path);
  if (!isKnownBackendPath(localeStripped)) {
    return null;
  }

  const search = parsed.search;
  return `/${locale}${localeStripped}${search}` as Route;
}

const UUID_SEGMENT =
  "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}";
const ROOM_CODE_SEGMENT = "[A-Za-z0-9]{4,16}";

const KNOWN_PATH_PATTERNS: readonly RegExp[] = [
  /^\/play$/,
  new RegExp(`^/games/${UUID_SEGMENT}$`),
  new RegExp(`^/games/${UUID_SEGMENT}/results$`),
  new RegExp(`^/matches/${UUID_SEGMENT}$`),
  new RegExp(`^/matches/${UUID_SEGMENT}/results$`),
  /^\/rooms$/,
  new RegExp(`^/rooms/${ROOM_CODE_SEGMENT}$`),
  /^\/matchmaking$/,
  new RegExp(`^/parties/${UUID_SEGMENT}$`),
  /^\/daily-mission$/,
  new RegExp(`^/daily-mission/play/${UUID_SEGMENT}$`),
  new RegExp(`^/daily-mission/results/${UUID_SEGMENT}$`),
];

function isKnownBackendPath(path: string): boolean {
  return KNOWN_PATH_PATTERNS.some((pattern) => pattern.test(path));
}

function stripLeadingLocale(path: string): string {
  const match = path.match(/^\/(en|ar)(\/.*)?$/);
  if (!match) return path;
  return match[2] && match[2].length > 0 ? match[2] : "/";
}
