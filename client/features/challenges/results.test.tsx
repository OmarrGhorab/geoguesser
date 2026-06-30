import type { AttemptSummary } from "./types";

export function shouldHideSpoilers(attempt?: AttemptSummary) {
  return !attempt || attempt.status !== "completed";
}

export const completedAttemptFixture: AttemptSummary = {
  id: "00000000-0000-0000-0000-000000000020",
  challenge_id: "00000000-0000-0000-0000-000000000021",
  status: "completed",
  leaderboard_eligible: true,
  total_score: 22000,
};
