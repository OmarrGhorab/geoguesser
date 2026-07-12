import "server-only";

import { cookies } from "next/headers";
import { parseSetCookieHeader } from "@/lib/api/cookie-parse";

function collectSetCookieHeaders(response: Response): string[] {
  if (typeof response.headers.getSetCookie === "function") {
    const values = response.headers.getSetCookie();
    if (values.length > 0) return values;
  }

  const single = response.headers.get("set-cookie");
  return single ? [single] : [];
}

/**
 * Forward backend Set-Cookie headers onto the Next.js response so the browser
 * receives HTTP-only session cookies from Server Actions (BFF pattern).
 *
 * Auth cookies are re-scoped to path `/` so they work on the Next origin.
 */
export async function forwardAuthCookies(response: Response): Promise<void> {
  const store = await cookies();
  const headers = collectSetCookieHeaders(response);

  for (const header of headers) {
    const parsed = parseSetCookieHeader(header);
    if (!parsed) continue;

    if (
      parsed.name !== "access_token" &&
      parsed.name !== "refresh_token" &&
      parsed.name !== "csrf_token" &&
      parsed.name !== "guest_session"
    ) {
      continue;
    }

    store.set(parsed.name, parsed.value, {
      httpOnly:
        parsed.name === "csrf_token" ? false : (parsed.httpOnly ?? true),
      secure: parsed.secure ?? process.env.NODE_ENV === "production",
      path: "/",
      sameSite: parsed.sameSite ?? "lax",
      maxAge: parsed.maxAge,
      expires: parsed.expires,
    });
  }
}
