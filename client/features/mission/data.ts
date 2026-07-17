import "server-only";

import { apiJson } from "@/lib/api/client";
import {
  currentRoundSchema,
  dailyMissionSchema,
  gameResultsSchema,
  gameResponseSchema,
  type DailyGame,
  type DailyGameResults,
  type DailyMissionData,
  type DailyRound,
} from "./schemas";

export async function getDailyMission(
  challengeDate?: string,
): Promise<DailyMissionData> {
  const query = challengeDate
    ? `?${new URLSearchParams({ date: challengeDate }).toString()}`
    : "";
  return apiJson(`/challenges/daily${query}`, dailyMissionSchema);
}

export async function getTodayDailyMission(): Promise<DailyMissionData> {
  return getDailyMission();
}

export type DailyGameState = {
  game: DailyGame;
  round: DailyRound | null;
};

export async function getDailyGameState(
  gameId: string,
): Promise<DailyGameState> {
  const { game } = await apiJson(`/games/${gameId}`, gameResponseSchema);
  if (game.status === "completed") {
    return { game, round: null };
  }

  const { round } = await apiJson(
    `/games/${gameId}/rounds/current`,
    currentRoundSchema,
  );
  return { game, round };
}

export async function getDailyGameResults(
  gameId: string,
): Promise<DailyGameResults> {
  return apiJson(`/games/${gameId}/results`, gameResultsSchema);
}
