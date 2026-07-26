import { notFound, redirect } from "next/navigation";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { z } from "zod";
import {
  SoloGameScreen,
  type SoloGameCopy,
} from "@/features/play/components/solo-game-screen";
import { getCurrentRound, getGame } from "@/features/play/data";
import { resolveGameDestination } from "@/features/play/dispatch";
import { playSetupHref } from "@/features/play/routes";
import type { AppLocale } from "@/lib/i18n/routing";
import { ApiError } from "@/lib/api/errors";

type GamePageProps = Readonly<{
  params: Promise<{ locale: string; gameId: string }>;
}>;

function buildSoloCopy(
  t: Awaited<ReturnType<typeof getTranslations>>,
): SoloGameCopy {
  return {
    title: t("game.title"),
    round: t("game.round"),
    of: t("game.of"),
    score: t("game.score"),
    distance: t("game.distance"),
    placePin: t("game.placePin"),
    submitGuess: t("game.submitGuess"),
    submitting: t("game.submitting"),
    next: t("game.next"),
    finalScore: t("game.finalScore"),
    breakdown: t("game.breakdown"),
    backToPlay: t("game.backToPlay"),
    loadingMap: t("game.loadingMap"),
    mapUnavailable: t("game.mapUnavailable"),
    retryMap: t("game.retryMap"),
    selectLocation: t("game.selectLocation"),
    total: t("game.total"),
    zoomIn: t("game.zoomIn"),
    zoomOut: t("game.zoomOut"),
    recenter: t("game.recenter"),
    settings: t("game.settings"),
    openReactions: t("game.openReactions"),
    reactionsTitle: t("game.reactionsTitle"),
    closeReactions: t("game.closeReactions"),
    reactionsPrompt: t("game.reactionsPrompt"),
    reactionSent: t("game.reactionSent"),
    expandMap: t("game.expandMap"),
    closeMap: t("game.closeMap"),
    mapLabel: t("game.mapLabel"),
    timeExpired: t("game.timeExpired"),
    correctLocation: t("game.correctLocation"),
    endPractice: t("game.endPractice"),
    nextPracticeRound: t("game.nextPracticeRound"),
    errors: {
      invalidGuess: t("game.errors.invalidGuess"),
      unavailable: t("game.errors.unavailable"),
    },
  };
}

export default async function GamePage({ params }: GamePageProps) {
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
  if (destination.kind === "not_found") notFound();
  if (destination.kind === "redirect") redirect(destination.href);

  let round = null;
  try {
    round = await getCurrentRound(game.id);
  } catch (error) {
    if (!(error instanceof ApiError && error.status === 404)) {
      throw error;
    }
  }

  const t = await getTranslations("Play");
  const modeQuery =
    destination.kind === "render_practice"
      ? "practice"
      : destination.kind === "render_quick_play"
        ? "quick_play"
        : "solo";

  return (
    <SoloGameScreen
      locale={appLocale}
      game={game}
      initialRound={round}
      googleMapsApiKey={process.env.NEXT_PUBLIC_GOOGLE_MAPS_API_KEY ?? ""}
      copy={buildSoloCopy(t)}
      backHref={playSetupHref(appLocale, modeQuery)}
    />
  );
}
