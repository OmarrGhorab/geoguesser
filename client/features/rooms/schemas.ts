import { z } from "zod";

const uuid = z.string().uuid();

export const createRoomInputSchema = z.object({
  mapId: uuid,
  roundCount: z.number().int().min(1).max(10).optional(),
  timerSeconds: z.number().int().min(10).max(600).nullable().optional(),
  maxPlayers: z.number().int().min(2).max(50).optional(),
  displayName: z.string().min(1).max(32).optional(),
  idempotencyKey: z.string().min(16).max(128),
});

export const joinRoomInputSchema = z.object({
  code: z.string().min(4).max(12),
  displayName: z.string().min(1).max(32).optional(),
});

export const roomCodeSchema = z
  .string()
  .min(4)
  .max(12)
  .regex(/^[A-Za-z0-9]+$/);

export const roomPlayerSchema = z
  .object({
    id: uuid,
    user_id: uuid.nullable().optional(),
    display_name: z.string(),
    role: z.string(),
    membership_status: z.string().optional(),
    status: z.string().optional(),
    is_ready: z.boolean().optional(),
    total_score: z.number().int().optional(),
  })
  .passthrough();

export const roomMediaSchema = z
  .object({
    type: z.string(),
    url: z.string().optional(),
    panorama_id: z.string().optional(),
    attribution: z.string().nullable().optional(),
  })
  .passthrough();

export const roomCurrentRoundSchema = z
  .object({
    id: uuid,
    round_number: z.number().int().positive(),
    status: z.string(),
    starts_at: z.string().nullable().optional(),
    ends_at: z.string().nullable().optional(),
    media: roomMediaSchema.nullable().optional(),
    revealed: z.boolean().optional(),
  })
  .passthrough();

export const roomGuessProgressSchema = z
  .object({
    submitted_count: z.number().int().nonnegative(),
    eligible_count: z.number().int().nonnegative(),
    submitted_player_ids: z.array(uuid).optional(),
  })
  .passthrough();

export const roomSnapshotSchema = z.object({
  room: z
    .object({
      id: uuid.optional(),
      code: z.string(),
      status: z.string(),
      host_player_id: uuid.nullable().optional(),
      current_player_id: uuid.nullable().optional(),
      max_players: z.number().int().positive().optional(),
      round_count: z.number().int().positive().optional(),
      timer_seconds: z.number().int().nullable().optional(),
      map_id: uuid.optional(),
      game_id: uuid.nullable().optional(),
      version: z.number().int().optional(),
      players: z.array(roomPlayerSchema).optional(),
      mode: z.string().optional(),
      current_round: roomCurrentRoundSchema.nullable().optional(),
      guess_progress: roomGuessProgressSchema.nullable().optional(),
    })
    .passthrough(),
});

export type RoomSnapshot = z.infer<typeof roomSnapshotSchema>;
export type RoomDTO = RoomSnapshot["room"];
export type CreateRoomInput = z.infer<typeof createRoomInputSchema>;
export type JoinRoomInput = z.infer<typeof joinRoomInputSchema>;

export type RoomActionResult =
  | { ok: true; roomCode: string; room?: RoomDTO }
  | { ok: false; code: string };
