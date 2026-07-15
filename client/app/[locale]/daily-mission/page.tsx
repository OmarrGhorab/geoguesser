import { getTranslations, setRequestLocale } from "next-intl/server";
import { DailyMissionScreen } from "@/features/mission/components/daily-mission-screen";
import type { DailyMissionCopy } from "@/features/mission/types";
import type { AppLocale } from "@/lib/i18n/routing";

type DailyMissionPageProps = Readonly<{
  params: Promise<{ locale: string }>;
}>;

export default async function DailyMissionPage({
  params,
}: DailyMissionPageProps) {
  const { locale } = await params;
  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);

  const t = await getTranslations("DailyMission");
  const copy: DailyMissionCopy = {
    back: t("back"),
    today: t("today"),
    prevDay: t("prevDay"),
    nextDay: t("nextDay"),
    play: t("play"),
    playedToday: t("playedToday"),
    cards: {
      locations: t("cards.locations"),
      compare: t("cards.compare"),
      daily: t("cards.daily"),
    },
    aria: {
      screen: t("aria.screen"),
      back: t("aria.back"),
      dayPicker: t("aria.dayPicker"),
    },
  };

  return <DailyMissionScreen locale={appLocale} copy={copy} />;
}
