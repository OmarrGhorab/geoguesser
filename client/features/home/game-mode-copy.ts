import "server-only";

import { getTranslations } from "next-intl/server";
import type { AppLocale } from "@/lib/i18n/routing";
import type { AuthenticatedGameModesCopy } from "./types";

export async function getAuthenticatedGameModesCopy(
  locale: AppLocale,
): Promise<AuthenticatedGameModesCopy> {
  const t = await getTranslations({
    locale,
    namespace: "AuthenticatedHome.gameModes",
  });

  return {
    title: t("title"),
    searchLabel: t("searchLabel"),
    searchPlaceholder: t("searchPlaceholder"),
    empty: t("empty"),
    recommended: t("recommended"),
    viewAll: t("viewAll"),
    escapeHint: t("escapeHint"),
    groups: {
      soloAdventures: t("groups.soloAdventures"),
      quickMatch: t("groups.quickMatch"),
      competitive: t("groups.competitive"),
      withFriends: t("groups.withFriends"),
    },
    items: {
      classicSolo: {
        title: t("items.classicSolo.title"),
        description: t("items.classicSolo.description"),
      },
      practice: {
        title: t("items.practice.title"),
        description: t("items.practice.description"),
      },
      dailyChallenge: {
        title: t("items.dailyChallenge.title"),
        description: t("items.dailyChallenge.description"),
      },
      quickPlay: {
        title: t("items.quickPlay.title"),
        description: t("items.quickPlay.description"),
      },
      casualSolo: {
        title: t("items.casualSolo.title"),
        description: t("items.casualSolo.description"),
      },
      casualDuos: {
        title: t("items.casualDuos.title"),
        description: t("items.casualDuos.description"),
      },
      casualSquads: {
        title: t("items.casualSquads.title"),
        description: t("items.casualSquads.description"),
      },
      rankedSolo: {
        title: t("items.rankedSolo.title"),
        description: t("items.rankedSolo.description"),
      },
      rankedDuos: {
        title: t("items.rankedDuos.title"),
        description: t("items.rankedDuos.description"),
      },
      rankedSquads: {
        title: t("items.rankedSquads.title"),
        description: t("items.rankedSquads.description"),
      },
      partyLobby: {
        title: t("items.partyLobby.title"),
        description: t("items.partyLobby.description"),
      },
    },
  };
}
