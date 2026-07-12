import "server-only";

import { getBackendApiUrl } from "@/lib/env";
import { forwardAuthCookies } from "@/lib/api/cookies";
import { ApiError, parseApiErrorBody } from "@/lib/api/errors";

type ApiFetchOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  /** When true, copy Set-Cookie from the backend onto the Next response. */
  forwardCookies?: boolean;
};

export async function apiFetch(
  path: string,
  options: ApiFetchOptions = {},
): Promise<Response> {
  const base = getBackendApiUrl();
  const url = `${base}${path.startsWith("/") ? path : `/${path}`}`;

  const headers = new Headers({
    Accept: "application/json",
  });

  let body: string | undefined;
  if (options.body !== undefined) {
    headers.set("Content-Type", "application/json");
    body = JSON.stringify(options.body);
  }

  const response = await fetch(url, {
    method: options.method ?? "GET",
    headers,
    body,
    cache: "no-store",
  });

  if (options.forwardCookies) {
    await forwardAuthCookies(response);
  }

  return response;
}

export async function apiJson<T>(
  path: string,
  options: ApiFetchOptions = {},
): Promise<T> {
  const response = await apiFetch(path, options);

  if (response.status === 204) {
    return undefined as T;
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

  return json as T;
}
