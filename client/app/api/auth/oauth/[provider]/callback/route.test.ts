import { describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { GET } from "./route";

vi.mock("@/lib/env", () => ({
  getBackendApiUrl: () => "http://api:8080/api/v1",
}));

describe("OAuth callback route", () => {
  it("forwards backend cookies and redirects to the localized frontend", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            user: {
              id: "019f568f-a9e1-73db-9e8f-1f71ebe683cd",
              email: "player@example.com",
              display_name: "Explorer",
              role: "user",
            },
          }),
          {
            status: 200,
            headers: {
              "content-type": "application/json",
              "set-cookie":
                "access_token=access.jwt; Path=/; HttpOnly; SameSite=Lax",
            },
          },
        ),
      ),
    );
    const request = new NextRequest(
      "http://localhost:3000/api/auth/oauth/google/callback?code=code&state=state",
      { headers: { cookie: "oauth_locale=ar" } },
    );

    const response = await GET(request, {
      params: Promise.resolve({ provider: "google" }),
    });

    expect(response.headers.get("location")).toBe("http://localhost:3000/ar");
    expect(response.cookies.get("access_token")?.value).toBe("access.jwt");
    expect(response.cookies.get("oauth_locale")?.value).toBe("");
  });

  it("redirects callback failures to localized login", async () => {
    const request = new NextRequest(
      "http://localhost:3000/api/auth/oauth/discord/callback?state=state",
      { headers: { cookie: "oauth_locale=ar" } },
    );

    const response = await GET(request, {
      params: Promise.resolve({ provider: "discord" }),
    });

    expect(response.headers.get("location")).toBe(
      "http://localhost:3000/ar/login?oauth_error=1",
    );
  });
});
