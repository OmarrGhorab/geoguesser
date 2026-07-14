import { NextRequest, NextResponse } from "next/server";
import { oauthProviderSchema } from "@/features/auth/oauth";
import { authCookieOptions, authCookiesFromResponse } from "@/lib/api/cookies";
import { getBackendApiUrl } from "@/lib/env";
import { routing, type AppLocale } from "@/lib/i18n/routing";
import { getPublicAppOrigin, publicAppUrl } from "@/lib/public-origin";

type OAuthStartContext = {
  params: Promise<{ provider: string }>;
};

function loginErrorRedirect(request: NextRequest, locale: AppLocale) {
  return NextResponse.redirect(
    publicAppUrl(request, `/${locale}/login?oauth_error=1`),
  );
}

export async function GET(request: NextRequest, context: OAuthStartContext) {
  const { provider: rawProvider } = await context.params;
  const provider = oauthProviderSchema.safeParse(rawProvider);
  const requestedLocale = request.nextUrl.searchParams.get("locale");
  const locale = routing.locales.includes(requestedLocale as AppLocale)
    ? (requestedLocale as AppLocale)
    : routing.defaultLocale;

  if (!provider.success) return loginErrorRedirect(request, locale);

  const upstream = await fetch(
    `${getBackendApiUrl()}/auth/oauth/${provider.data}`,
    {
      method: "GET",
      redirect: "manual",
      cache: "no-store",
      headers: { Accept: "application/json" },
    },
  );
  const location = upstream.headers.get("location");

  if (!location || upstream.status < 300 || upstream.status >= 400) {
    return loginErrorRedirect(request, locale);
  }

  const providerUrl = new URL(location);
  if (providerUrl.protocol !== "https:") {
    return loginErrorRedirect(request, locale);
  }

  const publicOrigin = getPublicAppOrigin(request);
  const response = NextResponse.redirect(providerUrl);
  response.cookies.set("oauth_locale", locale, {
    httpOnly: true,
    secure:
      publicOrigin.startsWith("https:") ||
      process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/api/auth/oauth",
    maxAge: 10 * 60,
  });

  for (const cookie of authCookiesFromResponse(upstream)) {
    response.cookies.set(cookie.name, cookie.value, authCookieOptions(cookie));
  }

  return response;
}
