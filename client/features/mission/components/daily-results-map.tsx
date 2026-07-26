"use client";

import { useMemo } from "react";
import {
  ResultsMap,
  type ResultRound,
} from "@/features/gameplay/components/results-map";
import { buildResultBreakdown } from "@/features/mission/result-breakdown";
import type { DailyGameResults } from "@/features/mission/schemas";
import type { AppLocale } from "@/lib/i18n/routing";

type DailyResultsMapProps = Readonly<{
  rounds: DailyGameResults["rounds"];
  locale: AppLocale;
  totalScore: number;
  googleMapsApiKey: string;
  roundLabel: string;
  scoreLabel: string;
  distanceLabel: string;
  overviewLabel: string;
  breakdownLabel: string;
  nextLabel: string;
  finalScoreLabel: string;
  pointsOfLabel: string;
  yourGuessLabel: string;
  correctPositionLabel: string;
  unavailableLabel: string;
  ariaLabel: string;
  className?: string;
  showZoomControls?: boolean;
}>;

/**
 * Daily adapter: maps mission result rounds into shared ResultsMap props.
 */
export function DailyResultsMap({
  rounds,
  locale,
  totalScore,
  googleMapsApiKey,
  roundLabel,
  scoreLabel,
  distanceLabel,
  overviewLabel,
  breakdownLabel,
  nextLabel,
  finalScoreLabel,
  pointsOfLabel,
  yourGuessLabel,
  correctPositionLabel,
  unavailableLabel,
  ariaLabel,
  className,
  showZoomControls = false,
}: DailyResultsMapProps) {
  const resultRounds = useMemo<ResultRound[]>(
    () =>
      buildResultBreakdown(rounds).map((round) => ({
        roundNumber: round.roundNumber,
        guess: round.guess
          ? {
              latitude: round.guess.latitude,
              longitude: round.guess.longitude,
            }
          : null,
        actual: {
          latitude: round.answer.latitude,
          longitude: round.answer.longitude,
        },
        score: round.score,
        distanceMeters: round.distanceMeters,
      })),
    [rounds],
  );

  return (
    <ResultsMap
      rounds={resultRounds}
      locale={locale}
      totalScore={totalScore}
      googleMapsApiKey={googleMapsApiKey}
      roundLabel={roundLabel}
      scoreLabel={scoreLabel}
      distanceLabel={distanceLabel}
      overviewLabel={overviewLabel}
      breakdownLabel={breakdownLabel}
      nextLabel={nextLabel}
      finalScoreLabel={finalScoreLabel}
      pointsOfLabel={pointsOfLabel}
      yourGuessLabel={yourGuessLabel}
      correctPositionLabel={correctPositionLabel}
      unavailableLabel={unavailableLabel}
      ariaLabel={ariaLabel}
      className={className}
      showZoomControls={showZoomControls}
    />
  );
}
