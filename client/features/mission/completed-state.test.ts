import { describe, expect, it } from "vitest";
import {
  resolveCompletedDailyGameId,
  shouldShowPastDailyEmptyState,
} from "./completed-state";

describe("completed daily mission state", () => {
  const today = "2026-07-16";
  const gameId = "11111111-1111-4111-8111-111111111111";

  it("selects the result dashboard for the selected completed mission", () => {
    expect(
      resolveCompletedDailyGameId(today, today, {
        status: "completed",
        game_id: gameId,
      }),
    ).toBe(gameId);
  });

  it("keeps the regular screen for an active attempt", () => {
    expect(
      resolveCompletedDailyGameId(today, today, {
        status: "active",
        game_id: gameId,
      }),
    ).toBeNull();
  });

  it("keeps the played-today dashboard while the next daily game is pending", () => {
    expect(
      resolveCompletedDailyGameId(
        today,
        today,
        { status: "pending", game_id: null },
        gameId,
      ),
    ).toBe(gameId);
  });

  it("does not reuse a result from a different challenge date", () => {
    expect(
      resolveCompletedDailyGameId(
        "2026-07-15",
        today,
        {
          status: "completed",
          game_id: gameId,
        },
        gameId,
      ),
    ).toBeNull();
  });

  it("shows the passed state only for an uncompleted historical mission", () => {
    expect(shouldShowPastDailyEmptyState("2026-07-15", today, null)).toBe(true);
    expect(shouldShowPastDailyEmptyState(today, today, null)).toBe(false);
    expect(shouldShowPastDailyEmptyState("2026-07-15", today, gameId)).toBe(
      false,
    );
  });
});
