import type { Route } from "next";
import type { AppLocale } from "@/lib/i18n/routing";

export function dailyMissionPlayHref(locale: AppLocale, gameId: string): Route {
  return `/${locale}/daily-mission/play/${gameId}` as Route;
}

export function dailyMissionResultsHref(
  locale: AppLocale,
  gameId: string,
): Route {
  return `/${locale}/daily-mission/results/${gameId}` as Route;
}
