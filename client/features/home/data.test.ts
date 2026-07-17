import { beforeEach, describe, expect, it, vi } from "vitest";

const apiJson = vi.fn();

vi.mock("@/lib/api/client", () => ({
  apiJson: (...args: unknown[]) => apiJson(...args),
}));

describe("getAuthenticatedHome", () => {
  beforeEach(() => {
    apiJson.mockReset();
  });

  it("calls only GET /home via apiJson with the home schema", async () => {
    const payload = {
      viewer: {
        user_id: "11111111-1111-4111-8111-111111111111",
        display_name: "Explorer",
      },
      stats: {
        games_played: 1,
        total_score: 100,
        average_score: 100,
        best_score: 100,
      },
      daily_challenge: {
        challenge: {
          id: "22222222-2222-4222-8222-222222222222",
          type: "daily",
          seed: "s",
          map: { id: "33333333-3333-4333-8333-333333333333" },
          settings: {
            round_count: 5,
            movement_rules: "moving",
            scoring_version: 1,
          },
          status: "active",
        },
        streak: {
          current_count: 0,
          best_count: 0,
          status: "none",
          protection_state: "none",
          guest_limited: false,
        },
        missions_summary: [],
        leaderboard_summary: { participants: 0 },
      },
      recommended_maps: [],
    };
    apiJson.mockResolvedValueOnce(payload);

    const { getAuthenticatedHome } = await import("./data");
    const { authenticatedHomeSchema } = await import("./schemas");
    const result = await getAuthenticatedHome();

    expect(apiJson).toHaveBeenCalledTimes(1);
    expect(apiJson).toHaveBeenCalledWith("/home", authenticatedHomeSchema);
    expect(result.viewer.display_name).toBe("Explorer");
  });
});
