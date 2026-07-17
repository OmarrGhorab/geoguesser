"use server";

import { redirect } from "next/navigation";
import type { Route } from "next";
import { apiJson } from "@/lib/api/client";
import { ApiError } from "@/lib/api/errors";
import { routing, type AppLocale } from "@/lib/i18n/routing";
import { dailyMissionPlayHref, dailyMissionResultsHref } from "./routes";
import {
  challengeAttemptResponseSchema,
  currentRoundSchema,
  gameResponseSchema,
  guessResultSchema,
  expireRoundInputSchema,
  submitGuessInputSchema,
  type DailyGuessResult,
  type DailyRound,
} from "./schemas";

export type SubmitDailyGuessResult =
  | {
      ok: true;
      kind: "next_round";
      reveal: DailyGuessResult;
      nextRound: DailyRound;
    }
  | {
      ok: true;
      kind: "completed";
      reveal: DailyGuessResult;
    }
  | { ok: false; code: string };

export async function startDailyMissionAction(locale: AppLocale) {
  const safeLocale = routing.locales.includes(locale)
    ? locale
    : routing.defaultLocale;
  const date = new Date().toISOString().slice(0, 10);
  const response = await apiJson(
    "/challenges/daily/attempts",
    challengeAttemptResponseSchema,
    {
      method: "POST",
      idempotencyKey: `daily-mission-playable-v2-${date}-${crypto.randomUUID()}`,
      requiresAuth: true,
    },
  ).catch((error: unknown) => {
    if (
      error instanceof ApiError &&
      (error.status === 401 || error.status === 403)
    ) {
      redirect(`/${safeLocale}/login?reason=session_expired` as Route);
    }
    redirect(`/${safeLocale}/daily-mission?error=unavailable` as Route);
  });

  const gameId = response.game?.id ?? response.attempt.game_id;
  if (!gameId) {
    redirect(`/${safeLocale}/daily-mission?error=unavailable`);
  }
  const destination =
    response.attempt.status === "completed"
      ? dailyMissionResultsHref(safeLocale, gameId)
      : dailyMissionPlayHref(safeLocale, gameId);
  redirect(destination);
}

export async function expireDailyRoundAction(
  input: unknown,
): Promise<SubmitDailyGuessResult> {
  const parsed = expireRoundInputSchema.safeParse(input);
  if (!parsed.success) return { ok: false, code: "invalid_round" };
  const value = parsed.data;
  try {
    const reveal = await apiJson(
      `/games/${value.gameId}/rounds/${value.roundId}/timeout`,
      guessResultSchema,
      { method: "POST", requiresAuth: true },
    );
    const { game } = await apiJson(`/games/${value.gameId}`, gameResponseSchema, {
      requiresAuth: true,
    });
    if (game.status === "completed") {
      return { ok: true, kind: "completed", reveal };
    }
    const { round } = await apiJson(
      `/games/${value.gameId}/rounds/current`,
      currentRoundSchema,
      { requiresAuth: true },
    );
    return { ok: true, kind: "next_round", reveal, nextRound: round };
  } catch (error) {
    if (error instanceof ApiError) return { ok: false, code: error.code };
    return { ok: false, code: "unavailable" };
  }
}

export async function submitDailyGuessAction(
  input: unknown,
): Promise<SubmitDailyGuessResult> {
  const parsed = submitGuessInputSchema.safeParse(input);
  if (!parsed.success) {
    return { ok: false, code: "invalid_guess" };
  }
  const value = parsed.data;

  try {
    const reveal = await apiJson(
      `/games/${value.gameId}/rounds/${value.roundId}/guesses`,
      guessResultSchema,
      {
        method: "POST",
        body: { latitude: value.latitude, longitude: value.longitude },
        idempotencyKey: value.idempotencyKey,
        requiresAuth: true,
      },
    );
    const { game } = await apiJson(
      `/games/${value.gameId}`,
      gameResponseSchema,
      { requiresAuth: true },
    );

    if (game.status === "completed") {
      return { ok: true, kind: "completed", reveal };
    }

    const { round } = await apiJson(
      `/games/${value.gameId}/rounds/current`,
      currentRoundSchema,
      { requiresAuth: true },
    );
    return { ok: true, kind: "next_round", reveal, nextRound: round };
  } catch (error) {
    if (error instanceof ApiError) {
      return { ok: false, code: error.code };
    }
    return { ok: false, code: "unavailable" };
  }
}
