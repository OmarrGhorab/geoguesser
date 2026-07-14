import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { GET } from "./route";

vi.mock("@/lib/env", () => ({
  getBackendApiUrl: () => "http://api:8080/api/v1",
}));

describe("OAuth initiation route", () => {
  beforeEach(() => vi.restoreAllMocks());
  afterEach(() => {
    delete process.env.NEXT_PUBLIC_APP_URL;
  });

  it("proxies the internal backend redirect and preserves locale", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(null, {
        status: 307,
        headers: {
          location: "https://accounts.google.com/o/oauth2/v2/auth?state=abc",
          "set-cookie": "csrf_token=signed.csrf; Path=/; SameSite=Lax",
        },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const response = await GET(
      new NextRequest("http://localhost:3000/api/auth/oauth/google?locale=ar"),
      { params: Promise.resolve({ provider: "google" }) },
    );

    expect(fetchMock).toHaveBeenCalledWith(
      "http://api:8080/api/v1/auth/oauth/google",
      expect.any(Object),
    );
    expect(response.status).toBe(307);
    expect(response.headers.get("location")).toContain("accounts.google.com");
    expect(response.cookies.get("oauth_locale")?.value).toBe("ar");
    expect(response.cookies.get("csrf_token")?.value).toBe("signed.csrf");
  });

  it("rejects unsupported providers", async () => {
    const response = await GET(
      new NextRequest("http://localhost:3000/api/auth/oauth/github?locale=en"),
      { params: Promise.resolve({ provider: "github" }) },
    );

    expect(response.headers.get("location")).toBe(
      "http://localhost:3000/en/login?oauth_error=1",
    );
  });
});
