import { NextRequest, NextResponse } from "next/server";
import { authCookieOptions, authCookiesFromResponse } from "@/lib/api/cookies";
import { safeReturnPath } from "@/lib/api/session-refresh";
import { getBackendApiUrl } from "@/lib/env";
import { publicAppUrl } from "@/lib/public-origin";

const REFRESH_BACKOFF_COOKIE = "session_refresh_backoff";

function setRefreshBackoff(response: NextResponse, maxAge: number) {
  response.cookies.set(REFRESH_BACKOFF_COOKIE, maxAge > 0 ? "1" : "", {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge,
  });
}

export async function GET(request: NextRequest) {
  const returnTo = safeReturnPath(request.nextUrl.searchParams.get("returnTo"));
  const destination = publicAppUrl(request, returnTo);
  const cookieHeader = request.headers.get("cookie") ?? "";
  const csrfToken = request.cookies.get("csrf_token")?.value;
  const headers = new Headers({ Accept: "application/json" });

  if (cookieHeader) headers.set("Cookie", cookieHeader);
  if (csrfToken) headers.set("X-CSRF-Token", csrfToken);

  const upstream = await fetch(`${getBackendApiUrl()}/auth/refresh`, {
    method: "POST",
    headers,
    cache: "no-store",
  });
  const response = NextResponse.redirect(destination);

  if (!upstream.ok) {
    // Parallel RSC requests can present the same rotating refresh token. A
    // late 401 must not clear cookies issued by the request that won the race.
    setRefreshBackoff(response, 30);
    return response;
  }

  setRefreshBackoff(response, 0);
  for (const cookie of authCookiesFromResponse(upstream)) {
    response.cookies.set(cookie.name, cookie.value, authCookieOptions(cookie));
  }

  return response;
}
