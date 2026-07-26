import { describe, expect, it } from "vitest";
import { gamePlayHref, gameResultsHref, playSetupHref } from "./routes";

describe("play routes", () => {
  const gameId = "11111111-1111-4111-8111-111111111111";

  it("builds localized setup paths with optional mode", () => {
    expect(playSetupHref("en")).toBe("/en/play");
    expect(playSetupHref("ar", "solo")).toBe("/ar/play?mode=solo");
    expect(playSetupHref("en", "quick_play")).toBe("/en/play?mode=quick_play");
    expect(playSetupHref("en", "practice")).toBe("/en/play?mode=practice");
  });

  it("builds localized active-game and results paths", () => {
    expect(gamePlayHref("en", gameId)).toBe(`/en/games/${gameId}`);
    expect(gameResultsHref("ar", gameId)).toBe(`/ar/games/${gameId}/results`);
  });
});
