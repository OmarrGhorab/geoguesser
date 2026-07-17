import { beforeEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { GET } from "./route";

describe("GET /api/auth/session/refresh", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    process.env.BACKEND_API_URL = "http://api:8080/api/v1";
    process.env.NEXT_PUBLIC_APP_URL = "https://worldguess.example";
  });

  it("rotates the remembered session and returns to the requested page", async () => {
    const upstream = new Response(JSON.stringify({ user: { id: "user-1" } }), {
      status: 200,
      headers: { "content-type": "application/json" },
    });
    upstream.headers.append(
      "set-cookie",
      "access_token=fresh.jwt; Path=/; HttpOnly; Secure; SameSite=Lax; Max-Age=900",
    );
    upstream.headers.append(
      "set-cookie",
      "refresh_token=rotated.jwt; Path=/; HttpOnly; Secure; SameSite=Lax; Max-Age=2592000",
    );
    const fetchMock = vi.fn().mockResolvedValue(upstream);
    vi.stubGlobal("fetch", fetchMock);

    const request = new NextRequest(
      "https://worldguess.example/api/auth/session/refresh?returnTo=%2Fen%2Fdaily-mission",
      {
        headers: {
          cookie: "refresh_token=old.jwt; csrf_token=signed.csrf",
        },
      },
    );
    const response = await GET(request);

    expect(response.status).toBe(307);
    expect(response.headers.get("location")).toBe(
      "https://worldguess.example/en/daily-mission",
    );
    expect(response.cookies.get("access_token")?.value).toBe("fresh.jwt");
    expect(response.cookies.get("refresh_token")?.value).toBe("rotated.jwt");

    const upstreamHeaders = new Headers(
      (fetchMock.mock.calls[0]?.[1] as RequestInit).headers,
    );
    expect(upstreamHeaders.get("cookie")).toContain("refresh_token=old.jwt");
    expect(upstreamHeaders.get("x-csrf-token")).toBe("signed.csrf");
  });

  it("backs off after a rejected rotation without allowing an open redirect", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(Response.json({ error: {} }, { status: 401 })),
    );
    const request = new NextRequest(
      "https://worldguess.example/api/auth/session/refresh?returnTo=https%3A%2F%2Fevil.example",
      { headers: { cookie: "refresh_token=invalid.jwt" } },
    );
    const response = await GET(request);

    expect(response.headers.get("location")).toBe(
      "https://worldguess.example/",
    );
    expect(response.cookies.get("refresh_token")).toBeUndefined();
    expect(response.cookies.get("session_refresh_backoff")?.value).toBe("1");
    expect(response.cookies.get("session_refresh_backoff")?.maxAge).toBe(30);
  });
});
