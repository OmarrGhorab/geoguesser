"use server";

import { revalidatePath } from "next/cache";
import { updateProfile } from "@/lib/api/profile";
import { ApiError } from "@/lib/api/errors";
import type { ProfileDTO } from "./types";

export type ProfileFormState =
  | { status: "idle" }
  | { status: "success"; profile: ProfileDTO }
  | { status: "error"; code: "validation" | "rate_limited" | "unauthorized" | "unexpected"; fieldErrors?: Record<string, string> };

export async function updateProfileAction(_prevState: ProfileFormState, formData: FormData): Promise<ProfileFormState> {
  const displayName = String(formData.get("display_name") ?? "").trim();
  const avatarUrl = String(formData.get("avatar_url") ?? "").trim();
  const countryCode = String(formData.get("country_code") ?? "").trim();
  const locale = String(formData.get("locale") ?? "").trim();
  const timezone = String(formData.get("timezone") ?? "").trim();

  try {
    const response = await updateProfile({
      display_name: displayName,
      avatar_url: avatarUrl === "" ? null : avatarUrl,
      country_code: countryCode === "" ? null : countryCode.toUpperCase(),
      locale: locale === "" ? undefined : locale,
      timezone: timezone === "" ? null : timezone,
    });
    revalidatePath("/profile");
    return { status: "success", profile: response.profile };
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.status === 401 || error.status === 403) {
        return { status: "error", code: "unauthorized" };
      }
      if (error.status === 429) {
        return { status: "error", code: "rate_limited" };
      }
      if (error.status === 400 && error.detail.fields) {
        const fieldErrors: Record<string, string> = {};
        for (const field of error.detail.fields) {
          fieldErrors[field.name] = field.code;
        }
        return { status: "error", code: "validation", fieldErrors };
      }
    }
    return { status: "error", code: "unexpected" };
  }
}
