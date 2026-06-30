"use server";

import { revalidatePath } from "next/cache";
import { removeRoomPlayer, startRoom, updateRoomSettings } from "@/lib/api/rooms";

export async function updateRoomSettingsAction(roomCode: string, formData: FormData) {
  const roundCount = Number(formData.get("round_count"));
  const maxPlayers = Number(formData.get("max_players"));
  const timerValue = String(formData.get("timer_seconds") ?? "").trim();

  await updateRoomSettings(roomCode, {
    round_count: Number.isFinite(roundCount) ? roundCount : undefined,
    max_players: Number.isFinite(maxPlayers) ? maxPlayers : undefined,
    timer_seconds: timerValue === "" ? null : Number(timerValue),
  });
  revalidatePath(`/rooms/${roomCode}`);
}

export async function startRoomAction(roomCode: string) {
  await startRoom(roomCode, crypto.randomUUID());
  revalidatePath(`/rooms/${roomCode}`);
}

export async function removeRoomPlayerAction(roomCode: string, playerId: string) {
  await removeRoomPlayer(roomCode, playerId);
  revalidatePath(`/rooms/${roomCode}`);
}
