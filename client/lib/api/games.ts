import "server-only";

import { apiFetch } from "@/lib/api/client";
import type {
  RankedCurrentRoundResponse,
  RankedGameResponse,
  RankedGameResultsResponse,
  RankedGuessRequest,
  RankedGuessResultResponse,
} from "@/features/game/ranked-types";

export async function getGame(gameId: string) {
  const response = await apiFetch<RankedGameResponse>(`/games/${encodeURIComponent(gameId)}`, {
    cache: "no-store",
  });
  return response.game;
}

export async function getCurrentRound(gameId: string) {
  const response = await apiFetch<RankedCurrentRoundResponse>(
    `/games/${encodeURIComponent(gameId)}/rounds/current`,
    { cache: "no-store" },
  );
  return response.round;
}

export async function submitGuess(
  gameId: string,
  roundId: string,
  body: RankedGuessRequest,
  idempotencyKey?: string,
) {
  return apiFetch<RankedGuessResultResponse>(
    `/games/${encodeURIComponent(gameId)}/rounds/${encodeURIComponent(roundId)}/guesses`,
    {
      method: "POST",
      body,
      cache: "no-store",
      headers: idempotencyKey ? { "Idempotency-Key": idempotencyKey } : undefined,
    },
  );
}

export async function getGameResults(gameId: string) {
  return apiFetch<RankedGameResultsResponse>(`/games/${encodeURIComponent(gameId)}/results`, {
    cache: "no-store",
  });
}
