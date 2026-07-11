import { describe, expect, it } from "vitest";
import en from "./en.json";
import ar from "./ar.json";

function collectKeys(value: unknown, prefix = ""): string[] {
  if (value == null || typeof value !== "object" || Array.isArray(value)) {
    return prefix ? [prefix] : [];
  }
  return Object.entries(value as Record<string, unknown>).flatMap(([key, child]) =>
    collectKeys(child, prefix ? `${prefix}.${key}` : key),
  );
}

describe("message catalog parity", () => {
  it("keeps Matchmaking keys aligned between en and ar", () => {
    const enKeys = collectKeys(en.Matchmaking, "Matchmaking").sort();
    const arKeys = collectKeys(ar.Matchmaking, "Matchmaking").sort();
    expect(arKeys).toEqual(enKeys);
  });

  it("keeps RankedGame keys aligned between en and ar", () => {
    const enKeys = collectKeys(en.RankedGame, "RankedGame").sort();
    const arKeys = collectKeys(ar.RankedGame, "RankedGame").sort();
    expect(arKeys).toEqual(enKeys);
  });
});
