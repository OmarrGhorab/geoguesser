import "server-only";

import { z } from "zod";
import { getBackendApiUrl } from "@/lib/env";
import {
  authCookiesFromResponse,
  forwardAuthCookies,
  getBackendCookieHeader,
  mergeCookieHeader,
} from "@/lib/api/cookies";
import { ApiError, parseApiErrorBody } from "@/lib/api/errors";

type ApiFetchOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  idempotencyKey?: string;
  /** Retry once through the refresh-token endpoint when auth has expired. */
  requiresAuth?: boolean;
  /** When true, copy Set-Cookie from the backend onto the Next response. */
  forwardCookies?: boolean;
};

function isUnsafeMethod(method: string): boolean {
  return !["GET", "HEAD", "OPTIONS", "TRACE"].includes(method);
}

function cookieValue(header: string, name: string): string | undefined {
  for (const part of header.split(";")) {
    const trimmed = part.trim();
    if (trimmed.startsWith(`${name}=`)) {
      return trimmed.slice(name.length + 1);
    }
  }
  return undefined;
}

async function prepareBackendCookies(
  base: string,
  method: string,
  options: { ensureGuest?: boolean } = {},
): Promise<{ cookieHeader: string; csrfToken?: string }> {
  let cookieHeader = await getBackendCookieHeader();
  let csrfToken = cookieValue(cookieHeader, "csrf_token");

  // CSRF is only required for unsafe methods; skip health bootstrap otherwise.
  if (isUnsafeMethod(method) && !csrfToken) {
    const bootstrapHeaders = new Headers({ Accept: "application/json" });
    if (cookieHeader) bootstrapHeaders.set("Cookie", cookieHeader);

    const bootstrap = await fetch(`${base}/health`, {
      method: "GET",
      headers: bootstrapHeaders,
      cache: "no-store",
    });

    if (!bootstrap.ok) {
      throw new ApiError(bootstrap.status, {
        code: "csrf_bootstrap_failed",
        message: "Unable to establish CSRF protection.",
      });
    }

    const issuedCookies = authCookiesFromResponse(bootstrap);
    cookieHeader = mergeCookieHeader(cookieHeader, issuedCookies);
    csrfToken = cookieValue(cookieHeader, "csrf_token");
    await forwardAuthCookies(bootstrap);

    if (!csrfToken) {
      throw new ApiError(500, {
        code: "csrf_bootstrap_failed",
        message: "The backend did not issue a CSRF token.",
      });
    }
  }

  // Solo-family and other guest-capable endpoints need a guest_session when the
  // browser has neither an access token nor an existing guest cookie.
  const hasIdentity =
    Boolean(cookieValue(cookieHeader, "access_token")) ||
    Boolean(cookieValue(cookieHeader, "guest_session"));
  if (options.ensureGuest && !hasIdentity) {
    const guestHeaders = new Headers({ Accept: "application/json" });
    if (cookieHeader) guestHeaders.set("Cookie", cookieHeader);
    const me = await fetch(`${base}/auth/me`, {
      method: "GET",
      headers: guestHeaders,
      cache: "no-store",
    });
    if (me.ok) {
      const issued = authCookiesFromResponse(me);
      cookieHeader = mergeCookieHeader(cookieHeader, issued);
      csrfToken = cookieValue(cookieHeader, "csrf_token") ?? csrfToken;
      await forwardAuthCookies(me);
    }
  }

  return { cookieHeader, csrfToken };
}

export async function apiFetch(
  path: string,
  options: ApiFetchOptions = {},
): Promise<Response> {
  const base = getBackendApiUrl();
  const url = `${base}${path.startsWith("/") ? path : `/${path}`}`;
  const method = options.method ?? "GET";
  // Guest-capable writes (games create, daily attempts, etc.) must obtain a
  // guest_session before the first authorized call when the user is anonymous.
  const prepared = await prepareBackendCookies(base, method, {
    ensureGuest: Boolean(options.requiresAuth),
  });

  let body: string | undefined;
  if (options.body !== undefined) {
    body = JSON.stringify(options.body);
  }

  const send = (cookieHeader: string, csrfToken?: string) => {
    const headers = new Headers({ Accept: "application/json" });
    if (body !== undefined) headers.set("Content-Type", "application/json");
    if (cookieHeader) headers.set("Cookie", cookieHeader);
    if (options.idempotencyKey) {
      headers.set("Idempotency-Key", options.idempotencyKey);
    }
    if (isUnsafeMethod(method) && csrfToken) {
      headers.set("X-CSRF-Token", csrfToken);
    }
    return fetch(url, { method, headers, body, cache: "no-store" });
  };

  let response = await send(prepared.cookieHeader, prepared.csrfToken);
  let cookieHeader = prepared.cookieHeader;
  let csrfToken = prepared.csrfToken;

  if (
    options.requiresAuth &&
    (response.status === 401 || response.status === 403) &&
    cookieValue(cookieHeader, "refresh_token")
  ) {
    const refreshHeaders = new Headers({ Accept: "application/json" });
    refreshHeaders.set("Cookie", cookieHeader);
    if (csrfToken) {
      refreshHeaders.set("X-CSRF-Token", csrfToken);
    }
    const refreshResponse = await fetch(`${base}/auth/refresh`, {
      method: "POST",
      headers: refreshHeaders,
      cache: "no-store",
    });
    await forwardAuthCookies(refreshResponse);

    if (refreshResponse.ok) {
      cookieHeader = mergeCookieHeader(
        cookieHeader,
        authCookiesFromResponse(refreshResponse),
      );
      csrfToken = cookieValue(cookieHeader, "csrf_token") ?? csrfToken;
      response = await send(cookieHeader, csrfToken);
    }
  }

  // Always persist session cookies issued by the backend (guest, access, csrf).
  await forwardAuthCookies(response);

  return response;
}

export async function apiJson<TSchema extends z.ZodType>(
  path: string,
  schema: TSchema,
  options: ApiFetchOptions = {},
): Promise<z.output<TSchema>> {
  const response = await apiFetch(path, options);

  if (response.status === 204) {
    return schema.parse(undefined);
  }

  const text = await response.text();
  let json: unknown = null;
  if (text) {
    try {
      json = JSON.parse(text) as unknown;
    } catch {
      throw new ApiError(response.status, {
        code: "internal_error",
        message: "Invalid JSON response from API.",
      });
    }
  }

  if (!response.ok) {
    throw parseApiErrorBody(json, response.status);
  }

  return schema.parse(json);
}
