import { z } from "zod";

export const missionSchema = z.object({
  id: z.string().uuid(),
  code: z.string(),
  mission_key: z.enum([
    "daily_completion",
    "score_threshold",
    "round_accuracy",
    "shared_participation",
    "streak_milestone",
  ]),
  title_key: z.string(),
  description_key: z.string(),
  mission_type: z.string(),
  cadence: z.enum(["daily", "weekly"]),
  period_key: z.string(),
  icon_key: z.enum([
    "daily-completion",
    "score-threshold",
    "round-accuracy",
    "shared-participation",
    "streak-milestone",
  ]),
  reward_xp: z.number().int().nonnegative(),
  current_value: z.number().int().nonnegative(),
  target_value: z.number().int().positive(),
  status: z.enum([
    "not_started",
    "in_progress",
    "completed",
    "claimed",
    "expired",
  ]),
  active_ends_at: z.string().nullable().optional(),
});

export const missionsResponseSchema = z.object({
  missions: z.array(missionSchema),
});
export type Mission = z.infer<typeof missionSchema>;
