import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api/matchmaking", () => ({
  joinMatchmaking: vi.fn(),
  leaveMatchmaking: vi.fn(),
}));

import { joinMatchmaking, leaveMatchmaking } from "@/lib/api/matchmaking";
import { ApiError } from "@/lib/api/errors";
import { joinMatchmakingAction, leaveMatchmakingAction } from "./actions";

describe("matchmaking actions", () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it("rejects unsupported modes without calling the API", async () => {
    const result = await joinMatchmakingAction("ranked_season" as "ranked_standard");
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.code).toBe("unsupported_matchmaking_mode");
    }
    expect(joinMatchmaking).not.toHaveBeenCalled();
  });

  it("returns searching status on successful join", async () => {
    vi.mocked(joinMatchmaking).mockResolvedValue({
      status: "searching",
      queue: {
        mode: "ranked_standard",
        search_started_at: "2026-07-11T12:00:00Z",
        lease_expires_at: "2026-07-11T12:00:30Z",
      },
      match: null,
    });
    const result = await joinMatchmakingAction("ranked_standard");
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.status.status).toBe("searching");
    }
  });

  it("maps backend errors safely", async () => {
    vi.mocked(joinMatchmaking).mockRejectedValue(
      new ApiError(403, { code: "matchmaking_account_ineligible", message: "Not eligible." }),
    );
    const result = await joinMatchmakingAction();
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.code).toBe("matchmaking_account_ineligible");
      expect(result.message).toBe("Not eligible.");
    }
  });

  it("leave returns not_queued on success", async () => {
    vi.mocked(leaveMatchmaking).mockResolvedValue(undefined);
    const result = await leaveMatchmakingAction();
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.status.status).toBe("not_queued");
    }
  });
});
