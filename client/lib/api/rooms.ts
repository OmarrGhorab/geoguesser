import "server-only";

import { apiFetch } from "@/lib/api/client";
import type { CreateRoomRequest, JoinRoomRequest, RoomResponse, UpdateRoomSettingsRequest } from "@/features/rooms/types";

export async function createRoom(request: CreateRoomRequest) {
  return apiFetch<RoomResponse>("/rooms", {
    method: "POST",
    body: request,
    cache: "no-store",
  });
}

export async function joinRoom(request: JoinRoomRequest) {
  return apiFetch<RoomResponse>("/rooms/join", {
    method: "POST",
    body: request,
    cache: "no-store",
  });
}

export async function getRoom(roomCode: string) {
  return apiFetch<RoomResponse>(`/rooms/${encodeURIComponent(roomCode)}`, {
    cache: "no-store",
  });
}

export async function updateRoomSettings(roomCode: string, request: UpdateRoomSettingsRequest) {
  return apiFetch<RoomResponse>(`/rooms/${encodeURIComponent(roomCode)}/settings`, {
    method: "PATCH",
    body: request,
    cache: "no-store",
  });
}

export async function startRoom(roomCode: string, idempotencyKey: string) {
  return apiFetch<RoomResponse>(`/rooms/${encodeURIComponent(roomCode)}/start`, {
    method: "POST",
    headers: { "Idempotency-Key": idempotencyKey },
    cache: "no-store",
  });
}

export async function removeRoomPlayer(roomCode: string, playerId: string) {
  return apiFetch<void>(`/rooms/${encodeURIComponent(roomCode)}/players/${encodeURIComponent(playerId)}`, {
    method: "DELETE",
    cache: "no-store",
  });
}
