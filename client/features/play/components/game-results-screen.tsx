"use client";

import Link from "next/link";
import type { Route } from "next";
import {
  ResultsMap,
  RoundBreakdown,
  type ResultRound,
} from "@/features/gameplay/components";
import type { GameResults } from "@/features/play/schemas";
import type { AppLocale } from "@/lib/i18n/routing";

export type GameResultsCopy = Readonly<{
  title: string;
  finalScore: string;
  breakdown: string;
  round: string;
  score: string;
  distance: string;
  total: string;
  next: string;
  backToPlay: string;
  playAgain: string;
  yourGuess: string;
  correctPosition: string;
  mapUnavailable: string;
  resultMap: string;
  overview: string;
  pointsOf: string;
}>;

type GameResultsScreenProps = Readonly<{
  locale: AppLocale;
  results: GameResults;
  googleMapsApiKey: string;
  copy: GameResultsCopy;
  playAgainHref: Route | string;
  backHref: Route | string;
}>;

function toResultRounds(results: GameResults): ResultRound[] {
  return results.rounds.map((round) => {
    const guess = round.guesses[0] ?? null;
    return {
      roundNumber: round.round_number,
      guess: guess
        ? {
            latitude: guess.latitude,
            longitude: guess.longitude,
            timed_out: guess.timed_out,
          }
        : null,
      actual: {
        latitude: round.actual_location.latitude,
        longitude: round.actual_location.longitude,
      },
      score: guess?.score,
      distanceMeters: guess?.distance_meters,
    };
  });
}

export function GameResultsScreen({
  locale,
  results,
  googleMapsApiKey,
  copy,
  playAgainHref,
  backHref,
}: GameResultsScreenProps) {
  const resultRounds = toResultRounds(results);
  const breakdownItems = resultRounds.map((round) => ({
    roundNumber: round.roundNumber,
    score: round.score ?? 0,
    distanceMeters: round.distanceMeters ?? 0,
  }));
  const totalDistance = breakdownItems.reduce(
    (sum, item) => sum + item.distanceMeters,
    0,
  );
  const totalScore = results.game.total_score;

  return (
    <main className="min-h-dvh bg-[#08051d] px-4 py-8 text-white sm:px-8">
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-6">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <Link
            href={backHref as Route}
            className="text-sm font-semibold text-violet-200 underline-offset-4 hover:underline"
          >
            {copy.backToPlay}
          </Link>
          <Link
            href={playAgainHref as Route}
            className="rounded-full bg-gradient-to-b from-[#8b72ff] to-[#5538d4] px-5 py-2 text-xs font-black tracking-wide uppercase"
          >
            {copy.playAgain}
          </Link>
        </div>

        <header className="space-y-2">
          <h1 className="text-3xl font-black tracking-tight sm:text-4xl">
            {copy.title}
          </h1>
          <p className="text-lg font-black text-[#ffbd59]">
            {copy.finalScore}: {totalScore.toLocaleString(locale)}
          </p>
        </header>

        <section
          aria-label={copy.resultMap}
          className="overflow-hidden rounded-2xl border border-white/10"
        >
          <ResultsMap
            rounds={resultRounds}
            googleMapsApiKey={googleMapsApiKey}
            locale={locale}
            totalScore={totalScore}
            roundLabel={copy.round}
            scoreLabel={copy.score}
            distanceLabel={copy.distance}
            overviewLabel={copy.overview}
            breakdownLabel={copy.breakdown}
            nextLabel={copy.next}
            finalScoreLabel={copy.finalScore}
            pointsOfLabel={copy.pointsOf}
            yourGuessLabel={copy.yourGuess}
            correctPositionLabel={copy.correctPosition}
            unavailableLabel={copy.mapUnavailable}
            ariaLabel={copy.resultMap}
          />
        </section>

        <section className="rounded-2xl border border-white/10 bg-white/5 p-4 sm:p-6">
          <h2 className="mb-4 text-sm font-black tracking-wide text-white/70 uppercase">
            {copy.breakdown}
          </h2>
          <RoundBreakdown
            rounds={breakdownItems}
            totalScore={totalScore}
            totalDistanceMeters={totalDistance}
            locale={locale}
            roundLabel={copy.round}
            totalLabel={copy.total}
          />
        </section>
      </div>
    </main>
  );
}
