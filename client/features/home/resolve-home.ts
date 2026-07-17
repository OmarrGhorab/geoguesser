import type { AuthenticatedHomeData } from "@/features/home/schemas";
import { ApiError } from "@/lib/api/errors";

export type HomeViewKind = "public" | "authenticated" | "unavailable";

export type HomeViewDecision =
  | { kind: "public" }
  | { kind: "authenticated"; data: AuthenticatedHomeData }
  | { kind: "unavailable" };

/**
 * Cookie presence is only a hint. Backend GET /home decides auth.
 * 401 → public landing. 503/other → unavailable retry state.
 */
export function decideHomeView(
  hasAuthCookie: boolean,
  homeResult:
    | { ok: true; data: AuthenticatedHomeData }
    | { ok: false; error: unknown },
): HomeViewDecision {
  if (!hasAuthCookie) {
    return { kind: "public" };
  }
  if (homeResult.ok) {
    return { kind: "authenticated", data: homeResult.data };
  }
  const err = homeResult.error;
  if (err instanceof ApiError && err.status === 401) {
    return { kind: "public" };
  }
  return { kind: "unavailable" };
}

export async function loadHomeDecision(
  hasAuthCookie: boolean,
  load: () => Promise<AuthenticatedHomeData>,
): Promise<HomeViewDecision> {
  if (!hasAuthCookie) {
    return { kind: "public" };
  }
  try {
    const data = await load();
    return { kind: "authenticated", data };
  } catch (error) {
    return decideHomeView(hasAuthCookie, { ok: false, error });
  }
}
