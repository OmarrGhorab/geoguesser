"use server";

import { getCurrentRound, getGame, getGameResults, submitGuess } from "@/lib/api/games";
import { ApiError } from "@/lib/api/errors";
import type { RankedActionState, RankedGuessRequest } from "./ranked-types";

function mapError(error: unknown): RankedActionState {
  if (error instanceof ApiError) {
    return {
      ok: false,
      code: error.detail.code || "request_failed",
      message: error.detail.message || "The request failed.",
    };
  }
  return {
    ok: false,
    code: "unexpected_error",
    message: "Something went wrong. Please try again.",
  };
}

function newIdempotencyKey(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `ranked-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export async function submitRankedGuessAction(
  gameId: string,
  roundId: string,
  body: RankedGuessRequest,
  idempotencyKey?: string,
): Promise<RankedActionState> {
  if (!Number.isFinite(body.latitude) || body.latitude < -90 || body.latitude > 90) {
    return { ok: false, code: "invalid_guess", message: "Latitude must be between -90 and 90." };
  }
  if (!Number.isFinite(body.longitude) || body.longitude < -180 || body.longitude > 180) {
    return { ok: false, code: "invalid_guess", message: "Longitude must be between -180 and 180." };
  }

  try {
    const result = await submitGuess(gameId, roundId, body, idempotencyKey || newIdempotencyKey());
    return { ok: true, kind: "guess", result };
  } catch (error) {
    return mapError(error);
  }
}

export async function reloadRankedRoundAction(gameId: string): Promise<RankedActionState> {
  try {
    const game = await getGame(gameId);
    if (game.status === "completed") {
      const results = await getGameResults(gameId);
      return { ok: true, kind: "results", results };
    }
    try {
      const round = await getCurrentRound(gameId);
      return { ok: true, kind: "reload", round, game };
    } catch (error) {
      if (error instanceof ApiError && (error.status === 404 || error.status === 409)) {
        return { ok: true, kind: "reload", round: null, game };
      }
      throw error;
    }
  } catch (error) {
    return mapError(error);
  }
}

export async function loadRankedResultsAction(gameId: string): Promise<RankedActionState> {
  try {
    const results = await getGameResults(gameId);
    return { ok: true, kind: "results", results };
  } catch (error) {
    return mapError(error);
  }
}
