import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { z } from "zod";
import { DailyResultsScreen } from "@/features/mission/components/daily-results-screen";
import { getDailyGameResults } from "@/features/mission/data";
import type { DailyResultsCopy } from "@/features/mission/types";
import type { AppLocale } from "@/lib/i18n/routing";

type DailyResultsPageProps = Readonly<{
  params: Promise<{ locale: string; gameId: string }>;
}>;

export async function generateMetadata({
  params,
}: DailyResultsPageProps): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({
    locale: locale as AppLocale,
    namespace: "DailyMission.game",
  });
  return {
    title: t("resultMetaTitle"),
    description: t("resultMetaDescription"),
    robots: { index: false, follow: false },
  };
}

export default async function DailyResultsPage({
  params,
}: DailyResultsPageProps) {
  const { locale, gameId } = await params;
  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);
  const parsedGameId = z.string().uuid().safeParse(gameId);
  if (!parsedGameId.success) notFound();

  const [results, t] = await Promise.all([
    getDailyGameResults(parsedGameId.data),
    getTranslations("DailyMission.game"),
  ]);
  const copy: DailyResultsCopy = {
    title: t("title"),
    resultsTab: t("resultsTab"),
    mapTab: t("mapTab"),
    myGame: t("myGame"),
    friends: t("friends"),
    clubs: t("clubs"),
    country: t("country"),
    all: t("all"),
    player: t("player"),
    you: t("you"),
    total: t("total"),
    statusTitle: t("statusTitle"),
    statusBody: t("statusBody"),
    curious: t("curious"),
    comparisonUnavailable: t("comparisonUnavailable"),
    replay: t("replay"),
    finalScore: t("finalScore"),
    pointsOf: t("pointsOf"),
    breakdown: t("breakdown"),
    overview: t("overview"),
    next: t("next"),
    backToMission: t("backToMission"),
    round: t("round"),
    score: t("score"),
    distance: t("distance"),
    roundsCompleted: t("roundsCompleted"),
    totalDistance: t("totalDistance"),
    yourGuess: t("yourGuess"),
    correctPosition: t("correctPosition"),
    mapUnavailable: t("resultMapUnavailable"),
    aria: {
      resultMap: t("aria.resultMap"),
      showBreakdown: t("aria.showBreakdown"),
      showOverview: t("aria.showOverview"),
    },
  };

  return (
    <DailyResultsScreen
      locale={appLocale}
      results={results}
      googleMapsApiKey={process.env.NEXT_PUBLIC_GOOGLE_MAPS_API_KEY ?? ""}
      copy={copy}
    />
  );
}
