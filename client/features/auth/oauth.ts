import "server-only";

import { getBackendApiUrl } from "@/lib/env";

export function getOAuthUrls() {
  const base = getBackendApiUrl();
  return {
    google: `${base}/auth/oauth/google`,
    discord: `${base}/auth/oauth/discord`,
  } as const;
}
