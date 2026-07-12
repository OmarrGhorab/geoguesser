import { z } from "zod";
import type { AppLocale } from "@/lib/i18n/routing";

export const oauthProviderSchema = z.enum(["google", "discord"]);
export type OAuthProvider = z.infer<typeof oauthProviderSchema>;

export function oauthStartPath(
  provider: OAuthProvider,
  locale: AppLocale,
): string {
  return `/api/auth/oauth/${provider}?locale=${locale}`;
}
