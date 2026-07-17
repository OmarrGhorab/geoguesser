import { describe, expect, it } from "vitest";
import {
  challengeResultSchema,
  currentRoundSchema,
  dailyMissionSchema,
  gameResultsSchema,
} from "./schemas";

const challenge = {
  id: "11111111-1111-4111-8111-111111111111",
  type: "daily",
  seed: "2026-07-15",
  challenge_date: "2026-07-15",
  map: { id: "22222222-2222-4222-8222-222222222222" },
  settings: {
    round_count: 5,
    timer_seconds: null,
    movement_rules: "moving",
    scoring_version: 1,
  },
  status: "active",
};

const attempt = {
  id: "33333333-3333-4333-8333-333333333333",
  challenge_id: challenge.id,
  status: "completed",
  leaderboard_eligible: true,
  total_score: 15_000,
  current_round_number: 5,
  game_id: "44444444-4444-4444-8444-444444444444",
};

describe("daily mission schemas", () => {
  it("retains the latest completed game while the next daily game is pending", () => {
    const gameId = "44444444-4444-4444-8444-444444444444";
    const result = dailyMissionSchema.parse({
      challenge,
      attempt_state: {
        ...attempt,
        status: "pending",
        game_id: null,
        daily_game_number: 5,
      },
      last_completed_game_id: gameId,
      streak: {
        current_count: 1,
        best_count: 2,
        status: "active",
        protection_state: "none",
        guest_limited: false,
      },
      missions_summary: [],
      leaderboard_summary: { participants: 4 },
      daily_games_played: 4,
      daily_games_total: 5,
    });

    expect(result.last_completed_game_id).toBe(gameId);
    expect(result.attempt_state?.status).toBe("pending");
  });

  it("accepts an opaque panorama without leaking coordinates", () => {
    const response = currentRoundSchema.parse({
      round: {
        id: "55555555-5555-4555-8555-555555555555",
        round_number: 1,
        status: "active",
        media: { type: "panorama", panorama_id: "safe_Pano-12345" },
      },
    });

    expect(response.round.media.panorama_id).toBe("safe_Pano-12345");
    expect(response.round).not.toHaveProperty("latitude");
    expect(response.round).not.toHaveProperty("longitude");
  });

  it("requires exactly five persisted scores in the final breakdown", () => {
    const round_results = Array.from({ length: 5 }, (_, index) => ({
      round_number: index + 1,
      score: 3_000 - index * 100,
      distance_meters: 1_000 + index,
    }));
    const result = challengeResultSchema.parse({
      challenge,
      attempt,
      visible: true,
      total_score: round_results.reduce((sum, round) => sum + round.score, 0),
      total_distance_meters: 5_010,
      round_results,
    });

    expect(result.round_results).toHaveLength(5);
    expect(result.round_results.map((round) => round.score)).toEqual([
      3_000, 2_900, 2_800, 2_700, 2_600,
    ]);
  });

  it("rejects an incomplete final breakdown", () => {
    const result = challengeResultSchema.safeParse({
      challenge,
      attempt,
      visible: true,
      total_score: 1_000,
      round_results: [{ round_number: 1, score: 1_000, distance_meters: 500 }],
    });

    expect(result.success).toBe(false);
  });

  it("accepts five owned game rounds with guesses and correct positions", () => {
    const result = gameResultsSchema.parse({
      game: {
        id: attempt.game_id,
        mode: "daily",
        status: "completed",
        map_id: challenge.map.id,
        round_count: 5,
        scoring_version: 1,
        total_score: 15_000,
      },
      players: [],
      rounds: Array.from({ length: 5 }, (_, index) => ({
        round_id: `${index + 5}5555555-5555-4555-8555-555555555555`,
        round_number: index + 1,
        actual_location: {
          latitude: 10 + index,
          longitude: 20 + index,
          country_code: "EG",
        },
        guesses: [
          {
            id: `${index + 1}6666666-6666-4666-8666-666666666666`,
            latitude: 11 + index,
            longitude: 21 + index,
            distance_meters: 1_000 + index,
            score: 3_000 - index * 100,
            submitted_at: "2026-07-16T08:00:00Z",
          },
        ],
      })),
    });

    expect(result.rounds).toHaveLength(5);
    expect(result.rounds[0]?.actual_location.country_code).toBe("EG");
    expect(result.rounds[0]?.guesses[0]?.score).toBe(3_000);
  });
});
