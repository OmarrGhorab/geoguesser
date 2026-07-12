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
): Promise<{ cookieHeader: string; csrfToken?: string }> {
  let cookieHeader = await getBackendCookieHeader();
  let csrfToken = cookieValue(cookieHeader, "csrf_token");

  if (!isUnsafeMethod(method) || csrfToken) {
    return { cookieHeader, csrfToken };
  }

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

  return { cookieHeader, csrfToken };
}

export async function apiFetch(
  path: string,
  options: ApiFetchOptions = {},
): Promise<Response> {
  const base = getBackendApiUrl();
  const url = `${base}${path.startsWith("/") ? path : `/${path}`}`;
  const method = options.method ?? "GET";
  const { cookieHeader, csrfToken } = await prepareBackendCookies(base, method);

  const headers = new Headers({
    Accept: "application/json",
  });

  let body: string | undefined;
  if (options.body !== undefined) {
    headers.set("Content-Type", "application/json");
    body = JSON.stringify(options.body);
  }

  if (cookieHeader) headers.set("Cookie", cookieHeader);
  if (isUnsafeMethod(method) && csrfToken) {
    headers.set("X-CSRF-Token", csrfToken);
  }

  const response = await fetch(url, {
    method,
    headers,
    body,
    cache: "no-store",
  });

  if (options.forwardCookies) {
    await forwardAuthCookies(response);
  }

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
