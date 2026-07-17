import type { DailyGameResults } from "./schemas";

export const RESULT_BREAKDOWN_STEP_MS = 2_400;

export type ResultBreakdownRound = {
  roundId: string;
  roundNumber: number;
  answer: { latitude: number; longitude: number };
  guess: {
    latitude: number;
    longitude: number;
    score: number;
    distanceMeters: number;
  } | null;
  score: number;
  distanceMeters: number;
};

export function buildResultBreakdown(
  rounds: DailyGameResults["rounds"],
): ResultBreakdownRound[] {
  return [...rounds]
    .sort((left, right) => left.round_number - right.round_number)
    .map((round) => {
      const submittedGuess = round.guesses[0];
      const visibleGuess = submittedGuess?.timed_out ? null : submittedGuess;

      return {
        roundId: round.round_id,
        roundNumber: round.round_number,
        answer: {
          latitude: round.actual_location.latitude,
          longitude: round.actual_location.longitude,
        },
        guess: visibleGuess
          ? {
              latitude: visibleGuess.latitude,
              longitude: visibleGuess.longitude,
              score: visibleGuess.score,
              distanceMeters: visibleGuess.distance_meters,
            }
          : null,
        score: submittedGuess?.score ?? 0,
        distanceMeters: submittedGuess?.distance_meters ?? 0,
      };
    });
}

export function nextResultBreakdownIndex(current: number, roundCount: number) {
  return Math.min(Math.max(0, current) + 1, Math.max(0, roundCount));
}
