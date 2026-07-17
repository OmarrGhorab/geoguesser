import createMiddleware from "next-intl/middleware";
import { NextRequest, NextResponse } from "next/server";
import { needsSessionRefresh } from "./lib/api/session-refresh";
import { routing } from "./lib/i18n/routing";
import { publicAppUrl } from "./lib/public-origin";

const handleI18nRouting = createMiddleware(routing);

export default function proxy(request: NextRequest) {
  if (
    needsSessionRefresh({
      method: request.method,
      accessToken: request.cookies.get("access_token")?.value,
      refreshToken: request.cookies.get("refresh_token")?.value,
      refreshBackoff: request.cookies.get("session_refresh_backoff")?.value,
    })
  ) {
    const refreshUrl = publicAppUrl(request, "/api/auth/session/refresh");
    refreshUrl.searchParams.set(
      "returnTo",
      `${request.nextUrl.pathname}${request.nextUrl.search}`,
    );
    return NextResponse.redirect(refreshUrl);
  }

  return handleI18nRouting(request);
}

export const config = {
  matcher: ["/((?!api|_next|.*\\..*).*)"],
};
