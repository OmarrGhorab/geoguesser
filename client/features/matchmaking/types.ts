export const MATCHMAKING_MODE_RANKED_STANDARD = "ranked_standard" as const;

export type MatchmakingMode = typeof MATCHMAKING_MODE_RANKED_STANDARD;

export type MatchmakingPublicStatus =
  | "not_queued"
  | "searching"
  | "matched"
  | "temporarily_unavailable";

export type MatchmakingQueueDetails = {
  mode: MatchmakingMode;
  search_started_at: string;
  lease_expires_at: string;
};

export type MatchmakingMatchDetails = {
  match_id: string;
  game_id: string;
  mode: MatchmakingMode;
  formed_at: string;
  destination: string;
};

export type MatchmakingStatusNotQueued = {
  status: "not_queued";
  queue: null;
  match: null;
};

export type MatchmakingStatusSearching = {
  status: "searching";
  queue: MatchmakingQueueDetails;
  match: null;
};

export type MatchmakingStatusMatched = {
  status: "matched";
  queue: null;
  match: MatchmakingMatchDetails;
};

export type MatchmakingStatusUnavailable = {
  status: "temporarily_unavailable";
  queue: null;
  match: null;
};

export type MatchmakingStatusResponse =
  | MatchmakingStatusNotQueued
  | MatchmakingStatusSearching
  | MatchmakingStatusMatched
  | MatchmakingStatusUnavailable;

export type JoinMatchmakingRequest = {
  mode: MatchmakingMode;
};

export type MatchmakingActionState =
  | { ok: true; status: MatchmakingStatusResponse }
  | { ok: false; code: string; message: string; retryAfterSeconds?: number };
