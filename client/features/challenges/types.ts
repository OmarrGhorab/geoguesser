export type SettingsSnapshot = {
  round_count: number;
  timer_seconds?: number | null;
  movement_rules: string;
  scoring_version: number;
};

export type ChallengeSummary = {
  id: string;
  type: "daily" | "shared";
  seed: string;
  challenge_date?: string;
  reset_starts_at?: string;
  reset_ends_at?: string;
  map: { id: string };
  settings: SettingsSnapshot;
  status: string;
  share_code?: string;
  share_url?: string;
};

export type AttemptSummary = {
  id: string;
  challenge_id: string;
  status: string;
  leaderboard_eligible: boolean;
  started_at?: string;
  completed_at?: string;
  total_score: number;
  current_round_number?: number;
  game_id?: string;
};

export type StreakSummary = {
  current_count: number;
  best_count: number;
  last_completed_challenge_date?: string;
  status: string;
  protection_state: string;
  guest_limited: boolean;
};

export type MissionSummary = {
  id?: string;
  code: string;
  title_key: string;
  description_key: string;
  mission_type: string;
  current_value: number;
  target_value: number;
  status: string;
  active_ends_at?: string;
};

export type ChallengeMetadataResponse = {
  challenge: ChallengeSummary;
  attempt_state?: AttemptSummary;
  streak: StreakSummary;
  missions_summary: MissionSummary[];
  leaderboard_summary: { participants: number };
  countdown?: {
    reset_ends_at: string;
    seconds_remaining: number;
  };
};

export type ChallengeAttemptResponse = {
  challenge: ChallengeSummary;
  attempt: AttemptSummary;
  game?: {
    id: string;
    status: string;
    current_round_number?: number;
  };
};

export type ChallengeRoundResult = {
  round_number: number;
  score: number;
  distance_meters: number;
};

export type ChallengeResultResponse = {
  challenge: ChallengeSummary;
  attempt: AttemptSummary;
  visible: boolean;
  total_score?: number | null;
  total_distance_meters?: number | null;
  round_results?: ChallengeRoundResult[];
  rank_context?: Record<string, unknown> | null;
  streak?: StreakSummary | null;
  missions_summary?: MissionSummary[];
  message?: string;
};
