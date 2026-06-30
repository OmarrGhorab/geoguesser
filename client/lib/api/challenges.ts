import "server-only";

import { apiFetch } from "@/lib/api/client";
import type { ChallengeAttemptResponse, ChallengeMetadataResponse } from "@/features/challenges/types";

export async function getDailyChallenge(date?: string) {
  const suffix = date ? `?date=${encodeURIComponent(date)}` : "";
  return apiFetch<ChallengeMetadataResponse>(`/challenges/daily${suffix}`, {
    cache: "no-store",
  });
}

export async function startDailyChallenge() {
  return apiFetch<ChallengeAttemptResponse>("/challenges/daily/attempts", {
    method: "POST",
    cache: "no-store",
  });
}

export async function getSharedChallenge(code: string) {
  return apiFetch<ChallengeMetadataResponse>(`/challenges/shared/${encodeURIComponent(code)}`, {
    cache: "no-store",
  });
}

export async function startChallenge(challengeId: string) {
  return apiFetch<ChallengeAttemptResponse>(`/challenges/${encodeURIComponent(challengeId)}/attempts`, {
    method: "POST",
    cache: "no-store",
  });
}
