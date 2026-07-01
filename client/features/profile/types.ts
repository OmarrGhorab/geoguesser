export type ProfileDTO = {
  user_id: string;
  email: string;
  display_name: string;
  avatar_url?: string | null;
  country_code?: string | null;
  locale: string;
  timezone?: string | null;
  preferences?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type StatsDTO = {
  games_played: number;
  total_score: number;
  average_score: number;
  best_score: number;
  last_played_at?: string | null;
};

export type GameHistoryItemDTO = {
  id: string;
  map_id: string;
  mode: string;
  status: string;
  round_count: number;
  current_round_number?: number | null;
  total_score: number;
  started_at?: string | null;
  completed_at?: string | null;
  created_at: string;
};

export type PageDTO = {
  limit: number;
  next_cursor?: string | null;
};

export type ProgressDTO = {
  recent_games: GameHistoryItemDTO[];
  page: PageDTO;
};

export type ProfileResponse = {
  profile: ProfileDTO;
  stats: StatsDTO;
  progress: ProgressDTO;
};

export type UpdateProfileRequest = {
  display_name?: string;
  avatar_url?: string | null;
  country_code?: string | null;
  locale?: string;
  timezone?: string | null;
};

export type PublicProfileDTO = {
  user_id: string;
  display_name: string;
  avatar_url?: string | null;
  country_code?: string | null;
};

export type PublicProfileResponse = {
  profile: PublicProfileDTO;
  stats: StatsDTO;
};

export type GameHistoryResponse = {
  games: GameHistoryItemDTO[];
  page: PageDTO;
};
