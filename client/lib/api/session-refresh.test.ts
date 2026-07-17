import { describe, expect, it } from "vitest";
import { needsSessionRefresh, safeReturnPath } from "./session-refresh";

function accessToken(exp: number): string {
  const payload = btoa(JSON.stringify({ exp }))
    .replaceAll("+", "-")
    .replaceAll("/", "_")
    .replaceAll("=", "");
  return `header.${payload}.signature`;
}

describe("needsSessionRefresh", () => {
  it("refreshes a remembered session when the short access cookie is gone", () => {
    expect(
      needsSessionRefresh({ method: "GET", refreshToken: "refresh-token" }),
    ).toBe(true);
  });

  it("refreshes shortly before the access token expires", () => {
    expect(
      needsSessionRefresh(
        {
          method: "GET",
          refreshToken: "refresh-token",
          accessToken: accessToken(1_020),
        },
        1_000,
      ),
    ).toBe(true);
  });

  it("does not refresh a healthy access token or mutate POST requests", () => {
    const decision = {
      refreshToken: "refresh-token",
      accessToken: accessToken(2_000),
    };
    expect(needsSessionRefresh({ ...decision, method: "GET" }, 1_000)).toBe(
      false,
    );
    expect(needsSessionRefresh({ ...decision, method: "POST" }, 1_000)).toBe(
      false,
    );
  });

  it("backs off after a concurrent refresh already consumed the token", () => {
    expect(
      needsSessionRefresh({
        method: "GET",
        refreshToken: "rotated-by-another-request",
        refreshBackoff: "1",
      }),
    ).toBe(false);
  });
});

describe("safeReturnPath", () => {
  it("keeps local paths and rejects external redirects", () => {
    expect(safeReturnPath("/en/daily-mission?date=2026-07-17")).toBe(
      "/en/daily-mission?date=2026-07-17",
    );
    expect(safeReturnPath("https://example.com/steal")).toBe("/");
    expect(safeReturnPath("//example.com/steal")).toBe("/");
  });
});
