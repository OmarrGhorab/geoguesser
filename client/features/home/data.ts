import "server-only";

import { apiJson } from "@/lib/api/client";
import {
  authenticatedHomeSchema,
  type AuthenticatedHomeData,
} from "@/features/home/schemas";

/**
 * Loads the authenticated home snapshot from GET /home.
 * Server-only, cache: no-store (via apiJson/apiFetch).
 */
export async function getAuthenticatedHome(): Promise<AuthenticatedHomeData> {
  return apiJson("/home", authenticatedHomeSchema);
}
