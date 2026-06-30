import type { Room } from "./types";

export type RoomConnectionState = "connecting" | "connected" | "reconnecting" | "degraded" | "closed";

export type RoomEventEnvelope<TPayload = unknown> = {
  event_id: string;
  type: RoomEventType;
  room_code: string;
  game_id?: string | null;
  occurred_at: string;
  version: number;
  payload: TPayload;
};

export type RoomEventType =
  | "room.snapshot"
  | "room.player_joined"
  | "room.player_left"
  | "room.player_disconnected"
  | "room.player_reconnected"
  | "room.player_removed"
  | "room.settings_updated"
  | "room.ready_updated"
  | "room.ready_reset"
  | "room.started"
  | "round.started"
  | "round.guess_count_changed"
  | "round.ended"
  | "round.results_revealed"
  | "game.completed"
  | "room.error";

export type RoomSnapshotPayload = {
  room: Room;
};
