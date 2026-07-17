import { notFound, redirect } from "next/navigation";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { z } from "zod";
import { startDailyMissionAction } from "@/features/mission/actions";
import { DailyGameScreen } from "@/features/mission/components/daily-game-screen";
import { getDailyGameState } from "@/features/mission/data";
import { dailyMissionResultsHref } from "@/features/mission/routes";
import type { DailyGameCopy } from "@/features/mission/types";
import type { AppLocale } from "@/lib/i18n/routing";

type DailyGamePageProps = Readonly<{
  params: Promise<{ locale: string; gameId: string }>;
}>;

export default async function DailyGamePage({ params }: DailyGamePageProps) {
  const { locale, gameId } = await params;
  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);

  const parsedGameId = z.string().uuid().safeParse(gameId);
  if (!parsedGameId.success) notFound();

  const [state, t] = await Promise.all([
    getDailyGameState(parsedGameId.data),
    getTranslations("DailyMission.game"),
  ]);
  if (state.game.status === "completed") {
    redirect(dailyMissionResultsHref(appLocale, parsedGameId.data));
  }

  const copy: DailyGameCopy = {
    title: t("title"),
    round: t("round"),
    of: t("of"),
    score: t("score"),
    distance: t("distance"),
    placePin: t("placePin"),
    submitGuess: t("submitGuess"),
    submitting: t("submitting"),
    next: t("next"),
    finalScore: t("finalScore"),
    breakdown: t("breakdown"),
    backToMission: t("backToMission"),
    loadingMap: t("loadingMap"),
    mapUnavailable: t("mapUnavailable"),
    retryMap: t("retryMap"),
    selectLocation: t("selectLocation"),
    total: t("total"),
    zoomIn: t("zoomIn"),
    zoomOut: t("zoomOut"),
    recenter: t("recenter"),
    settings: t("settings"),
    openReactions: t("openReactions"),
    reactionsTitle: t("reactionsTitle"),
    closeReactions: t("closeReactions"),
    reactionsPrompt: t("reactionsPrompt"),
    reactionSent: t("reactionSent"),
    expandMap: t("expandMap"),
    closeMap: t("closeMap"),
    mapLabel: t("mapLabel"),
    timeExpired: t("timeExpired"),
    correctLocation: t("correctLocation"),
    errors: {
      invalidGuess: t("errors.invalidGuess"),
      unavailable: t("errors.unavailable"),
    },
  };

  return (
    <DailyGameScreen
      locale={appLocale}
      game={state.game}
      initialRound={state.round}
      googleMapsApiKey={process.env.NEXT_PUBLIC_GOOGLE_MAPS_API_KEY ?? ""}
      copy={copy}
      retryAction={startDailyMissionAction.bind(null, appLocale)}
    />
  );
}
