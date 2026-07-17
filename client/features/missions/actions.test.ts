import { beforeEach, describe, expect, it, vi } from "vitest";

const apiJson = vi.fn();
const revalidatePath = vi.fn();

vi.mock("@/lib/api/client", () => ({
  apiJson: (...args: unknown[]) => apiJson(...args),
}));
vi.mock("next/cache", () => ({ revalidatePath }));

const mission = {
  id: "11111111-1111-4111-8111-111111111111",
  code: "daily:2026-07-17:daily_completion",
  mission_key: "daily_completion",
  title_key: "Challenges.missions.dailyCompletion.title",
  description_key: "Challenges.missions.dailyCompletion.description",
  mission_type: "daily_completion",
  cadence: "daily",
  period_key: "2026-07-17",
  icon_key: "daily-completion",
  reward_xp: 75,
  current_value: 1,
  target_value: 1,
  status: "claimed",
};

function claimForm(overrides: Record<string, string> = {}): FormData {
  const form = new FormData();
  form.set("locale", overrides.locale ?? "en");
  form.set("missionId", overrides.missionId ?? mission.id);
  return form;
}

describe("claimMissionAction", () => {
  beforeEach(() => {
    apiJson.mockReset();
    revalidatePath.mockReset();
  });

  it("claims with an idempotency key and revalidates the localized route", async () => {
    apiJson.mockResolvedValueOnce(mission);
    const { claimMissionAction, initialClaimMissionState } =
      await import("./actions");

    await expect(
      claimMissionAction(initialClaimMissionState, claimForm()),
    ).resolves.toEqual({ status: "success" });
    expect(apiJson).toHaveBeenCalledWith(
      `/missions/${mission.id}/claim`,
      expect.anything(),
      expect.objectContaining({
        method: "POST",
        requiresAuth: true,
        idempotencyKey: expect.any(String),
      }),
    );
    expect(revalidatePath).toHaveBeenCalledWith("/en/missions");
  });

  it("rejects forged input without calling the API", async () => {
    const { claimMissionAction, initialClaimMissionState } =
      await import("./actions");

    await expect(
      claimMissionAction(
        initialClaimMissionState,
        claimForm({ locale: "fr", missionId: "not-a-uuid" }),
      ),
    ).resolves.toEqual({ status: "error" });
    expect(apiJson).not.toHaveBeenCalled();
    expect(revalidatePath).not.toHaveBeenCalled();
  });

  it("returns a safe error state when the API rejects the claim", async () => {
    apiJson.mockRejectedValueOnce(new Error("conflict"));
    const { claimMissionAction, initialClaimMissionState } =
      await import("./actions");

    await expect(
      claimMissionAction(initialClaimMissionState, claimForm()),
    ).resolves.toEqual({ status: "error" });
    expect(revalidatePath).not.toHaveBeenCalled();
  });
});
