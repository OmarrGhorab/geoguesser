export type RoomStatus = "lobby" | "active" | "completed" | "expired" | "cancelled";
export type RoomVisibility = "private" | "public";
export type RoomPresenceStatus = "connected" | "disconnected";
export type RoomMembershipStatus = "joined" | "left" | "kicked" | "disconnected";
export type RoomPlayerRole = "host" | "player" | "spectator";

export type CreateRoomRequest = {
  map_id: string;
  visibility: RoomVisibility;
  round_count: number;
  timer_seconds: number | null;
  max_players: number;
  display_name?: string;
};

export type JoinRoomRequest = {
  code: string;
  display_name?: string;
};

export type UpdateRoomSettingsRequest = {
  map_id?: string;
  round_count?: number;
  timer_seconds?: number | null;
  max_players?: number;
};

export type RoomResponse = {
  room: Room;
};

export type Room = {
  id: string;
  code: string;
  visibility: RoomVisibility;
  status: RoomStatus;
  game_id: string | null;
  host_player_id: string | null;
  current_player_id: string | null;
  version: number;
  max_players: number;
  round_count: number;
  timer_seconds: number | null;
  expires_at: string;
  players: RoomPlayer[];
  ready_player_ids: string[];
  current_round?: RoomCurrentRound;
  guess_progress?: RoomGuessProgress;
};

export type RoomPlayer = {
  id: string;
  user_id: string | null;
  display_name: string;
  role: RoomPlayerRole;
  membership_status: RoomMembershipStatus;
  presence_status: RoomPresenceStatus;
  is_ready: boolean;
  total_score: number;
  joined_at: string;
  left_at: string | null;
};

export type RoomCurrentRound = {
  id: string;
  round_number: number;
  status: string;
  starts_at: string | null;
  ends_at: string | null;
  media: {
    type: string;
    url: string;
    attribution?: string | null;
  } | null;
  revealed: boolean;
};

export type RoomGuessProgress = {
  submitted_count: number;
  eligible_count: number;
  submitted_player_ids: string[];
};
