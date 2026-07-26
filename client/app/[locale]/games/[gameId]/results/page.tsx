import { notFound, redirect } from "next/navigation";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { z } from "zod";
import { GameResultsScreen } from "@/features/play/components/game-results-screen";
import { getGame, getGameResults } from "@/features/play/data";
import { resolveGameDestination } from "@/features/play/dispatch";
import { gamePlayHref, playSetupHref } from "@/features/play/routes";
import type { AppLocale } from "@/lib/i18n/routing";
import { ApiError } from "@/lib/api/errors";

type GameResultsPageProps = Readonly<{
  params: Promise<{ locale: string; gameId: string }>;
}>;

export default async function GameResultsPage({
  params,
}: GameResultsPageProps) {
  const { locale, gameId } = await params;
  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);

  const parsedGameId = z.string().uuid().safeParse(gameId);
  if (!parsedGameId.success) notFound();

  let game;
  try {
    game = await getGame(parsedGameId.data);
  } catch (error) {
    if (error instanceof ApiError && (error.status === 404 || error.status === 403)) {
      notFound();
    }
    throw error;
  }

  const destination = resolveGameDestination(appLocale, game);
  if (
    destination.kind === "render_solo" ||
    destination.kind === "render_quick_play" ||
    destination.kind === "render_practice"
  ) {
    redirect(gamePlayHref(appLocale, game.id));
  }
  if (destination.kind === "not_found") notFound();
  if (destination.kind === "redirect" && !destination.href.includes("/results")) {
    // Daily and other semantic redirects win over this results page.
    if (game.mode === "daily") redirect(destination.href);
  }

  let results;
  try {
    results = await getGameResults(parsedGameId.data);
  } catch (error) {
    if (error instanceof ApiError && (error.status === 404 || error.status === 403)) {
      notFound();
    }
    throw error;
  }

  const t = await getTranslations("Play");
  const modeQuery =
    game.mode === "practice"
      ? "practice"
      : game.mode === "quick_play"
        ? "quick_play"
        : "solo";

  return (
    <GameResultsScreen
      locale={appLocale}
      results={results}
      googleMapsApiKey={process.env.NEXT_PUBLIC_GOOGLE_MAPS_API_KEY ?? ""}
      playAgainHref={playSetupHref(appLocale, modeQuery)}
      backHref={playSetupHref(appLocale, modeQuery)}
      copy={{
        title: t("results.title"),
        finalScore: t("results.finalScore"),
        breakdown: t("results.breakdown"),
        round: t("results.round"),
        score: t("results.score"),
        distance: t("results.distance"),
        total: t("results.total"),
        next: t("results.next"),
        backToPlay: t("results.backToPlay"),
        playAgain: t("results.playAgain"),
        yourGuess: t("results.yourGuess"),
        correctPosition: t("results.correctPosition"),
        mapUnavailable: t("results.mapUnavailable"),
        resultMap: t("results.resultMap"),
        overview: t("results.overview"),
        pointsOf: t("results.pointsOf"),
      }}
    />
  );
}
