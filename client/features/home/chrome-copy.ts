import "server-only";

import { getTranslations } from "next-intl/server";
import type { AppLocale } from "@/lib/i18n/routing";
import type { AuthenticatedHomeData } from "./schemas";
import type { AuthenticatedChromeCopy } from "./types";

export async function getAuthenticatedChromeCopy(
  locale: AppLocale,
  data: AuthenticatedHomeData,
): Promise<AuthenticatedChromeCopy> {
  const t = await getTranslations({ locale, namespace: "AuthenticatedHome" });
  return {
    brand: t("brand"),
    languageLabel: t("languageLabel"),
    viewerName: data.viewer.display_name,
    viewerAvatarUrl: data.viewer.avatar_url ?? null,
    nav: {
      singleplayer: t("nav.singleplayer"),
      multiplayer: t("nav.multiplayer"),
      party: t("nav.party"),
      challenges: t("nav.challenges"),
      maps: t("nav.maps"),
      leaderboards: t("nav.leaderboards"),
      friends: t("nav.friends"),
      home: t("nav.home"),
      profile: t("nav.profile"),
      stats: t("nav.stats"),
      missions: t("nav.missions"),
      settings: t("nav.settings"),
    },
    premium: {
      title: t("premium.title"),
      body: t("premium.body"),
      sidebarTitle: t("premium.sidebarTitle"),
      sidebarBody: t("premium.sidebarBody"),
      cta: t("premium.cta"),
    },
    aria: {
      primary: t("aria.primary"),
      dashboard: t("aria.dashboard"),
      search: t("aria.search"),
      menu: t("aria.menu"),
    },
  };
}
