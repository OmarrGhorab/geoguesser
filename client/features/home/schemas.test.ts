import { describe, expect, it } from "vitest";
import { authenticatedHomeSchema } from "./schemas";

const sampleHome = {
  viewer: {
    user_id: "11111111-1111-4111-8111-111111111111",
    display_name: "Explorer",
    avatar_url: null,
    country_code: "US",
  },
  stats: {
    games_played: 12,
    total_score: 48000,
    average_score: 4000.5,
    best_score: 9200,
    last_played_at: "2026-07-14T18:00:00Z",
  },
  daily_challenge: {
    challenge: {
      id: "22222222-2222-4222-8222-222222222222",
      type: "daily",
      seed: "seed-1",
      challenge_date: "2026-07-15",
      reset_starts_at: "2026-07-15T00:00:00Z",
      reset_ends_at: "2026-07-16T00:00:00Z",
      map: { id: "33333333-3333-4333-8333-333333333333" },
      settings: {
        round_count: 5,
        timer_seconds: 60,
        movement_rules: "moving",
        scoring_version: 1,
      },
      status: "active",
    },
    streak: {
      current_count: 3,
      best_count: 10,
      status: "active",
      protection_state: "none",
      guest_limited: false,
    },
    missions_summary: [],
    leaderboard_summary: { participants: 122690 },
    countdown: {
      reset_ends_at: "2026-07-16T00:00:00Z",
      seconds_remaining: 3600,
    },
  },
  recommended_maps: [
    {
      id: "44444444-4444-4444-8444-444444444444",
      slug: "world",
      name: "World",
      description: null,
      visibility: "public",
      access_tier: "free",
      difficulty: "easy",
      status: "active",
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    },
  ],
};

describe("authenticatedHomeSchema", () => {
  it("parses a valid GET /home payload", () => {
    const result = authenticatedHomeSchema.safeParse(sampleHome);
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.viewer.display_name).toBe("Explorer");
      expect(result.data.stats.games_played).toBe(12);
      expect(result.data.daily_challenge.streak.current_count).toBe(3);
      expect(result.data.recommended_maps).toHaveLength(1);
    }
  });

  it("accepts empty recommended_maps and missing avatar", () => {
    const result = authenticatedHomeSchema.safeParse({
      ...sampleHome,
      viewer: {
        user_id: sampleHome.viewer.user_id,
        display_name: "NoAvatar",
      },
      recommended_maps: [],
      daily_challenge: {
        ...sampleHome.daily_challenge,
        countdown: null,
        missions_summary: null,
      },
    });
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.viewer.avatar_url).toBeUndefined();
      expect(result.data.recommended_maps).toEqual([]);
      expect(result.data.daily_challenge.missions_summary).toEqual([]);
    }
  });

  it("rejects invalid user_id", () => {
    const result = authenticatedHomeSchema.safeParse({
      ...sampleHome,
      viewer: { ...sampleHome.viewer, user_id: "not-a-uuid" },
    });
    expect(result.success).toBe(false);
  });
});
