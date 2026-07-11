export type RankedGameDTO = {
  id: string;
  mode: string;
  status: "pending" | "active" | "completed" | "abandoned" | "cancelled" | string;
  map_id: string;
  round_count: number;
  timer_seconds: number | null;
  scoring_version: number;
  current_round_number: number | null;
  total_score: number;
  started_at: string | null;
  completed_at: string | null;
};

export type RankedRoundMedia = {
  type: string;
  url: string;
  attribution: string | null;
};

export type RankedRoundDTO = {
  id: string;
  round_number: number;
  status: string;
  starts_at: string | null;
  ends_at: string | null;
  media: RankedRoundMedia | null;
};

export type RankedGameResponse = {
  game: RankedGameDTO;
};

export type RankedCurrentRoundResponse = {
  round: RankedRoundDTO;
};

export type RankedGuessResult = {
  id: string;
  latitude: number;
  longitude: number;
  distance_meters: number;
  score: number;
  submitted_at: string;
};

export type RankedRevealedLocation = {
  latitude: number;
  longitude: number;
  country_code: string;
  region: string | null;
  locality: string | null;
};

export type RankedGuessResultResponse = {
  guess: RankedGuessResult;
  actual_location: RankedRevealedLocation;
};

export type RankedGamePlayerDTO = {
  id: string;
  user_id: string | null;
  display_name: string;
  role: string;
  status: string;
  total_score: number;
};

export type RankedRoundResult = {
  round_id: string;
  round_number: number;
  actual_location: RankedRevealedLocation;
  guesses: RankedGuessResult[];
};

export type RankedGameResultsResponse = {
  game: RankedGameDTO;
  players: RankedGamePlayerDTO[];
  rounds: RankedRoundResult[];
};

export type RankedGuessRequest = {
  latitude: number;
  longitude: number;
};

export type RankedActionState =
  | { ok: true; kind: "guess"; result: RankedGuessResultResponse }
  | { ok: true; kind: "reload"; round: RankedRoundDTO | null; game: RankedGameDTO }
  | { ok: true; kind: "results"; results: RankedGameResultsResponse }
  | { ok: false; code: string; message: string };
