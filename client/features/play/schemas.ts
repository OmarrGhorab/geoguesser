import { z } from "zod";
import {
  currentRoundSchema,
  gameResponseSchema,
  gameResultsSchema,
  gameSchema,
  guessResultSchema,
  mapListResponseSchema,
  mapSummarySchema,
  practiceHistorySchema,
  quickPlayResponseSchema,
  revealedLocationSchema,
  submitGuessBodySchema as backendSubmitGuessBodySchema,
} from "@/features/gameplay/schemas";

const uuid = z.string().uuid();
const idempotencyKeySchema = z.string().min(16).max(128);

export {
  currentRoundSchema,
  gameResponseSchema,
  gameResultsSchema,
  gameSchema,
  guessResultSchema,
  mapListResponseSchema,
  practiceHistorySchema,
  quickPlayResponseSchema,
  revealedLocationSchema,
};

export type Game = z.infer<typeof gameSchema>;
export type CurrentRound = z.infer<typeof currentRoundSchema>["round"];
export type GuessResult = z.infer<typeof guessResultSchema>;
export type GameResults = z.infer<typeof gameResultsSchema>;
export type PracticeHistory = z.infer<typeof practiceHistorySchema>;
export type PlayableMap = z.infer<typeof mapSummarySchema>;

export const playModeQuerySchema = z.enum(["solo", "quick_play", "practice"]);
export type PlayModeQuery = z.infer<typeof playModeQuerySchema>;

export const createSoloGameInputSchema = z.object({
  mapId: uuid,
  roundCount: z.number().int().min(1).max(10).optional(),
  timerSeconds: z.number().int().min(10).max(600).nullable().optional(),
  idempotencyKey: idempotencyKeySchema,
});
export type CreateSoloGameInput = z.infer<typeof createSoloGameInputSchema>;

export const createPracticeGameInputSchema = z.object({
  mapId: uuid,
  idempotencyKey: idempotencyKeySchema,
});
export type CreatePracticeGameInput = z.infer<
  typeof createPracticeGameInputSchema
>;

export const createQuickPlayInputSchema = z.object({
  idempotencyKey: idempotencyKeySchema,
});
export type CreateQuickPlayInput = z.infer<typeof createQuickPlayInputSchema>;

export const startGameInputSchema = z.object({
  gameId: uuid,
  idempotencyKey: idempotencyKeySchema.optional(),
});

export const practiceGameCommandSchema = z.object({
  gameId: uuid,
  idempotencyKey: idempotencyKeySchema,
});

export const playActionResultSchema = z.discriminatedUnion("ok", [
  z.object({ ok: z.literal(true), gameId: uuid }),
  z.object({ ok: z.literal(false), code: z.string().min(1) }),
]);
export type PlayActionResult = z.infer<typeof playActionResultSchema>;

/** Same-origin guess route body: coordinates + durable idempotency key. */
export const submitGuessBodySchema = backendSubmitGuessBodySchema.extend({
  idempotencyKey: idempotencyKeySchema,
});
export type SubmitGuessBody = z.infer<typeof submitGuessBodySchema>;
