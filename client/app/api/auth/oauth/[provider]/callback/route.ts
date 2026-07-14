import { NextRequest, NextResponse } from "next/server";
import { oauthProviderSchema } from "@/features/auth/oauth";
import { authResponseSchema } from "@/features/auth/schemas";
import { authCookieOptions, authCookiesFromResponse } from "@/lib/api/cookies";
import { getBackendApiUrl } from "@/lib/env";
import { routing, type AppLocale } from "@/lib/i18n/routing";
import { publicAppUrl } from "@/lib/public-origin";

type OAuthCallbackContext = {
  params: Promise<{ provider: string }>;
};

function resolveLocale(request: NextRequest): AppLocale {
  const locale = request.cookies.get("oauth_locale")?.value;
  return routing.locales.includes(locale as AppLocale)
    ? (locale as AppLocale)
    : routing.defaultLocale;
}

function loginErrorRedirect(request: NextRequest, locale: AppLocale) {
  const response = NextResponse.redirect(
    publicAppUrl(request, `/${locale}/login?oauth_error=1`),
  );
  response.cookies.delete("oauth_locale");
  return response;
}

export async function GET(request: NextRequest, context: OAuthCallbackContext) {
  const locale = resolveLocale(request);
  const { provider: rawProvider } = await context.params;
  const provider = oauthProviderSchema.safeParse(rawProvider);
  const code = request.nextUrl.searchParams.get("code");
  const state = request.nextUrl.searchParams.get("state");

  if (!provider.success || !code || !state) {
    return loginErrorRedirect(request, locale);
  }

  const callbackUrl = new URL(
    `${getBackendApiUrl()}/auth/oauth/${provider.data}/callback`,
  );
  callbackUrl.searchParams.set("code", code);
  callbackUrl.searchParams.set("state", state);

  const upstream = await fetch(callbackUrl, {
    method: "GET",
    redirect: "manual",
    cache: "no-store",
    headers: { Accept: "application/json" },
  });

  if (!upstream.ok) return loginErrorRedirect(request, locale);

  const body = await upstream.json().catch(() => null);
  if (!authResponseSchema.safeParse(body).success) {
    return loginErrorRedirect(request, locale);
  }

  // Use the configured public app origin, not untrusted forwarded headers.
  const response = NextResponse.redirect(
    publicAppUrl(request, `/${locale}`),
  );
  response.cookies.delete("oauth_locale");

  for (const cookie of authCookiesFromResponse(upstream)) {
    response.cookies.set(cookie.name, cookie.value, authCookieOptions(cookie));
  }

  return response;
}
