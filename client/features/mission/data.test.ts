import { beforeEach, describe, expect, it, vi } from "vitest";

const apiJson = vi.fn();

vi.mock("@/lib/api/client", () => ({
  apiJson: (...args: unknown[]) => apiJson(...args),
}));

describe("daily mission data", () => {
  beforeEach(() => apiJson.mockReset());

  it("loads date-specific metadata for a selected historical mission", async () => {
    apiJson.mockResolvedValueOnce({ challenge: {} });

    const { getDailyMission } = await import("./data");
    await getDailyMission("2026-07-16");

    expect(apiJson.mock.calls[0]?.[0]).toBe(
      "/challenges/daily?date=2026-07-16",
    );
  });

  it("loads the game and only the current round for an active attempt", async () => {
    const game = {
      id: "11111111-1111-4111-8111-111111111111",
      mode: "daily",
      status: "active",
      map_id: "22222222-2222-4222-8222-222222222222",
      round_count: 5,
      scoring_version: 1,
      current_round_number: 2,
      total_score: 4_500,
    };
    const round = {
      id: "33333333-3333-4333-8333-333333333333",
      round_number: 2,
      status: "active",
      media: { type: "panorama", panorama_id: "safe_Pano-12345" },
    };
    apiJson.mockResolvedValueOnce({ game }).mockResolvedValueOnce({ round });

    const { getDailyGameState } = await import("./data");
    const state = await getDailyGameState(game.id);

    expect(apiJson).toHaveBeenCalledTimes(2);
    expect(apiJson.mock.calls[0]?.[0]).toBe(`/games/${game.id}`);
    expect(apiJson.mock.calls[1]?.[0]).toBe(`/games/${game.id}/rounds/current`);
    expect(state.round?.round_number).toBe(2);
  });

  it("does not request another round for a completed attempt", async () => {
    const game = {
      id: "11111111-1111-4111-8111-111111111111",
      mode: "daily",
      status: "completed",
      map_id: "22222222-2222-4222-8222-222222222222",
      round_count: 5,
      scoring_version: 1,
      current_round_number: 5,
      total_score: 20_000,
    };
    apiJson.mockResolvedValueOnce({ game });

    const { getDailyGameState } = await import("./data");
    const state = await getDailyGameState(game.id);

    expect(apiJson).toHaveBeenCalledTimes(1);
    expect(state.round).toBeNull();
  });

  it("loads the owned completed game results for the semantic result route", async () => {
    const gameId = "11111111-1111-4111-8111-111111111111";
    const results = { game: { id: gameId }, players: [], rounds: [] };
    apiJson.mockResolvedValueOnce(results);

    const { getDailyGameResults } = await import("./data");
    await expect(getDailyGameResults(gameId)).resolves.toEqual(results);
    expect(apiJson.mock.calls[0]?.[0]).toBe(`/games/${gameId}/results`);
  });
});
