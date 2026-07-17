import { z } from "zod";

const uuid = z.string().uuid();
const date = z.string().regex(/^\d{4}-\d{2}-\d{2}$/);
const dateTime = z.string().min(1);

export const gameSchema = z.object({
  id: uuid,
  mode: z.string(),
  status: z.enum(["pending", "active", "completed", "abandoned", "cancelled"]),
  map_id: uuid,
  round_count: z.number().int().positive(),
  timer_seconds: z.number().int().positive().nullable().optional(),
  scoring_version: z.number().int().positive(),
  current_round_number: z.number().int().positive().nullable().optional(),
  total_score: z.number().int().nonnegative(),
  started_at: dateTime.nullable().optional(),
  completed_at: dateTime.nullable().optional(),
});

const challengeSummarySchema = z.object({
  id: uuid,
  type: z.enum(["daily", "shared"]),
  seed: z.string(),
  challenge_date: date.nullable().optional(),
  reset_starts_at: dateTime.nullable().optional(),
  reset_ends_at: dateTime.nullable().optional(),
  map: z.object({ id: uuid }),
  settings: z.object({
    round_count: z.number().int().min(1).max(10),
    timer_seconds: z.number().int().nullable().optional(),
    movement_rules: z.string(),
    scoring_version: z.number().int().positive(),
  }),
  status: z.string(),
});

const dailyChallengeSummarySchema = challengeSummarySchema.extend({
  type: z.literal("daily"),
  challenge_date: date,
  settings: challengeSummarySchema.shape.settings.extend({
    round_count: z.literal(5),
  }),
});

export const challengeAttemptSchema = z.object({
  id: uuid,
  challenge_id: uuid,
  status: z.enum(["pending", "active", "completed", "abandoned", "expired"]),
  leaderboard_eligible: z.boolean(),
  started_at: dateTime.nullable().optional(),
  completed_at: dateTime.nullable().optional(),
  total_score: z.number().int().nonnegative(),
  current_round_number: z.number().int().positive().nullable().optional(),
  game_id: uuid.nullable().optional(),
  daily_game_number: z.number().int().min(1).max(5).optional().default(1),
});

export const dailyMissionSchema = z.object({
  challenge: dailyChallengeSummarySchema,
  attempt_state: challengeAttemptSchema.nullable().optional(),
  last_completed_game_id: uuid.nullable().optional(),
  streak: z.object({
    current_count: z.number().int().nonnegative(),
    best_count: z.number().int().nonnegative(),
    status: z.string(),
    protection_state: z.string(),
    guest_limited: z.boolean(),
  }),
  missions_summary: z.array(z.unknown()),
  leaderboard_summary: z.object({
    participants: z.number().int().nonnegative(),
  }),
  countdown: z
    .object({
      reset_ends_at: dateTime,
      seconds_remaining: z.number().int().nonnegative(),
    })
    .nullable()
    .optional(),
  daily_games_played: z.number().int().min(0).max(5).optional().default(0),
  daily_games_total: z.number().int().positive().optional().default(5),
});

export const challengeAttemptResponseSchema = z.object({
  challenge: dailyChallengeSummarySchema,
  attempt: challengeAttemptSchema,
  game: gameSchema.nullable().optional(),
});

export const currentRoundSchema = z.object({
  round: z.object({
    id: uuid,
    round_number: z.number().int().min(1).max(10),
    status: z.string(),
    starts_at: dateTime.nullable().optional(),
    ends_at: dateTime.nullable().optional(),
    media: z.object({
      type: z.enum(["image", "panorama"]),
      url: z.string().url().optional(),
      panorama_id: z.string().min(8).max(512).optional(),
      attribution: z.string().nullable().optional(),
    }),
  }),
});

export const guessResultSchema = z.object({
  guess: z.object({
    id: uuid,
    latitude: z.number(),
    longitude: z.number(),
    distance_meters: z.number().int().nonnegative(),
    score: z.number().int().nonnegative(),
    submitted_at: dateTime,
    timed_out: z.boolean().optional().default(false),
  }),
  actual_location: z.object({
    latitude: z.number(),
    longitude: z.number(),
    country_code: z.string().length(2),
    region: z.string().nullable().optional(),
    locality: z.string().nullable().optional(),
  }),
  max_score: z.number().int().positive().optional().default(5_000),
  score_percent: z.number().int().min(0).max(100).optional().default(0),
  outcome: z
    .enum(["perfect", "close", "miss", "timed_out"])
    .optional()
    .default("miss"),
});

const revealedLocationSchema = z.object({
  latitude: z.number(),
  longitude: z.number(),
  country_code: z.string().length(2),
  region: z.string().nullable().optional(),
  locality: z.string().nullable().optional(),
});

export const gameResultsSchema = z.object({
  game: gameSchema,
  players: z.array(
    z.object({
      id: uuid,
      user_id: uuid.nullable().optional(),
      display_name: z.string(),
      role: z.enum(["host", "player", "spectator"]),
      status: z.enum(["active", "disconnected", "left", "kicked"]),
      total_score: z.number().int().nonnegative(),
    }),
  ),
  rounds: z
    .array(
      z.object({
        round_id: uuid,
        round_number: z.number().int().positive(),
        actual_location: revealedLocationSchema,
        guesses: z.array(guessResultSchema.shape.guess).min(1),
      }),
    )
    .length(5),
});

export const challengeResultSchema = z.object({
  challenge: dailyChallengeSummarySchema,
  attempt: challengeAttemptSchema,
  visible: z.boolean(),
  total_score: z.number().int().nonnegative().nullable().optional(),
  total_distance_meters: z.number().int().nonnegative().nullable().optional(),
  round_results: z
    .array(
      z.object({
        round_number: z.number().int().positive(),
        score: z.number().int().nonnegative(),
        distance_meters: z.number().int().nonnegative(),
      }),
    )
    .length(5)
    .optional()
    .default([]),
});

export const gameResponseSchema = z.object({ game: gameSchema });

export const submitGuessInputSchema = z.object({
  gameId: uuid,
  roundId: uuid,
  latitude: z.number().min(-90).max(90),
  longitude: z.number().min(-180).max(180),
  idempotencyKey: uuid,
});

export const expireRoundInputSchema = z.object({
  gameId: uuid,
  roundId: uuid,
});

export type DailyMissionData = z.infer<typeof dailyMissionSchema>;
export type DailyGame = z.infer<typeof gameSchema>;
export type DailyRound = z.infer<typeof currentRoundSchema>["round"];
export type DailyGuessResult = z.infer<typeof guessResultSchema>;
export type DailyChallengeResult = z.infer<typeof challengeResultSchema>;
export type DailyGameResults = z.infer<typeof gameResultsSchema>;
