import { describe, expect, it } from "vitest";
import { dailyMissionPlayHref, dailyMissionResultsHref } from "./routes";

describe("daily mission routes", () => {
  const gameId = "11111111-1111-4111-8111-111111111111";

  it("uses a semantic localized gameplay path without query parameters", () => {
    expect(dailyMissionPlayHref("en", gameId)).toBe(
      `/en/daily-mission/play/${gameId}`,
    );
  });

  it("uses a semantic localized result path without query parameters", () => {
    expect(dailyMissionResultsHref("ar", gameId)).toBe(
      `/ar/daily-mission/results/${gameId}`,
    );
  });
});
