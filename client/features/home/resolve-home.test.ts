import { describe, expect, it } from "vitest";
import { ApiError } from "@/lib/api/errors";
import type { AuthenticatedHomeData } from "./schemas";
import { decideHomeView, loadHomeDecision } from "./resolve-home";

const sample: AuthenticatedHomeData = {
  viewer: {
    user_id: "11111111-1111-4111-8111-111111111111",
    display_name: "Explorer",
  },
  stats: {
    games_played: 0,
    total_score: 0,
    average_score: 0,
    best_score: 0,
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

describe("decideHomeView", () => {
  it("renders public when no auth cookie", () => {
    expect(
      decideHomeView(false, { ok: true, data: sample }).kind,
    ).toBe("public");
  });

  it("renders authenticated on successful /home", () => {
    const decision = decideHomeView(true, { ok: true, data: sample });
    expect(decision.kind).toBe("authenticated");
    if (decision.kind === "authenticated") {
      expect(decision.data.viewer.display_name).toBe("Explorer");
    }
  });

  it("renders public on 401 from /home", () => {
    const decision = decideHomeView(true, {
      ok: false,
      error: new ApiError(401, {
        code: "unauthorized",
        message: "Authentication is required.",
      }),
    });
    expect(decision.kind).toBe("public");
  });

  it("renders unavailable on 503 from /home", () => {
    const decision = decideHomeView(true, {
      ok: false,
      error: new ApiError(503, {
        code: "service_unavailable",
        message: "Home is unavailable.",
      }),
    });
    expect(decision.kind).toBe("unavailable");
  });
});

describe("loadHomeDecision", () => {
  it("loads authenticated data via the provided loader", async () => {
    const decision = await loadHomeDecision(true, async () => sample);
    expect(decision.kind).toBe("authenticated");
  });

  it("maps loader 401 to public", async () => {
    const decision = await loadHomeDecision(true, async () => {
      throw new ApiError(401, {
        code: "unauthorized",
        message: "Authentication is required.",
      });
    });
    expect(decision.kind).toBe("public");
  });
});
