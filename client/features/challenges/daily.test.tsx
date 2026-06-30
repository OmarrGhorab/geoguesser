import type { ChallengeMetadataResponse } from "./types";

export const dailyChallengeFixture: ChallengeMetadataResponse = {
  challenge: {
    id: "00000000-0000-0000-0000-000000000001",
    type: "daily",
    seed: "daily-seed",
    challenge_date: "2026-06-27",
    map: { id: "00000000-0000-0000-0000-000000000002" },
    settings: { round_count: 5, timer_seconds: null, movement_rules: "standard", scoring_version: 1 },
    status: "active",
  },
  streak: {
    current_count: 1,
    best_count: 3,
    status: "active",
    protection_state: "none",
    guest_limited: true,
  },
  missions_summary: [],
  leaderboard_summary: { participants: 0 },
  countdown: {
    reset_ends_at: "2026-06-28T00:00:00Z",
    seconds_remaining: 3600,
  },
};

export function hasLockedDailySettings(data: ChallengeMetadataResponse) {
  return data.challenge.type === "daily" && data.challenge.settings.round_count > 0 && data.challenge.seed.length > 0;
}
