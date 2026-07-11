"use server";

import { joinMatchmaking, leaveMatchmaking } from "@/lib/api/matchmaking";
import { ApiError } from "@/lib/api/errors";
import {
  MATCHMAKING_MODE_RANKED_STANDARD,
  type MatchmakingActionState,
  type MatchmakingMode,
} from "./types";

function mapError(error: unknown): MatchmakingActionState {
  if (error instanceof ApiError) {
    return {
      ok: false,
      code: error.detail.code || "request_failed",
      message: error.detail.message || "The request failed.",
      retryAfterSeconds: error.status === 429 ? 60 : undefined,
    };
  }
  return {
    ok: false,
    code: "unexpected_error",
    message: "Something went wrong. Please try again.",
  };
}

export async function joinMatchmakingAction(
  mode: MatchmakingMode = MATCHMAKING_MODE_RANKED_STANDARD,
): Promise<MatchmakingActionState> {
  if (mode !== MATCHMAKING_MODE_RANKED_STANDARD) {
    return {
      ok: false,
      code: "unsupported_matchmaking_mode",
      message: "That matchmaking mode is not supported.",
    };
  }

  try {
    const status = await joinMatchmaking({ mode });
    return { ok: true, status };
  } catch (error) {
    return mapError(error);
  }
}

export async function leaveMatchmakingAction(): Promise<MatchmakingActionState> {
  try {
    await leaveMatchmaking();
    return {
      ok: true,
      status: { status: "not_queued", queue: null, match: null },
    };
  } catch (error) {
    return mapError(error);
  }
}
