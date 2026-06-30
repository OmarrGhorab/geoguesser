import type { Room } from "./types";
import type { RoomEventEnvelope, RoomSnapshotPayload } from "./realtime-types";

export type RoomState = {
  room: Room;
  seenEventIds: Set<string>;
  needsRefetch: boolean;
};

export function createRoomState(room: Room): RoomState {
  return { room, seenEventIds: new Set<string>(), needsRefetch: false };
}

export function applyRoomEvent(state: RoomState, event: RoomEventEnvelope): RoomState {
  if (state.seenEventIds.has(event.event_id) || event.version <= state.room.version) {
    return state;
  }
  if (event.version !== state.room.version + 1 && event.type !== "room.snapshot") {
    return { ...state, needsRefetch: true };
  }

  const seenEventIds = new Set(state.seenEventIds);
  seenEventIds.add(event.event_id);

  if (event.type === "room.snapshot") {
    const payload = event.payload as RoomSnapshotPayload;
    return { room: payload.room, seenEventIds, needsRefetch: false };
  }

  return { ...state, seenEventIds, needsRefetch: true };
}
