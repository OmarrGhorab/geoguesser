import { z } from "zod";

const uuid = z.string().uuid();
const dateTime = z.string().min(1);

/** All GameMode values accepted on recovery/read paths (including legacy aliases). */
export const gameModeSchema = z.enum([
  "solo",
  "private_room",
  "party_lobby",
  "practice",
  "quick_play",
  "daily",
  "ranked",
  "casual_solo",
  "casual_duo",
  "casual_squad",
  "ranked_solo",
  "ranked_duo",
  "ranked_squad",
  "ranked_standard",
]);

export const gameStatusSchema = z.enum([
  "pending",
  "active",
  "completed",
  "abandoned",
  "cancelled",
]);

export const createGameModeSchema = z.enum(["solo", "practice"]);

export const gameSchema = z.object({
  id: uuid,
  mode: gameModeSchema,
  status: gameStatusSchema,
  map_id: uuid.optional(),
  round_count: z.number().int().positive(),
  timer_seconds: z.number().int().positive().nullable().optional(),
  scoring_version: z.number().int().positive(),
  current_round_number: z.number().int().positive().nullable().optional(),
  total_score: z.number().int().nonnegative(),
  started_at: dateTime.nullable().optional(),
  completed_at: dateTime.nullable().optional(),
  open_ended: z.boolean().optional(),
});

export const gameResponseSchema = z.object({
  game: gameSchema,
});

/** POST /games/quick-play success body (same shape as GameResponse). */
export const quickPlayResponseSchema = gameResponseSchema;

export const createGameRequestSchema = z.object({
  mode: createGameModeSchema,
  map_id: uuid,
  round_count: z.number().int().min(1).max(10).optional(),
  timer_seconds: z.number().int().min(10).max(600).nullable().optional(),
});

export const roundMediaSchema = z.object({
  type: z.enum(["image", "panorama"]),
  url: z.string().url().optional(),
  panorama_id: z.string().min(8).max(512).optional(),
  attribution: z.string().nullable().optional(),
});

export const roundSchema = z.object({
  id: uuid,
  round_number: z.number().int().min(1),
  status: z.enum(["pending", "active", "completed", "cancelled"]),
  starts_at: dateTime.nullable().optional(),
  ends_at: dateTime.nullable().optional(),
  media: roundMediaSchema,
});

/** GET /games/{id}/rounds/current — reveal-safe media only (no true coordinates). */
export const currentRoundSchema = z.object({
  round: roundSchema,
});

export const revealedLocationSchema = z.object({
  latitude: z.number(),
  longitude: z.number(),
  country_code: z.string().length(2),
  region: z.string().nullable().optional(),
  locality: z.string().nullable().optional(),
});

export const guessSchema = z.object({
  id: uuid,
  latitude: z.number(),
  longitude: z.number(),
  distance_meters: z.number().int().nonnegative(),
  accuracy_score: z.number().int().min(0).max(5_000).optional(),
  speed_bonus: z.number().int().min(0).max(250).optional(),
  score: z.number().int().nonnegative(),
  submitted_at: dateTime,
  timed_out: z.boolean().optional().default(false),
});

/** POST /games/{id}/rounds/{roundId}/guesses body. */
export const submitGuessBodySchema = z.object({
  latitude: z.number().min(-90).max(90),
  longitude: z.number().min(-180).max(180),
});

/**
 * GuessResultResponse — solo/daily always populate actual_location after accept;
 * multiplayer may leave it null while the shared round is open.
 */
export const guessResultSchema = z.object({
  guess: guessSchema,
  actual_location: revealedLocationSchema.nullable().optional(),
  max_score: z.number().int().positive().optional().default(5_000),
  score_percent: z.number().int().min(0).max(100).optional().default(0),
  max_accuracy_score: z.number().int().min(0).max(5_000).optional(),
  max_speed_bonus: z.number().int().min(0).max(250).optional(),
  outcome: z
    .enum(["perfect", "close", "miss", "timed_out", "submitted"])
    .optional()
    .default("miss"),
  round_completed: z.boolean().optional(),
  game_completed: z.boolean().optional(),
  submitted_count: z.number().int().nonnegative().optional(),
  eligible_count: z.number().int().nonnegative().optional(),
  next_round_number: z.number().int().positive().optional(),
  next_round_available: z.boolean().optional(),
});

export const gamePlayerSchema = z.object({
  id: uuid,
  user_id: uuid.nullable().optional(),
  display_name: z.string(),
  role: z.enum(["host", "player", "spectator"]),
  status: z.enum(["active", "disconnected", "left", "kicked"]),
  total_score: z.number().int().nonnegative(),
});

export const roundResultSchema = z.object({
  round_id: uuid,
  round_number: z.number().int().positive(),
  actual_location: revealedLocationSchema,
  guesses: z.array(guessSchema),
});

/**
 * GameResultsResponse — flexible round count for solo/practice/quick_play.
 * Daily still uses mission/schemas with fixed five-round breakdowns.
 */
export const gameResultsSchema = z.object({
  game: gameSchema,
  players: z.array(gamePlayerSchema),
  rounds: z.array(roundResultSchema),
});

export const practiceRoundHistoryItemSchema = z.object({
  round: roundSchema,
  guess: guessSchema.nullable().optional(),
  actual_location: revealedLocationSchema.nullable().optional(),
});

/** GET /games/{id}/rounds — Practice history page. */
export const practiceHistorySchema = z.object({
  items: z.array(practiceRoundHistoryItemSchema).max(100),
  next_cursor: z.string().nullable().optional(),
  has_more: z.boolean(),
});

export const mapAccessTierSchema = z.enum(["free", "premium", "admin"]);
export const mapDifficultySchema = z.enum(["mixed", "easy", "medium", "hard"]);

export const mapSummarySchema = z.object({
  id: uuid,
  slug: z.string(),
  name: z.string(),
  description: z.string().nullable().optional(),
  visibility: z.enum(["public", "private", "unlisted"]),
  access_tier: mapAccessTierSchema,
  difficulty: mapDifficultySchema,
  status: z.enum(["draft", "active", "archived"]),
  created_at: dateTime,
  updated_at: dateTime,
});

export const pageInfoSchema = z.object({
  limit: z.number().int().min(1).max(100),
  next_cursor: z.string().nullable().optional(),
});

export const mapListResponseSchema = z.object({
  data: z.array(mapSummarySchema),
  page: pageInfoSchema,
});

// --- Rooms (Party Lobby) ---

export const roomStatusSchema = z.enum([
  "lobby",
  "active",
  "completed",
  "expired",
  "cancelled",
]);

export const roomPlayerSchema = z.object({
  id: uuid,
  user_id: uuid.nullable().optional(),
  display_name: z.string(),
  role: z.enum(["host", "player", "spectator"]),
  membership_status: z.enum(["joined", "left", "kicked", "disconnected"]),
  presence_status: z.enum(["connected", "disconnected"]),
  is_ready: z.boolean(),
  total_score: z.number().int(),
  joined_at: dateTime,
  left_at: dateTime.nullable().optional(),
});

export const roomCurrentRoundSchema = z.object({
  id: uuid,
  round_number: z.number().int().min(1),
  status: z.enum(["pending", "active", "completed", "cancelled"]),
  starts_at: dateTime.nullable().optional(),
  ends_at: dateTime.nullable().optional(),
  media: roundMediaSchema.nullable().optional(),
  revealed: z.boolean(),
});

export const roomGuessProgressSchema = z.object({
  submitted_count: z.number().int().nonnegative(),
  eligible_count: z.number().int().nonnegative(),
  submitted_player_ids: z.array(uuid),
});

export const partyLobbyStandingSchema = z.object({
  placement: z.number().int().min(1).max(50),
  player_id: uuid,
  display_name: z.string(),
  total_score: z.number().int().nonnegative(),
  total_distance_meters: z.number().int().nonnegative(),
  rounds_scored: z.number().int().min(0).max(10),
  tied: z.boolean(),
});

export const roomSnapshotSchema = z.object({
  id: uuid,
  game_id: uuid.nullable().optional(),
  host_player_id: uuid.nullable().optional(),
  current_player_id: uuid.nullable().optional(),
  code: z.string(),
  visibility: z.enum(["private", "public"]),
  status: roomStatusSchema,
  version: z.number().int().nonnegative(),
  max_players: z.number().int(),
  round_count: z.number().int(),
  timer_seconds: z.number().int().nullable().optional(),
  expires_at: dateTime,
  players: z.array(roomPlayerSchema),
  ready_player_ids: z.array(uuid),
  current_round: roomCurrentRoundSchema.nullable().optional(),
  guess_progress: roomGuessProgressSchema.nullable().optional(),
  mode: z.literal("party_lobby"),
  standings: z.array(partyLobbyStandingSchema).max(50).optional(),
});

export const roomResponseSchema = z.object({
  room: roomSnapshotSchema,
});

// --- Parties ---

export const partyStatusSchema = z.enum([
  "forming",
  "queued",
  "in_match",
  "closed",
]);

export const partyFormatSchema = z.enum(["solo", "duo", "squad"]);
export const partyDurableFormatSchema = z.enum(["duo", "squad"]);

export const partyMemberSchema = z.object({
  user_id: uuid,
  display_name: z.string(),
  avatar_url: z.string().nullable().optional(),
  ready: z.boolean(),
  is_leader: z.boolean(),
  joined_at: dateTime,
});

export const partySchema = z.object({
  id: uuid,
  format: partyDurableFormatSchema,
  capacity: z.union([z.literal(2), z.literal(4)]),
  status: partyStatusSchema,
  version: z.number().int().min(1),
  leader_user_id: uuid,
  active_match_id: uuid.nullable().optional(),
  members: z.array(partyMemberSchema),
  created_at: dateTime,
  updated_at: dateTime,
});

export const partyResponseSchema = z.object({
  party: partySchema,
});

export const currentPartyResponseSchema = z.object({
  party: partySchema.nullable(),
});

// --- Matchmaking / queue ---

export const queueStatusSchema = z.enum([
  "not_queued",
  "searching",
  "matched",
  "temporarily_unavailable",
]);

export const playlistSchema = z.enum(["casual", "ranked"]);

export const matchModeSchema = z.enum([
  "casual_solo",
  "casual_duo",
  "casual_squad",
  "ranked_solo",
  "ranked_duo",
  "ranked_squad",
  "ranked_standard",
]);

export const matchmakingRatingWindowSchema = z.object({
  minimum: z.number().int().nonnegative(),
  maximum: z.number().int().nonnegative(),
});

export const matchmakingQueueSchema = z.object({
  ticket_id: z.string().optional(),
  playlist: playlistSchema.optional(),
  format: partyFormatSchema.optional(),
  mode: matchModeSchema,
  party_id: uuid.nullable().optional(),
  search_started_at: dateTime,
  lease_expires_at: dateTime,
  rating_window: matchmakingRatingWindowSchema.nullable().optional(),
});

export const matchmakingMatchSchema = z.object({
  match_id: uuid,
  game_id: uuid,
  playlist: playlistSchema.optional(),
  format: partyFormatSchema.optional(),
  mode: matchModeSchema,
  formed_at: dateTime,
  destination: z.string().min(1),
});

export const matchmakingStatusResponseSchema = z.object({
  status: queueStatusSchema,
  queue: matchmakingQueueSchema.nullable(),
  match: matchmakingMatchSchema.nullable(),
});

export const enterMatchmakingRequestSchema = z.object({
  mode: matchModeSchema.optional(),
  playlist: playlistSchema.optional(),
  format: partyFormatSchema.optional(),
  party_id: uuid.nullable().optional(),
  party_version: z.number().int().positive().optional(),
});

// --- Matchplay ---

export const matchStatusSchema = z.enum([
  "matched",
  "active",
  "completed",
  "cancelled",
  "failed_to_start",
]);

export const matchResultEnumSchema = z.enum([
  "team_one_win",
  "team_two_win",
  "draw",
  "forfeit",
  "abandoned",
  "cancelled",
]);

export const matchViewerSchema = z.object({
  user_id: uuid,
  game_player_id: uuid,
  team_slot: z.union([z.literal(1), z.literal(2)]),
  submitted: z.boolean(),
  can_chat: z.boolean(),
  allowed_spectate_player_ids: z.array(uuid),
});

export const matchParticipantSchema = z.object({
  game_player_id: uuid,
  user_id: uuid,
  display_name: z.string(),
  status: z.enum(["active", "disconnected", "left"]),
  submitted: z.boolean(),
  total_score: z.number().int().nonnegative().optional(),
});

export const matchTeamSchema = z.object({
  slot: z.union([z.literal(1), z.literal(2)]),
  score: z.number().int().nonnegative(),
  players: z.array(matchParticipantSchema),
});

export const matchRoundMediaSchema = z.object({
  type: z.enum(["panorama", "image"]),
  panorama_id: z.string().optional(),
  url: z.string().optional(),
  attribution: z.string().nullable().optional(),
});

export const matchRoundSchema = z.object({
  id: uuid,
  number: z.number().int().min(1),
  status: z.enum(["pending", "active", "completed"]),
  starts_at: dateTime.nullable().optional(),
  ends_at: dateTime.nullable().optional(),
  media: matchRoundMediaSchema.nullable().optional(),
  submitted_count: z.number().int().nonnegative(),
  eligible_count: z.number().int().nonnegative(),
});

export const matchTeamMarkerSchema = z.object({
  user_id: uuid,
  latitude: z.number(),
  longitude: z.number(),
  version: z.number().int(),
});

export const matchTeamScoreSchema = z.object({
  slot: z.union([z.literal(1), z.literal(2)]),
  score: z.number().int().nonnegative(),
});

export const revealedGuessSchema = z.object({
  game_player_id: uuid,
  user_id: uuid,
  team_slot: z.union([z.literal(1), z.literal(2)]),
  latitude: z.number().nullable().optional(),
  longitude: z.number().nullable().optional(),
  distance_meters: z.number().int().nonnegative().nullable().optional(),
  accuracy_score: z.number().int().nonnegative(),
  speed_bonus: z.number().int().nonnegative(),
  score: z.number().int().nonnegative(),
  timed_out: z.boolean(),
  submitted_at: dateTime.optional(),
});

export const revealedRoundResultSchema = z.object({
  round_id: uuid,
  round_number: z.number().int().min(1),
  actual_location: revealedLocationSchema,
  guesses: z.array(revealedGuessSchema),
  teams: z.array(matchTeamScoreSchema).min(2).max(2),
});

export const matchSnapshotSchema = z.object({
  id: uuid,
  game_id: uuid,
  playlist: playlistSchema,
  format: partyFormatSchema,
  status: matchStatusSchema,
  result: matchResultEnumSchema.nullable().optional(),
  team_size: z.union([z.literal(1), z.literal(2), z.literal(4)]),
  viewer: matchViewerSchema,
  teams: z.array(matchTeamSchema).min(2).max(2),
  round: matchRoundSchema.nullable().optional(),
  last_round_result: revealedRoundResultSchema.nullable().optional(),
  team_markers: z.array(matchTeamMarkerSchema),
  realtime_version: z.number().int().nonnegative(),
  formed_at: dateTime,
  timer_seconds: z.number().int().nullable().optional(),
});

export const matchSnapshotResponseSchema = z.object({
  match: matchSnapshotSchema,
});

export const rankCodeSchema = z.object({
  code: z.string(),
});

export const matchProgressionSchema = z.object({
  applied: z.boolean(),
  reason: z.enum(["casual", "progression_pending"]).optional(),
  old_rating: z.number().int().nonnegative().optional(),
  base_delta: z.number().int().optional(),
  abandon_penalty: z.number().int().optional(),
  total_delta: z.number().int().optional(),
  new_rating: z.number().int().nonnegative().optional(),
  old_rank: rankCodeSchema.nullable().optional(),
  new_rank: rankCodeSchema.nullable().optional(),
  placement: z.number().int().min(1).nullable().optional(),
  placements_completed: z.number().int().min(0).max(5).nullable().optional(),
  placements_required: z.literal(5).nullable().optional(),
  top500_position: z.number().int().min(1).nullable().optional(),
});

/** GET /matches/{id}/results — terminal Casual/Ranked result. */
export const matchResultsSchema = z.object({
  result: matchResultEnumSchema,
  winner_team_slot: z
    .union([z.literal(1), z.literal(2)])
    .nullable()
    .optional(),
  teams: z.array(matchTeamSchema).min(2).max(2),
  rounds: z.array(revealedRoundResultSchema).optional(),
  progression: matchProgressionSchema,
});

// --- Realtime tickets ---

export const realtimeTicketRequestSchema = z.object({
  channel_kind: z.enum(["party", "match"]),
  channel_id: uuid,
});

export const realtimeTicketResponseSchema = z.object({
  ticket: z.string().min(1),
  expires_at: dateTime,
  websocket_url: z.string().min(1),
});
