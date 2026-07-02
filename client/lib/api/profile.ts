import "server-only";

import { apiFetch } from "@/lib/api/client";
import type { GameHistoryResponse, ProfileResponse, PublicProfileResponse, UpdateProfileRequest } from "@/features/profile/types";

export async function getCurrentProfile() {
  return apiFetch<ProfileResponse>("/profile", { cache: "no-store" });
}

export async function updateProfile(request: UpdateProfileRequest) {
  return apiFetch<ProfileResponse>("/profile", {
    method: "PATCH",
    body: request,
    cache: "no-store",
  });
}

export async function getPublicProfile(userId: string) {
  return apiFetch<PublicProfileResponse>(`/users/${encodeURIComponent(userId)}/stats`, {
    cache: "no-store",
  });
}

export async function getGameHistory(userId: string, options: { limit?: number; cursor?: string } = {}) {
  const query = new URLSearchParams();
  if (options.limit) {
    query.set("limit", String(options.limit));
  }
  if (options.cursor) {
    query.set("cursor", options.cursor);
  }
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return apiFetch<GameHistoryResponse>(`/users/${encodeURIComponent(userId)}/games${suffix}`, {
    cache: "no-store",
  });
}
