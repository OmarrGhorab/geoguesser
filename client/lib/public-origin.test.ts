import { afterEach, describe, expect, it } from "vitest";
import { getPublicAppOrigin, publicAppUrl } from "./public-origin";

describe("getPublicAppOrigin", () => {
  afterEach(() => {
    delete process.env.NEXT_PUBLIC_APP_URL;
  });

  it("prefers NEXT_PUBLIC_APP_URL over the request host", () => {
    process.env.NEXT_PUBLIC_APP_URL =
      "https://janeen-composable-offishly.ngrok-free.dev/";
    const request = new Request(
      "https://localhost:3000/api/auth/oauth/callback",
    );

    expect(getPublicAppOrigin(request)).toBe(
      "https://janeen-composable-offishly.ngrok-free.dev",
    );
    expect(publicAppUrl(request, "/en").toString()).toBe(
      "https://janeen-composable-offishly.ngrok-free.dev/en",
    );
  });

  it("ignores untrusted forwarded headers when app URL is unset", () => {
    const request = new Request("http://localhost:3000/callback", {
      headers: {
        host: "localhost:3000",
        "x-forwarded-host": "janeen-composable-offishly.ngrok-free.dev",
        "x-forwarded-proto": "https",
      },
    });

    expect(getPublicAppOrigin(request)).toBe("http://localhost:3000");
  });

  it("rejects non-http configured origins", () => {
    process.env.NEXT_PUBLIC_APP_URL = "javascript:alert(1)";
    const request = new Request("http://localhost:3000/callback");

    expect(() => getPublicAppOrigin(request)).toThrow(
      "NEXT_PUBLIC_APP_URL must use http or https",
    );
  });

  it("falls back to request host for local development", () => {
    const request = new Request("http://localhost:3000/en");

    expect(getPublicAppOrigin(request)).toBe("http://localhost:3000");
  });
});
