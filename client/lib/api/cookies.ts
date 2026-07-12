import "server-only";

import { cookies } from "next/headers";
import {
  parseSetCookieHeader,
  type ParsedCookie,
} from "@/lib/api/cookie-parse";

const AUTH_COOKIE_NAMES = new Set([
  "access_token",
  "refresh_token",
  "csrf_token",
  "guest_session",
]);

function collectSetCookieHeaders(response: Response): string[] {
  if (typeof response.headers.getSetCookie === "function") {
    const values = response.headers.getSetCookie();
    if (values.length > 0) return values;
  }

  const single = response.headers.get("set-cookie");
  return single ? [single] : [];
}

export function authCookiesFromResponse(response: Response): ParsedCookie[] {
  return collectSetCookieHeaders(response)
    .map(parseSetCookieHeader)
    .filter(
      (cookie): cookie is ParsedCookie =>
        cookie !== null && AUTH_COOKIE_NAMES.has(cookie.name),
    );
}

export function authCookieOptions(cookie: ParsedCookie) {
  return {
    httpOnly: cookie.name === "csrf_token" ? false : (cookie.httpOnly ?? true),
    secure: cookie.secure ?? process.env.NODE_ENV === "production",
    path: "/",
    sameSite: cookie.sameSite ?? ("lax" as const),
    maxAge: cookie.maxAge,
    expires: cookie.expires,
  };
}

export async function getBackendCookieHeader(): Promise<string> {
  const store = await cookies();
  return store
    .getAll()
    .filter((cookie) => AUTH_COOKIE_NAMES.has(cookie.name))
    .map((cookie) => `${cookie.name}=${cookie.value}`)
    .join("; ");
}

export function mergeCookieHeader(
  header: string,
  additions: ParsedCookie[],
): string {
  const values = new Map<string, string>();

  for (const part of header.split(";")) {
    const trimmed = part.trim();
    if (!trimmed) continue;
    const separator = trimmed.indexOf("=");
    if (separator <= 0) continue;
    values.set(trimmed.slice(0, separator), trimmed.slice(separator + 1));
  }

  for (const cookie of additions) {
    values.set(cookie.name, cookie.value);
  }

  return [...values.entries()]
    .map(([name, value]) => `${name}=${value}`)
    .join("; ");
}

/**
 * Forward backend Set-Cookie headers onto the Next.js response so the browser
 * receives HTTP-only session cookies from Server Actions (BFF pattern).
 *
 * Auth cookies are re-scoped to path `/` so they work on the Next origin.
 */
export async function forwardAuthCookies(response: Response): Promise<void> {
  const store = await cookies();

  for (const cookie of authCookiesFromResponse(response)) {
    store.set(cookie.name, cookie.value, authCookieOptions(cookie));
  }
}
