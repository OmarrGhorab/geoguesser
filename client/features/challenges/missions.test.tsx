import type { MissionSummary, StreakSummary } from "./types";

export const missionFixture: MissionSummary = {
  code: "daily_completion",
  title_key: "Challenges.missions.dailyCompletion.title",
  description_key: "Challenges.missions.dailyCompletion.description",
  mission_type: "daily_completion",
  current_value: 1,
  target_value: 1,
  status: "completed",
};

export const guestStreakFixture: StreakSummary = {
  current_count: 1,
  best_count: 1,
  status: "active",
  protection_state: "none",
  guest_limited: true,
};
