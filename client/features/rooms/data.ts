import "server-only";

import { apiJson } from "@/lib/api/client";
import { roomSnapshotSchema, type RoomSnapshot } from "./schemas";

export async function getRoom(roomCode: string): Promise<RoomSnapshot> {
  return apiJson(`/rooms/${encodeURIComponent(roomCode)}`, roomSnapshotSchema, {
    requiresAuth: true,
  });
}
