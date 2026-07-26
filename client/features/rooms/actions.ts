"use server";

import { redirect } from "next/navigation";
import type { Route } from "next";
import { apiJson, apiFetch } from "@/lib/api/client";
import { ApiError } from "@/lib/api/errors";
import { routing, type AppLocale } from "@/lib/i18n/routing";
import {
  createRoomInputSchema,
  joinRoomInputSchema,
  roomCodeSchema,
  roomSnapshotSchema,
  type RoomActionResult,
} from "./schemas";

function safeLocale(locale: AppLocale | string | undefined): AppLocale {
  if (locale && routing.locales.includes(locale as AppLocale)) {
    return locale as AppLocale;
  }
  return routing.defaultLocale;
}

function actionError(error: unknown): RoomActionResult {
  if (error instanceof ApiError) {
    if (error.status === 401 || error.status === 403) {
      return { ok: false, code: "unauthenticated" };
    }
    if (error.status === 409) {
      return { ok: false, code: error.code || "conflict" };
    }
    if (error.status === 404) {
      return { ok: false, code: "not_found" };
    }
    if (error.status === 400 || error.code === "validation_failed") {
      return { ok: false, code: "validation_failed" };
    }
    if (error.status === 422) {
      // Backend uses invalid request when fewer than 2 players try to start.
      return { ok: false, code: error.code || "need_players" };
    }
    return { ok: false, code: error.code || "unavailable" };
  }
  return { ok: false, code: "unavailable" };
}

export async function createRoomAction(
  input: unknown,
  locale: AppLocale = routing.defaultLocale,
): Promise<RoomActionResult> {
  const parsed = createRoomInputSchema.safeParse(input);
  if (!parsed.success) return { ok: false, code: "validation_failed" };
  const value = parsed.data;
  const safe = safeLocale(locale);

  try {
    const body: Record<string, unknown> = {
      map_id: value.mapId,
      // Backend currently requires private visibility for Party Lobby.
      visibility: "private",
    };
    if (value.roundCount != null) body.round_count = value.roundCount;
    if (value.timerSeconds !== undefined) body.timer_seconds = value.timerSeconds;
    if (value.maxPlayers != null) body.max_players = value.maxPlayers;
    if (value.displayName) body.display_name = value.displayName;

    const response = await apiJson("/rooms", roomSnapshotSchema, {
      method: "POST",
      body,
      idempotencyKey: value.idempotencyKey,
      requiresAuth: true,
      forwardCookies: true,
    });
    return { ok: true, roomCode: response.room.code };
  } catch (error) {
    if (
      error instanceof ApiError &&
      (error.status === 401 || error.status === 403)
    ) {
      redirect(`/${safe}/login?reason=session_expired` as Route);
    }
    return actionError(error);
  }
}

export async function joinRoomAction(
  input: unknown,
  locale: AppLocale = routing.defaultLocale,
): Promise<RoomActionResult> {
  const parsed = joinRoomInputSchema.safeParse(input);
  if (!parsed.success) return { ok: false, code: "validation_failed" };
  const value = parsed.data;
  const safe = safeLocale(locale);

  try {
    const body: Record<string, unknown> = { code: value.code };
    if (value.displayName) body.display_name = value.displayName;
    const response = await apiJson("/rooms/join", roomSnapshotSchema, {
      method: "POST",
      body,
      requiresAuth: true,
      forwardCookies: true,
    });
    return { ok: true, roomCode: response.room.code };
  } catch (error) {
    if (
      error instanceof ApiError &&
      (error.status === 401 || error.status === 403)
    ) {
      redirect(`/${safe}/login?reason=session_expired` as Route);
    }
    return actionError(error);
  }
}

export async function startRoomAction(
  roomCode: string,
): Promise<RoomActionResult> {
  const code = roomCodeSchema.safeParse(roomCode);
  if (!code.success) return { ok: false, code: "validation_failed" };
  try {
    const response = await apiJson(
      `/rooms/${encodeURIComponent(code.data)}/start`,
      roomSnapshotSchema,
      {
        method: "POST",
        body: {},
        idempotencyKey: `room-start-${code.data}-${crypto.randomUUID()}`,
        requiresAuth: true,
        forwardCookies: true,
      },
    );
    return {
      ok: true,
      roomCode: response.room.code,
      room: response.room,
    };
  } catch (error) {
    return actionError(error);
  }
}

/** Server Action for lobby polling / snapshot repair (HTTP is source of truth). */
export async function refreshRoomAction(
  roomCode: string,
): Promise<RoomActionResult> {
  const code = roomCodeSchema.safeParse(roomCode);
  if (!code.success) return { ok: false, code: "validation_failed" };
  try {
    const response = await apiJson(
      `/rooms/${encodeURIComponent(code.data)}`,
      roomSnapshotSchema,
      { requiresAuth: true },
    );
    return {
      ok: true,
      roomCode: response.room.code,
      room: response.room,
    };
  } catch (error) {
    return actionError(error);
  }
}

export async function leaveRoomSelfAction(
  roomCode: string,
): Promise<RoomActionResult> {
  const code = roomCodeSchema.safeParse(roomCode);
  if (!code.success) return { ok: false, code: "validation_failed" };
  try {
    const response = await apiFetch(
      `/rooms/${encodeURIComponent(code.data)}/players/me`,
      { method: "DELETE", requiresAuth: true },
    );
    if (response.status === 204 || response.ok) {
      return { ok: true, roomCode: code.data };
    }
    const text = await response.text();
    let json: unknown = null;
    try {
      json = text ? JSON.parse(text) : null;
    } catch {
      return { ok: false, code: "unavailable" };
    }
    throw parseError(json, response.status);
  } catch (error) {
    return actionError(error);
  }
}

export async function cancelRoomAction(
  roomCode: string,
): Promise<RoomActionResult> {
  const code = roomCodeSchema.safeParse(roomCode);
  if (!code.success) return { ok: false, code: "validation_failed" };
  try {
    const response = await apiFetch(`/rooms/${encodeURIComponent(code.data)}`, {
      method: "DELETE",
      requiresAuth: true,
    });
    if (response.status === 204 || response.ok) {
      return { ok: true, roomCode: code.data };
    }
    const text = await response.text();
    let json: unknown = null;
    try {
      json = text ? JSON.parse(text) : null;
    } catch {
      return { ok: false, code: "unavailable" };
    }
    throw parseError(json, response.status);
  } catch (error) {
    return actionError(error);
  }
}

function parseError(body: unknown, status: number): ApiError {
  if (
    body &&
    typeof body === "object" &&
    "error" in body &&
    body.error &&
    typeof body.error === "object" &&
    "code" in body.error
  ) {
    const err = body.error as { code: string; message?: string };
    return new ApiError(status, {
      code: err.code,
      message: err.message ?? "Request failed",
    });
  }
  return new ApiError(status, {
    code: "unavailable",
    message: "Request failed",
  });
}
