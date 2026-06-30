import type { ChallengeMetadataResponse } from "./types";

export const sharedChallengeFixture: ChallengeMetadataResponse = {
  challenge: {
    id: "00000000-0000-0000-0000-000000000010",
    type: "shared",
    seed: "shared-seed",
    share_code: "ABCDE12345",
    map: { id: "00000000-0000-0000-0000-000000000011" },
    settings: { round_count: 5, timer_seconds: 120, movement_rules: "standard", scoring_version: 1 },
    status: "active",
  },
  streak: {
    current_count: 0,
    best_count: 0,
    status: "inactive",
    protection_state: "none",
    guest_limited: false,
  },
  missions_summary: [],
  leaderboard_summary: { participants: 0 },
};

export function stableSharedIdentity(data: ChallengeMetadataResponse) {
  return data.challenge.type === "shared" && Boolean(data.challenge.share_code) && Boolean(data.challenge.seed);
}
