import { beforeEach, describe, expect, it, vi } from "vitest";

const cookiesMock = vi.fn();
const fetchMock = vi.fn();

vi.mock("next/headers", () => ({
  cookies: () => cookiesMock(),
}));

vi.mock("@/lib/env", () => ({
  env: { BACKEND_API_URL: "http://backend.test/api/v1" },
}));

describe("GET /api/matchmaking/status", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.stubGlobal("fetch", fetchMock);
    cookiesMock.mockResolvedValue({
      toString: () => "access_token=abc; csrf_token=xyz",
    });
  });

  it("forwards cookies and status payload", async () => {
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ status: "not_queued", queue: null, match: null }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    const { GET } = await import("./route");
    const response = await GET();
    expect(response.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://backend.test/api/v1/matchmaking/status",
      expect.objectContaining({
        method: "GET",
        headers: { Cookie: "access_token=abc; csrf_token=xyz" },
        cache: "no-store",
      }),
    );
    const json = await response.json();
    expect(json.status).toBe("not_queued");
  });

  it("forwards Retry-After on 429", async () => {
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ error: { code: "rate_limited", message: "Too many" } }), {
        status: 429,
        headers: { "Content-Type": "application/json", "Retry-After": "12" },
      }),
    );
    const { GET } = await import("./route");
    const response = await GET();
    expect(response.status).toBe(429);
    expect(response.headers.get("Retry-After")).toBe("12");
  });

  it("maps backend outages to 503", async () => {
    fetchMock.mockRejectedValue(new Error("network down"));
    const { GET } = await import("./route");
    const response = await GET();
    expect(response.status).toBe(503);
    const json = await response.json();
    expect(json.error.code).toBe("matchmaking_unavailable");
  });
});
