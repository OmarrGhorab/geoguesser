import { describe, expect, it } from "vitest";
import {
  createPracticeGameInputSchema,
  createQuickPlayInputSchema,
  createSoloGameInputSchema,
  currentRoundSchema,
  gameResultsSchema,
  playActionResultSchema,
  playModeQuerySchema,
  practiceHistorySchema,
  submitGuessBodySchema,
} from "./schemas";

const gameId = "11111111-1111-4111-8111-111111111111";
const mapId = "22222222-2222-4222-8222-222222222222";
const key = "solo-create-key-16chars";

describe("playModeQuerySchema", () => {
  it("accepts solo, quick_play, and practice only", () => {
    expect(playModeQuerySchema.parse("solo")).toBe("solo");
    expect(playModeQuerySchema.parse("quick_play")).toBe("quick_play");
    expect(playModeQuerySchema.parse("practice")).toBe("practice");
    expect(playModeQuerySchema.safeParse("daily").success).toBe(false);
    expect(playModeQuerySchema.safeParse("ranked_solo").success).toBe(false);
  });
});

describe("create input schemas", () => {
  it("validates classic solo configuration", () => {
    const value = createSoloGameInputSchema.parse({
      mapId,
      roundCount: 5,
      timerSeconds: 60,
      idempotencyKey: key,
    });
    expect(value.roundCount).toBe(5);
  });

  it("requires only map and key for practice", () => {
    const value = createPracticeGameInputSchema.parse({
      mapId,
      idempotencyKey: key,
    });
    expect(value.mapId).toBe(mapId);
  });

  it("requires a durable key for quick play", () => {
    expect(
      createQuickPlayInputSchema.safeParse({ idempotencyKey: "short" }).success,
    ).toBe(false);
    expect(
      createQuickPlayInputSchema.parse({ idempotencyKey: key }).idempotencyKey,
    ).toBe(key);
  });
});

describe("playActionResultSchema", () => {
  it("discriminates success and failure", () => {
    expect(playActionResultSchema.parse({ ok: true, gameId })).toEqual({
      ok: true,
      gameId,
    });
    expect(
      playActionResultSchema.parse({ ok: false, code: "unavailable" }),
    ).toEqual({ ok: false, code: "unavailable" });
  });
});

describe("gameplay response schemas", () => {
  it("accepts opaque current-round media without coordinates", () => {
    const response = currentRoundSchema.parse({
      round: {
        id: "55555555-5555-4555-8555-555555555555",
        round_number: 12,
        status: "active",
        media: { type: "panorama", panorama_id: "safe_Pano-12345" },
      },
    });
    expect(response.round.round_number).toBe(12);
    expect(response.round).not.toHaveProperty("latitude");
  });

  it("accepts variable-length game results", () => {
    const result = gameResultsSchema.parse({
      game: {
        id: gameId,
        mode: "solo",
        status: "completed",
        map_id: mapId,
        round_count: 3,
        scoring_version: 1,
        current_round_number: 3,
        total_score: 9000,
      },
      players: [
        {
          id: "33333333-3333-4333-8333-333333333333",
          display_name: "Explorer",
          role: "host",
          status: "active",
          total_score: 9000,
        },
      ],
      rounds: [
        {
          round_id: "44444444-4444-4444-8444-444444444444",
          round_number: 1,
          actual_location: {
            latitude: 30,
            longitude: 31,
            country_code: "EG",
          },
          guesses: [
            {
              id: "66666666-6666-4666-8666-666666666666",
              latitude: 30.1,
              longitude: 31.1,
              distance_meters: 1200,
              score: 4000,
              submitted_at: "2026-07-18T10:00:00Z",
              timed_out: false,
            },
          ],
        },
      ],
    });
    expect(result.rounds).toHaveLength(1);
  });

  it("parses practice history pages", () => {
    const page = practiceHistorySchema.parse({
      items: [],
      has_more: false,
      next_cursor: null,
    });
    expect(page.has_more).toBe(false);
  });

  it("requires coordinates and idempotency for guesses", () => {
    expect(
      submitGuessBodySchema.safeParse({
        latitude: 30,
        longitude: 31,
        idempotencyKey: key,
      }).success,
    ).toBe(true);
    expect(
      submitGuessBodySchema.safeParse({
        latitude: 100,
        longitude: 31,
        idempotencyKey: key,
      }).success,
    ).toBe(false);
  });
});
