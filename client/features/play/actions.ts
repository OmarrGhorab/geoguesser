"use server";

import { redirect } from "next/navigation";
import type { Route } from "next";
import { z } from "zod";
import { apiJson } from "@/lib/api/client";
import { ApiError } from "@/lib/api/errors";
import { routing, type AppLocale } from "@/lib/i18n/routing";
import {
  createPracticeGameInputSchema,
  createQuickPlayInputSchema,
  createSoloGameInputSchema,
  currentRoundSchema,
  gameResponseSchema,
  guessResultSchema,
  practiceGameCommandSchema,
  quickPlayResponseSchema,
  startGameInputSchema,
  type CurrentRound,
  type PlayActionResult,
} from "@/features/play/schemas";

function safeLocale(locale: AppLocale | string | undefined): AppLocale {
  if (locale && routing.locales.includes(locale as AppLocale)) {
    return locale as AppLocale;
  }
  return routing.defaultLocale;
}

function actionError(error: unknown): PlayActionResult {
  if (error instanceof ApiError) {
    if (error.status === 401 || error.status === 403) {
      return { ok: false, code: "unauthenticated" };
    }
    if (error.status === 404) {
      return { ok: false, code: "unavailable" };
    }
    if (error.status === 429) {
      return { ok: false, code: "rate_limited" };
    }
    // Map location-selector failures (422) to a stable UI code.
    if (
      error.status === 422 ||
      error.code === "unprocessable_entity" ||
      error.code === "not_enough_locations"
    ) {
      return { ok: false, code: "not_enough_locations" };
    }
    if (error.code === "validation_failed" || error.status === 400) {
      return { ok: false, code: "validation_failed" };
    }
    return { ok: false, code: error.code || "unavailable" };
  }
  return { ok: false, code: "unavailable" };
}

function redirectIfUnauthenticated(
  error: unknown,
  locale: AppLocale,
): never | void {
  if (
    error instanceof ApiError &&
    (error.status === 401 || error.status === 403)
  ) {
    redirect(`/${locale}/login?reason=session_expired` as Route);
  }
}

async function createAndStartGame(input: {
  mode: "solo" | "practice";
  mapId: string;
  roundCount?: number;
  timerSeconds?: number | null;
  idempotencyKey: string;
  locale: AppLocale;
}): Promise<PlayActionResult> {
  try {
    const body: Record<string, unknown> = {
      mode: input.mode,
      map_id: input.mapId,
    };
    if (input.roundCount != null) body.round_count = input.roundCount;
    if (input.timerSeconds !== undefined) {
      body.timer_seconds = input.timerSeconds;
    }

    const created = await apiJson("/games", gameResponseSchema, {
      method: "POST",
      body,
      idempotencyKey: input.idempotencyKey,
      requiresAuth: true,
      forwardCookies: true,
    });

    if (created.game.status === "active") {
      return { ok: true, gameId: created.game.id };
    }

    const started = await apiJson(
      `/games/${created.game.id}/start`,
      gameResponseSchema,
      {
        method: "POST",
        idempotencyKey: `start-${created.game.id}`,
        requiresAuth: true,
        forwardCookies: true,
      },
    );
    return { ok: true, gameId: started.game.id };
  } catch (error) {
    redirectIfUnauthenticated(error, input.locale);
    return actionError(error);
  }
}

export async function createSoloGameAction(
  input: unknown,
  locale: AppLocale = routing.defaultLocale,
): Promise<PlayActionResult> {
  const parsed = createSoloGameInputSchema.safeParse(input);
  if (!parsed.success) return { ok: false, code: "validation_failed" };
  const value = parsed.data;
  return createAndStartGame({
    mode: "solo",
    mapId: value.mapId,
    roundCount: value.roundCount,
    timerSeconds: value.timerSeconds,
    idempotencyKey: value.idempotencyKey,
    locale: safeLocale(locale),
  });
}

export async function createPracticeGameAction(
  input: unknown,
  locale: AppLocale = routing.defaultLocale,
): Promise<PlayActionResult> {
  const parsed = createPracticeGameInputSchema.safeParse(input);
  if (!parsed.success) return { ok: false, code: "validation_failed" };
  const value = parsed.data;
  return createAndStartGame({
    mode: "practice",
    mapId: value.mapId,
    timerSeconds: null,
    idempotencyKey: value.idempotencyKey,
    locale: safeLocale(locale),
  });
}

/**
 * POST /games/quick-play. Returns unavailable when the backend operation is
 * not mounted yet (404) or the map is not configured.
 */
export async function createQuickPlayAction(
  input: unknown,
  locale: AppLocale = routing.defaultLocale,
): Promise<PlayActionResult> {
  const parsed = createQuickPlayInputSchema.safeParse(input);
  if (!parsed.success) return { ok: false, code: "validation_failed" };
  const safe = safeLocale(locale);

  try {
    const response = await apiJson("/games/quick-play", quickPlayResponseSchema, {
      method: "POST",
      body: {},
      idempotencyKey: parsed.data.idempotencyKey,
      requiresAuth: true,
    });
    return { ok: true, gameId: response.game.id };
  } catch (error) {
    redirectIfUnauthenticated(error, safe);
    if (error instanceof ApiError && error.status === 404) {
      return { ok: false, code: "unavailable" };
    }
    return actionError(error);
  }
}

export async function startGameAction(
  input: unknown,
  locale: AppLocale = routing.defaultLocale,
): Promise<PlayActionResult> {
  const parsed = startGameInputSchema.safeParse(input);
  if (!parsed.success) return { ok: false, code: "validation_failed" };
  const safe = safeLocale(locale);

  try {
    const response = await apiJson(
      `/games/${parsed.data.gameId}/start`,
      gameResponseSchema,
      {
        method: "POST",
        idempotencyKey:
          parsed.data.idempotencyKey ?? `start-${parsed.data.gameId}`,
        requiresAuth: true,
      },
    );
    return { ok: true, gameId: response.game.id };
  } catch (error) {
    redirectIfUnauthenticated(error, safe);
    return actionError(error);
  }
}

export type NextPracticeRoundResult =
  | { ok: true; round: CurrentRound }
  | { ok: false; code: string };

export async function nextPracticeRoundAction(
  input: unknown,
  locale: AppLocale = routing.defaultLocale,
): Promise<NextPracticeRoundResult> {
  const parsed = practiceGameCommandSchema.safeParse(input);
  if (!parsed.success) return { ok: false, code: "validation_failed" };
  const safe = safeLocale(locale);

  try {
    const response = await apiJson(
      `/games/${parsed.data.gameId}/rounds/next`,
      currentRoundSchema,
      {
        method: "POST",
        idempotencyKey: parsed.data.idempotencyKey,
        requiresAuth: true,
      },
    );
    return { ok: true, round: response.round };
  } catch (error) {
    redirectIfUnauthenticated(error, safe);
    if (error instanceof ApiError) return { ok: false, code: error.code };
    return { ok: false, code: "unavailable" };
  }
}

export async function endPracticeAction(
  input: unknown,
  locale: AppLocale = routing.defaultLocale,
): Promise<PlayActionResult> {
  const parsed = practiceGameCommandSchema.safeParse(input);
  if (!parsed.success) return { ok: false, code: "validation_failed" };
  const safe = safeLocale(locale);

  try {
    const response = await apiJson(
      `/games/${parsed.data.gameId}/end`,
      gameResponseSchema,
      {
        method: "POST",
        idempotencyKey: parsed.data.idempotencyKey,
        requiresAuth: true,
      },
    );
    return { ok: true, gameId: response.game.id };
  } catch (error) {
    redirectIfUnauthenticated(error, safe);
    return actionError(error);
  }
}

export type SubmitPlayGuessResult =
  | {
      ok: true;
      kind: "next_round";
      reveal: import("@/features/play/schemas").GuessResult;
      nextRound: CurrentRound;
    }
  | {
      ok: true;
      kind: "completed";
      reveal: import("@/features/play/schemas").GuessResult;
    }
  | { ok: false; code: string };

const submitPlayGuessInputSchema = z.object({
  gameId: z.string().uuid(),
  roundId: z.string().uuid(),
  latitude: z.number().min(-90).max(90),
  longitude: z.number().min(-180).max(180),
  idempotencyKey: z.string().min(16).max(128),
});

const expirePlayRoundInputSchema = z.object({
  gameId: z.string().uuid(),
  roundId: z.string().uuid(),
});

/** Submit a solo/quick/practice guess and resolve next round or completion. */
export async function submitPlayGuessAction(
  input: unknown,
): Promise<SubmitPlayGuessResult> {
  const parsed = submitPlayGuessInputSchema.safeParse(input);
  if (!parsed.success) return { ok: false, code: "invalid_guess" };
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
    if (error instanceof ApiError) return { ok: false, code: error.code };
    return { ok: false, code: "unavailable" };
  }
}

export async function expirePlayRoundAction(
  input: unknown,
): Promise<SubmitPlayGuessResult> {
  const parsed = expirePlayRoundInputSchema.safeParse(input);
  if (!parsed.success) return { ok: false, code: "invalid_round" };
  const value = parsed.data;

  try {
    const reveal = await apiJson(
      `/games/${value.gameId}/rounds/${value.roundId}/timeout`,
      guessResultSchema,
      { method: "POST", requiresAuth: true },
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
    if (error instanceof ApiError) return { ok: false, code: error.code };
    return { ok: false, code: "unavailable" };
  }
}
