import type { z } from "zod";

import type {
  createGameRequestSchema,
  currentRoundSchema,
  enterMatchmakingRequestSchema,
  gameModeSchema,
  gamePlayerSchema,
  gameResponseSchema,
  gameResultsSchema,
  gameSchema,
  gameStatusSchema,
  guessResultSchema,
  guessSchema,
  mapListResponseSchema,
  mapSummarySchema,
  matchProgressionSchema,
  matchResultsSchema,
  matchSnapshotResponseSchema,
  matchSnapshotSchema,
  matchmakingStatusResponseSchema,
  partyResponseSchema,
  partySchema,
  practiceHistorySchema,
  queueStatusSchema,
  quickPlayResponseSchema,
  realtimeTicketRequestSchema,
  realtimeTicketResponseSchema,
  revealedLocationSchema,
  roomResponseSchema,
  roomSnapshotSchema,
  roundMediaSchema,
  roundSchema,
  submitGuessBodySchema,
} from "./schemas";

export type GameMode = z.infer<typeof gameModeSchema>;
export type GameStatus = z.infer<typeof gameStatusSchema>;
export type Game = z.infer<typeof gameSchema>;
export type GameResponse = z.infer<typeof gameResponseSchema>;
export type QuickPlayResponse = z.infer<typeof quickPlayResponseSchema>;
export type CreateGameRequest = z.infer<typeof createGameRequestSchema>;
export type RoundMedia = z.infer<typeof roundMediaSchema>;
export type Round = z.infer<typeof roundSchema>;
export type CurrentRound = z.infer<typeof currentRoundSchema>;
export type RevealedLocation = z.infer<typeof revealedLocationSchema>;
export type Guess = z.infer<typeof guessSchema>;
export type SubmitGuessBody = z.infer<typeof submitGuessBodySchema>;
export type GuessResult = z.infer<typeof guessResultSchema>;
export type GamePlayer = z.infer<typeof gamePlayerSchema>;
export type GameResults = z.infer<typeof gameResultsSchema>;
export type PracticeHistory = z.infer<typeof practiceHistorySchema>;
export type MapSummary = z.infer<typeof mapSummarySchema>;
export type MapListResponse = z.infer<typeof mapListResponseSchema>;
export type RoomSnapshot = z.infer<typeof roomSnapshotSchema>;
export type RoomResponse = z.infer<typeof roomResponseSchema>;
export type Party = z.infer<typeof partySchema>;
export type PartyResponse = z.infer<typeof partyResponseSchema>;
export type QueueStatus = z.infer<typeof queueStatusSchema>;
export type MatchmakingStatusResponse = z.infer<
  typeof matchmakingStatusResponseSchema
>;
export type EnterMatchmakingRequest = z.infer<
  typeof enterMatchmakingRequestSchema
>;
export type MatchSnapshot = z.infer<typeof matchSnapshotSchema>;
export type MatchSnapshotResponse = z.infer<typeof matchSnapshotResponseSchema>;
export type MatchResults = z.infer<typeof matchResultsSchema>;
export type MatchProgression = z.infer<typeof matchProgressionSchema>;
export type RealtimeTicketRequest = z.infer<typeof realtimeTicketRequestSchema>;
export type RealtimeTicketResponse = z.infer<
  typeof realtimeTicketResponseSchema
>;
