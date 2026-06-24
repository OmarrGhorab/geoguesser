import "server-only";

import { cookies } from "next/headers";
import { env } from "@/lib/env";
import { ApiError, type ApiErrorDetail } from "./errors";

type ApiFetchOptions = Omit<RequestInit, "body"> & {
  body?: BodyInit | Record<string, unknown>;
};

export async function apiFetch<T>(path: string, options: ApiFetchOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  const cookieStore = await cookies();
  const cookieHeader = cookieStore.toString();

  if (cookieHeader) {
    headers.set("Cookie", cookieHeader);
  }

  let body = options.body;
  if (body && !(body instanceof FormData) && !(body instanceof URLSearchParams)) {
    headers.set("Content-Type", "application/json");
    body = JSON.stringify(body);
  }

  const response = await fetch(`${env.BACKEND_API_URL}${path}`, {
    ...options,
    headers,
    body,
  });

  if (!response.ok) {
    throw new ApiError(response.status, await readError(response));
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

async function readError(response: Response): Promise<ApiErrorDetail> {
  try {
    const payload = (await response.json()) as { error?: ApiErrorDetail };
    if (payload.error) {
      return payload.error;
    }
  } catch {
    // Fall through to a stable generic error.
  }

  return {
    code: "request_failed",
    message: "The request failed.",
  };
}
