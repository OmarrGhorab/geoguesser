import { describe, expect, it } from "vitest";
import {
  formatAverageScore,
  formatDate,
  formatNumber,
  formatPlayedToday,
  formatWelcome,
  weekContaining,
} from "./format";

describe("home format helpers", () => {
  it("formats numbers for en and ar", () => {
    expect(formatNumber("en", 122690)).toBe("122,690");
    expect(formatNumber("ar", 122690)).toMatch(/122/);
  });

  it("formats average scores with one fraction digit", () => {
    expect(formatAverageScore("en", 4000.5)).toBe("4,000.5");
  });

  it("formats challenge dates in locale", () => {
    const en = formatDate("en", "2026-07-15");
    expect(en).toMatch(/2026/);
    expect(en.toLowerCase()).toMatch(/july|15/);
    const ar = formatDate("ar", "2026-07-15");
    expect(ar).toMatch(/2026/);
  });

  it("builds week days around challenge date", () => {
    // 2026-07-15 is a Wednesday
    const week = weekContaining("2026-07-15");
    expect(week.days).toHaveLength(7);
    expect(week.todayDay).toBe(15);
    expect(week.days).toContain(15);
  });

  it("fills welcome and players-today templates", () => {
    expect(formatWelcome("Welcome back, {name}!", "Alex")).toBe(
      "Welcome back, Alex!",
    );
    expect(formatPlayedToday("en", "{count} have played today", 1000)).toBe(
      "1,000 have played today",
    );
  });
});
