import "server-only";

import { NextResponse } from "next/server";
import { z } from "zod";
import { parseApiErrorBody } from "@/lib/api/errors";
import { getBackendApiUrl } from "@/lib/env";

type ProxyMethod = "POST" | "PUT" | "PATCH" | "DELETE";

export type ProxyMutationOptions<TSchema extends z.ZodType> = {
  path: string;
  method?: ProxyMethod;
  schema: TSchema;
  body?: unknown;
  /** Explicit key; falls back to the request Idempotency-Key header. */
  idempotencyKey?: string;
};

function cookieValue(header: string, name: string): string | undefined {
  for (const part of header.split(";")) {
    const trimmed = part.trim();
    if (trimmed.startsWith(`${name}=`)) {
      return trimmed.slice(name.length + 1);
    }
  }
  return undefined;
}

function mergeCookieHeader(header: string, setCookieLines: string[]): string {
  const values = new Map<string, string>();
  for (const part of header.split(";")) {
    const trimmed = part.trim();
    if (!trimmed) continue;
    const separator = trimmed.indexOf("=");
    if (separator <= 0) continue;
    values.set(trimmed.slice(0, separator), trimmed.slice(separator + 1));
  }
  for (const line of setCookieLines) {
    const pair = line.split(";")[0] ?? "";
    const separator = pair.indexOf("=");
    if (separator <= 0) continue;
    values.set(pair.slice(0, separator).trim(), pair.slice(separator + 1).trim());
  }
  return [...values.entries()]
    .map(([name, value]) => `${name}=${value}`)
    .join("; ");
}

function collectSetCookie(response: Response): string[] {
  if (typeof response.headers.getSetCookie === "function") {
    const values = response.headers.getSetCookie();
    if (values.length > 0) return values;
  }
  const single = response.headers.get("set-cookie");
  return single ? [single] : [];
}

function errorJson(
  status: number,
  code: string,
  message: string,
  extraHeaders?: HeadersInit,
): NextResponse {
  return NextResponse.json(
    { error: { code, message } },
    { status, headers: { "Cache-Control": "no-store", ...extraHeaders } },
  );
}

/**
 * Same-origin Route Handler helper: forward the browser cookie jar and CSRF
 * context to the Go API, validate the response with Zod, and return a safe
 * JSON error envelope. Never logs secrets or raw session material.
 */
export async function proxyMutation<TSchema extends z.ZodType>(
  request: Request,
  options: ProxyMutationOptions<TSchema>,
): Promise<NextResponse> {
  const method = options.method ?? "POST";
  const base = getBackendApiUrl();
  const path = options.path.startsWith("/") ? options.path : `/${options.path}`;
  const url = `${base}${path}`;

  let cookieHeader = request.headers.get("cookie") ?? "";
  let csrfToken =
    request.headers.get("x-csrf-token") ??
    cookieValue(cookieHeader, "csrf_token");

  if (!csrfToken) {
    const bootstrapHeaders = new Headers({ Accept: "application/json" });
    if (cookieHeader) bootstrapHeaders.set("Cookie", cookieHeader);
    const bootstrap = await fetch(`${base}/health`, {
      method: "GET",
      headers: bootstrapHeaders,
      cache: "no-store",
    });
    if (!bootstrap.ok) {
      return errorJson(
        502,
        "csrf_bootstrap_failed",
        "Unable to establish CSRF protection.",
      );
    }
    cookieHeader = mergeCookieHeader(cookieHeader, collectSetCookie(bootstrap));
    csrfToken = cookieValue(cookieHeader, "csrf_token");
    if (!csrfToken) {
      return errorJson(
        500,
        "csrf_bootstrap_failed",
        "The backend did not issue a CSRF token.",
      );
    }
  }

  const idempotencyKey =
    options.idempotencyKey?.trim() ||
    request.headers.get("idempotency-key")?.trim() ||
    undefined;

  const headers = new Headers({ Accept: "application/json" });
  if (cookieHeader) headers.set("Cookie", cookieHeader);
  headers.set("X-CSRF-Token", csrfToken);
  if (idempotencyKey) headers.set("Idempotency-Key", idempotencyKey);

  let body: string | undefined;
  if (options.body !== undefined) {
    body = JSON.stringify(options.body);
    headers.set("Content-Type", "application/json");
  }

  let upstream: Response;
  try {
    upstream = await fetch(url, {
      method,
      headers,
      body,
      cache: "no-store",
    });
  } catch {
    return errorJson(503, "dependency_unavailable", "Upstream service unavailable.");
  }

  if (upstream.status === 204) {
    try {
      const parsed = options.schema.parse(undefined);
      return NextResponse.json(parsed ?? null, {
        status: 204,
        headers: { "Cache-Control": "no-store" },
      });
    } catch {
      return errorJson(502, "internal_error", "Invalid empty response from API.");
    }
  }

  const text = await upstream.text();
  let json: unknown = null;
  if (text) {
    try {
      json = JSON.parse(text) as unknown;
    } catch {
      return errorJson(
        upstream.status >= 400 ? upstream.status : 502,
        "internal_error",
        "Invalid JSON response from API.",
      );
    }
  }

  if (!upstream.ok) {
    const apiError = parseApiErrorBody(json, upstream.status);
    const extra: Record<string, string> = {};
    if (upstream.status === 429) {
      const retryAfter = upstream.headers.get("retry-after");
      if (retryAfter) extra["Retry-After"] = retryAfter;
    }
    return errorJson(apiError.status, apiError.code, apiError.message, extra);
  }

  try {
    const data = options.schema.parse(json);
    return NextResponse.json(data, {
      status: upstream.status,
      headers: { "Cache-Control": "no-store" },
    });
  } catch {
    return errorJson(502, "internal_error", "Response failed schema validation.");
  }
}
