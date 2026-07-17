import { describe, expect, it } from "vitest";
import type { DailyGameResults } from "./schemas";
import {
  buildResultBreakdown,
  nextResultBreakdownIndex,
} from "./result-breakdown";

const rounds = [
  {
    round_id: "20000000-0000-4000-8000-000000000002",
    round_number: 2,
    actual_location: {
      latitude: 30,
      longitude: 31,
      country_code: "EG",
    },
    guesses: [
      {
        id: "30000000-0000-4000-8000-000000000002",
        latitude: 0,
        longitude: 0,
        distance_meters: 0,
        score: 0,
        submitted_at: "2026-07-17T12:00:00Z",
        timed_out: true,
      },
    ],
  },
  {
    round_id: "20000000-0000-4000-8000-000000000001",
    round_number: 1,
    actual_location: {
      latitude: 48.85,
      longitude: 2.35,
      country_code: "FR",
    },
    guesses: [
      {
        id: "30000000-0000-4000-8000-000000000001",
        latitude: 51.5,
        longitude: -0.12,
        distance_meters: 344_000,
        score: 3_820,
        submitted_at: "2026-07-17T11:57:00Z",
        timed_out: false,
      },
    ],
  },
] satisfies DailyGameResults["rounds"];

describe("result breakdown", () => {
  it("orders rounds and hides a synthetic timeout coordinate", () => {
    const result = buildResultBreakdown(rounds);

    expect(result.map((round) => round.roundNumber)).toEqual([1, 2]);
    expect(result[0]?.guess).toMatchObject({ score: 3_820 });
    expect(result[1]?.guess).toBeNull();
    expect(result[1]?.answer).toEqual({ latitude: 30, longitude: 31 });
  });

  it("advances through every round and stops on the overview step", () => {
    expect(nextResultBreakdownIndex(0, 5)).toBe(1);
    expect(nextResultBreakdownIndex(4, 5)).toBe(5);
    expect(nextResultBreakdownIndex(5, 5)).toBe(5);
  });
});
