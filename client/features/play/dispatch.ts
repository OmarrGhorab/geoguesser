import type { Route } from "next";
import type { AppLocale } from "@/lib/i18n/routing";
import { dailyMissionPlayHref } from "@/features/mission/routes";
import { normalizeMode } from "@/features/gameplay/modes";
import { gameResultsHref } from "./routes";

export type GameDispatchInput = {
  id: string;
  mode: string;
  status: string;
};

export type GameDestination =
  | { kind: "render_solo" }
  | { kind: "render_practice" }
  | { kind: "render_quick_play" }
  | { kind: "redirect"; href: Route }
  | { kind: "not_found" };

/** @deprecated Prefer normalizeMode from @/features/gameplay/modes */
export function normalizeInboundMode(mode: string): string {
  return normalizeMode(mode) ?? mode;
}

function isTerminal(status: string): boolean {
  return (
    status === "completed" ||
    status === "abandoned" ||
    status === "cancelled"
  );
}

function isActiveLike(status: string): boolean {
  return status === "pending" || status === "active";
}

/**
 * Authoritative mode-aware destination for GET /games/{id} recovery.
 * Room/match destinations require identifiers not present on the game row;
 * those recovery paths return not_found until dedicated links are available.
 */
export function resolveGameDestination(
  locale: AppLocale,
  game: GameDispatchInput,
): GameDestination {
  const mode = normalizeMode(game.mode);
  if (!mode) return { kind: "not_found" };

  if (mode === "daily") {
    return {
      kind: "redirect",
      href: dailyMissionPlayHref(locale, game.id),
    };
  }

  if (mode === "party_lobby") {
    return { kind: "not_found" };
  }

  if (
    mode === "casual_solo" ||
    mode === "casual_duo" ||
    mode === "casual_squad" ||
    mode === "ranked_solo" ||
    mode === "ranked_duo" ||
    mode === "ranked_squad"
  ) {
    if (isTerminal(game.status)) {
      return {
        kind: "redirect",
        href: gameResultsHref(locale, game.id),
      };
    }
    return { kind: "not_found" };
  }

  if (mode === "solo") {
    if (isTerminal(game.status)) {
      return {
        kind: "redirect",
        href: gameResultsHref(locale, game.id),
      };
    }
    if (isActiveLike(game.status)) return { kind: "render_solo" };
    return { kind: "not_found" };
  }

  if (mode === "quick_play") {
    if (isTerminal(game.status)) {
      return {
        kind: "redirect",
        href: gameResultsHref(locale, game.id),
      };
    }
    if (isActiveLike(game.status)) return { kind: "render_quick_play" };
    return { kind: "not_found" };
  }

  if (mode === "practice") {
    if (isTerminal(game.status)) {
      return {
        kind: "redirect",
        href: gameResultsHref(locale, game.id),
      };
    }
    if (isActiveLike(game.status)) return { kind: "render_practice" };
    return { kind: "not_found" };
  }

  return { kind: "not_found" };
}
