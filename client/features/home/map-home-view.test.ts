import { describe, expect, it } from "vitest";
import {
  formatAverageScore,
  formatNumber,
  formatPlayedToday,
  formatWelcome,
  weekContaining,
} from "./format";
import type { AuthenticatedHomeData } from "./schemas";

/**
 * Pure mapping assertions for how AuthenticatedHome consumes GET /home data.
 * Drives the same format helpers the components use.
 */
function mapHomeView(data: AuthenticatedHomeData, locale: "en" | "ar") {
  const challengeDate = data.daily_challenge.challenge.challenge_date;
  return {
    welcome: formatWelcome("Welcome back, {name}!", data.viewer.display_name),
    viewerName: data.viewer.display_name,
    avatarUrl: data.viewer.avatar_url ?? null,
    gamesPlayed: formatNumber(locale, data.stats.games_played),
    averageScore: formatAverageScore(locale, data.stats.average_score),
    bestScore: formatNumber(locale, data.stats.best_score),
    streak: data.daily_challenge.streak.current_count,
    playersToday: formatPlayedToday(
      locale,
      "{count} have played today",
      data.daily_challenge.leaderboard_summary.participants,
    ),
    week: weekContaining(challengeDate ?? undefined),
    mapNames: data.recommended_maps.map((m) => m.name),
    mapCount: data.recommended_maps.length,
    resetEndsAt: data.daily_challenge.countdown?.reset_ends_at ?? null,
  };
}

const baseData: AuthenticatedHomeData = {
  viewer: {
    user_id: "11111111-1111-4111-8111-111111111111",
    display_name: "MapMaster",
    avatar_url: "https://cdn.example.com/a.png",
  },
  stats: {
    games_played: 248,
    total_score: 100000,
    average_score: 1234.6,
    best_score: 24680,
  },
  daily_challenge: {
    challenge: {
      id: "22222222-2222-4222-8222-222222222222",
      type: "daily",
      seed: "s",
      challenge_date: "2026-07-15",
      map: { id: "33333333-3333-4333-8333-333333333333" },
      settings: {
        round_count: 5,
        movement_rules: "moving",
        scoring_version: 1,
      },
      status: "active",
    },
    streak: {
      current_count: 7,
      best_count: 20,
      status: "active",
      protection_state: "none",
      guest_limited: false,
    },
    missions_summary: [],
    leaderboard_summary: { participants: 122690 },
    countdown: {
      reset_ends_at: "2026-07-16T00:00:00Z",
      seconds_remaining: 100,
    },
  },
  recommended_maps: [
    {
      id: "44444444-4444-4444-8444-444444444444",
      slug: "world",
      name: "World",
      visibility: "public",
      access_tier: "free",
      difficulty: "easy",
      status: "active",
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    },
  ],
};

describe("mapHomeView from GET /home fields", () => {
  it("maps viewer, stats, daily, and maps", () => {
    const view = mapHomeView(baseData, "en");
    expect(view.welcome).toBe("Welcome back, MapMaster!");
    expect(view.viewerName).toBe("MapMaster");
    expect(view.avatarUrl).toBe("https://cdn.example.com/a.png");
    expect(view.gamesPlayed).toBe("248");
    expect(view.averageScore).toBe("1,234.6");
    expect(view.bestScore).toBe("24,680");
    expect(view.streak).toBe(7);
    expect(view.playersToday).toBe("122,690 have played today");
    expect(view.week.todayDay).toBe(15);
    expect(view.mapNames).toEqual(["World"]);
    expect(view.resetEndsAt).toBe("2026-07-16T00:00:00Z");
  });

  it("handles empty maps and missing avatar", () => {
    const view = mapHomeView(
      {
        ...baseData,
        viewer: {
          user_id: baseData.viewer.user_id,
          display_name: "Solo",
        },
        recommended_maps: [],
      },
      "en",
    );
    expect(view.avatarUrl).toBeNull();
    expect(view.mapCount).toBe(0);
    expect(view.welcome).toBe("Welcome back, Solo!");
  });
});
