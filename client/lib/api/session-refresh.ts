const REFRESH_WINDOW_SECONDS = 30;

type RefreshDecision = Readonly<{
  method: string;
  accessToken?: string;
  refreshToken?: string;
  refreshBackoff?: string;
}>;

function accessTokenExpiresAt(token: string): number | null {
  const payload = token.split(".")[1];
  if (!payload) return null;

  try {
    const normalized = payload.replaceAll("-", "+").replaceAll("_", "/");
    const padded = normalized.padEnd(
      normalized.length + ((4 - (normalized.length % 4)) % 4),
      "=",
    );
    const claims = JSON.parse(atob(padded)) as { exp?: unknown };
    return typeof claims.exp === "number" ? claims.exp : null;
  } catch {
    return null;
  }
}

export function needsSessionRefresh(
  decision: RefreshDecision,
  nowSeconds = Math.floor(Date.now() / 1_000),
): boolean {
  if (
    (decision.method !== "GET" && decision.method !== "HEAD") ||
    !decision.refreshToken ||
    decision.refreshBackoff
  ) {
    return false;
  }

  if (!decision.accessToken) return true;

  const expiresAt = accessTokenExpiresAt(decision.accessToken);
  return expiresAt === null || expiresAt <= nowSeconds + REFRESH_WINDOW_SECONDS;
}

export function safeReturnPath(value: string | null): string {
  if (!value?.startsWith("/") || value.startsWith("//")) return "/";

  const parsed = new URL(value, "http://app.local");
  if (parsed.origin !== "http://app.local") return "/";
  return `${parsed.pathname}${parsed.search}`;
}
