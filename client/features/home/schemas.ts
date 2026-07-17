import { z } from "zod";

const uuidSchema = z.string().uuid();
/** RFC3339-ish timestamps from Go encoding/json */
const isoDateTimeSchema = z.string().min(1);
const isoDateSchema = z
  .string()
  .regex(/^\d{4}-\d{2}-\d{2}$/, "expected YYYY-MM-DD");

export const homeViewerSchema = z.object({
  user_id: uuidSchema,
  display_name: z.string().min(1),
  avatar_url: z.string().nullable().optional(),
  country_code: z
    .string()
    .regex(/^[A-Z]{2}$/)
    .nullable()
    .optional(),
});

export const homeStatsSchema = z.object({
  games_played: z.number().int().nonnegative(),
  total_score: z.number().int(),
  average_score: z.number(),
  best_score: z.number().int(),
  last_played_at: isoDateTimeSchema.nullable().optional(),
});

const challengeMapSummarySchema = z.object({
  id: uuidSchema,
});

const settingsSnapshotSchema = z.object({
  round_count: z.number().int(),
  timer_seconds: z.number().int().nullable().optional(),
  movement_rules: z.string(),
  scoring_version: z.number().int(),
});

export const homeChallengeSummarySchema = z.object({
  id: uuidSchema,
  type: z.string(),
  seed: z.string(),
  challenge_date: isoDateSchema.nullable().optional(),
  reset_starts_at: isoDateTimeSchema.nullable().optional(),
  reset_ends_at: isoDateTimeSchema.nullable().optional(),
  map: challengeMapSummarySchema,
  settings: settingsSnapshotSchema,
  status: z.string(),
  share_code: z.string().nullable().optional(),
  share_url: z.string().nullable().optional(),
});

export const homeStreakSchema = z.object({
  current_count: z.number().int().nonnegative(),
  best_count: z.number().int().nonnegative(),
  last_completed_challenge_date: isoDateSchema.nullable().optional(),
  status: z.string(),
  protection_state: z.string(),
  guest_limited: z.boolean(),
});

export const homeLeaderboardSummarySchema = z.object({
  participants: z.number().int().nonnegative(),
});

export const homeCountdownSchema = z.object({
  reset_ends_at: isoDateTimeSchema,
  seconds_remaining: z.number().int().nonnegative(),
});

export const homeDailyChallengeSchema = z.object({
  challenge: homeChallengeSummarySchema,
  attempt_state: z.unknown().nullable().optional(),
  streak: homeStreakSchema,
  missions_summary: z.array(z.unknown()).nullish().transform((v) => v ?? []),
  leaderboard_summary: homeLeaderboardSummarySchema,
  countdown: homeCountdownSchema.nullable().optional(),
});

export const homeRecommendedMapSchema = z.object({
  id: uuidSchema,
  slug: z.string().min(1),
  name: z.string().min(1),
  description: z.string().nullable().optional(),
  visibility: z.string(),
  access_tier: z.string(),
  difficulty: z.string(),
  status: z.string(),
  created_at: isoDateTimeSchema,
  updated_at: isoDateTimeSchema,
});

/** GET /home — authenticated home snapshot. */
export const authenticatedHomeSchema = z.object({
  viewer: homeViewerSchema,
  stats: homeStatsSchema,
  daily_challenge: homeDailyChallengeSchema,
  recommended_maps: z.array(homeRecommendedMapSchema).max(4),
});

export type AuthenticatedHomeData = z.infer<typeof authenticatedHomeSchema>;
export type HomeViewer = z.infer<typeof homeViewerSchema>;
export type HomeStats = z.infer<typeof homeStatsSchema>;
export type HomeDailyChallenge = z.infer<typeof homeDailyChallengeSchema>;
export type HomeRecommendedMap = z.infer<typeof homeRecommendedMapSchema>;
