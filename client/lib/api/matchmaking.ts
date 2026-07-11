import "server-only";

import { apiFetch } from "@/lib/api/client";
import type { JoinMatchmakingRequest, MatchmakingStatusResponse } from "@/features/matchmaking/types";

export async function joinMatchmaking(request: JoinMatchmakingRequest) {
  return apiFetch<MatchmakingStatusResponse>("/matchmaking/queue", {
    method: "POST",
    body: request,
    cache: "no-store",
  });
}

export async function leaveMatchmaking() {
  return apiFetch<void>("/matchmaking/queue", {
    method: "DELETE",
    cache: "no-store",
  });
}

export async function getMatchmakingStatus() {
  return apiFetch<MatchmakingStatusResponse>("/matchmaking/status", {
    cache: "no-store",
  });
}
