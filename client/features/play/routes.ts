import type { Route } from "next";
import type { AppLocale } from "@/lib/i18n/routing";
import {
  gameHref,
  gameResultsHref as sharedGameResultsHref,
  playHref,
} from "@/features/gameplay/routes";
import type { PlayModeQuery } from "./schemas";

export function playSetupHref(
  locale: AppLocale,
  mode?: PlayModeQuery,
): Route {
  return playHref(locale, mode);
}

export function gamePlayHref(locale: AppLocale, gameId: string): Route {
  return gameHref(locale, gameId);
}

export function gameResultsHref(locale: AppLocale, gameId: string): Route {
  return sharedGameResultsHref(locale, gameId);
}
