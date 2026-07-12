import { describe, expect, it } from "vitest";
import { parseSetCookieHeader } from "@/lib/api/cookie-parse";

describe("parseSetCookieHeader", () => {
  it("parses auth cookie attributes", () => {
    const parsed = parseSetCookieHeader(
      "access_token=abc123; Path=/; HttpOnly; Secure; SameSite=Lax; Max-Age=3600",
    );
    expect(parsed).toMatchObject({
      name: "access_token",
      value: "abc123",
      path: "/",
      httpOnly: true,
      secure: true,
      sameSite: "lax",
      maxAge: 3600,
    });
  });

  it("returns null for malformed headers", () => {
    expect(parseSetCookieHeader("")).toBeNull();
    expect(parseSetCookieHeader("=value")).toBeNull();
  });
});
