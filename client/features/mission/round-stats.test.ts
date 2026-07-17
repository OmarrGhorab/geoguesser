import { describe, expect, it } from "vitest";
import { formatRoundTimer, roundStatValue } from "./round-stats";

describe("daily game round stats", () => {
  it("formats the live server-backed timer", () => {
    expect(formatRoundTimer(180)).toBe("3:00");
    expect(formatRoundTimer(11)).toBe("0:11");
    expect(formatRoundTimer(-1)).toBe("0:00");
  });

  it("uses recorded values and never invents future round times", () => {
    const completed = { 1: "2:41" };
    expect(roundStatValue(1, 2, "1:48", completed)).toBe("2:41");
    expect(roundStatValue(2, 2, "1:48", completed)).toBe("1:48");
    expect(roundStatValue(3, 2, "1:48", completed)).toBe("");
  });

  it("marks resumed earlier rounds as completed when their timings are unavailable", () => {
    expect(roundStatValue(1, 3, "2:00", {})).toBe("✓");
  });
});
