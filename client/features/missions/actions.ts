"use server";

import { revalidatePath } from "next/cache";
import { z } from "zod";
import { apiJson } from "@/lib/api/client";
import { locales } from "@/lib/i18n/routing";
import { missionSchema } from "./schemas";

const claimMissionInputSchema = z.object({
  locale: z.enum(locales),
  missionId: z.string().uuid(),
});

export type ClaimMissionState = Readonly<{
  status: "idle" | "success" | "error";
}>;

export const initialClaimMissionState: ClaimMissionState = { status: "idle" };

export async function claimMissionAction(
  _previousState: ClaimMissionState,
  formData: FormData,
): Promise<ClaimMissionState> {
  const parsed = claimMissionInputSchema.safeParse({
    locale: formData.get("locale"),
    missionId: formData.get("missionId"),
  });
  if (!parsed.success) {
    return { status: "error" };
  }

  try {
    await apiJson(`/missions/${parsed.data.missionId}/claim`, missionSchema, {
      method: "POST",
      idempotencyKey: crypto.randomUUID(),
      requiresAuth: true,
    });
    revalidatePath(`/${parsed.data.locale}/missions`);
    return { status: "success" };
  } catch {
    return { status: "error" };
  }
}
