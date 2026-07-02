import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "@/lib/api/errors";
import { updateProfileAction } from "./actions";
import type { ProfileResponse } from "./types";

vi.mock("next/cache", () => ({
  revalidatePath: vi.fn(),
}));

vi.mock("@/lib/api/profile", () => ({
  updateProfile: vi.fn(),
}));

const { updateProfile } = await import("@/lib/api/profile");

function formData(overrides: Record<string, string> = {}) {
  const data = new FormData();
  data.set("display_name", overrides.display_name ?? "Raven");
  data.set("avatar_url", overrides.avatar_url ?? "");
  data.set("country_code", overrides.country_code ?? "eg");
  data.set("locale", overrides.locale ?? "en");
  data.set("timezone", overrides.timezone ?? "");
  return data;
}

function profileResponse(): ProfileResponse {
  return {
    profile: {
      user_id: "user-1",
      email: "raven@example.com",
      display_name: "Raven",
      avatar_url: null,
      country_code: "EG",
      locale: "en",
      timezone: null,
      preferences: {},
      created_at: "2026-07-01T00:00:00Z",
      updated_at: "2026-07-01T00:00:00Z",
    },
    stats: { games_played: 0, total_score: 0, average_score: 0, best_score: 0, last_played_at: null },
    progress: { recent_games: [], page: { limit: 20, next_cursor: null } },
  };
}

describe("updateProfileAction", () => {
  beforeEach(() => {
    vi.mocked(updateProfile).mockReset();
  });

  it("normalizes form values and returns saved profile on success", async () => {
    vi.mocked(updateProfile).mockResolvedValue(profileResponse());

    const result = await updateProfileAction({ status: "idle" }, formData());

    expect(updateProfile).toHaveBeenCalledWith({
      display_name: "Raven",
      avatar_url: null,
      country_code: "EG",
      locale: "en",
      timezone: null,
    });
    expect(result.status).toBe("success");
  });

  it("maps validation field errors", async () => {
    vi.mocked(updateProfile).mockRejectedValue(
      new ApiError(400, {
        code: "validation_failed",
        message: "Invalid",
        fields: [{ name: "timezone", code: "invalid_format", message: "Invalid timezone" }],
      }),
    );

    const result = await updateProfileAction({ status: "idle" }, formData({ timezone: "Nope" }));

    expect(result).toEqual({ status: "error", code: "validation", fieldErrors: { timezone: "invalid_format" } });
  });

  it("maps rate-limited and unauthorized outcomes", async () => {
    vi.mocked(updateProfile).mockRejectedValueOnce(new ApiError(429, { code: "rate_limited", message: "Slow down" }));
    await expect(updateProfileAction({ status: "idle" }, formData())).resolves.toMatchObject({ status: "error", code: "rate_limited" });

    vi.mocked(updateProfile).mockRejectedValueOnce(new ApiError(401, { code: "unauthorized", message: "Sign in" }));
    await expect(updateProfileAction({ status: "idle" }, formData())).resolves.toMatchObject({ status: "error", code: "unauthorized" });
  });
});
