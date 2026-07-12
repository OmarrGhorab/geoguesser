import { beforeEach, describe, expect, it, vi } from "vitest";
import { cookies } from "next/headers";
import { z } from "zod";
import { apiJson } from "@/lib/api/client";

vi.mock("next/headers", () => ({ cookies: vi.fn() }));
vi.mock("@/lib/env", () => ({
  getBackendApiUrl: () => "http://api:8080/api/v1",
}));

const cookieStore = {
  getAll: vi.fn(),
  set: vi.fn(),
};

describe("apiJson", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    cookieStore.getAll.mockReturnValue([]);
    vi.mocked(cookies).mockResolvedValue(cookieStore as never);
  });

  it("bootstraps double-submit CSRF and forwards auth cookies", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response('{"status":"ok"}', {
          status: 200,
          headers: {
            "content-type": "application/json",
            "set-cookie":
              "csrf_token=signed.csrf; Path=/; SameSite=Lax; Max-Age=3600",
          },
        }),
      )
      .mockResolvedValueOnce(
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
      );
    vi.stubGlobal("fetch", fetchMock);

    const schema = z.object({
      user: z.object({ id: z.string().uuid() }),
    });
    await apiJson("/auth/login", schema, {
      method: "POST",
      body: { email: "player@example.com", password: "password" },
      forwardCookies: true,
    });

    expect(fetchMock).toHaveBeenCalledTimes(2);
    const mutationInit = fetchMock.mock.calls[1]?.[1] as RequestInit;
    const mutationHeaders = new Headers(mutationInit.headers);
    expect(mutationHeaders.get("x-csrf-token")).toBe("signed.csrf");
    expect(mutationHeaders.get("cookie")).toContain("csrf_token=signed.csrf");
    expect(cookieStore.set).toHaveBeenCalledWith(
      "csrf_token",
      "signed.csrf",
      expect.any(Object),
    );
    expect(cookieStore.set).toHaveBeenCalledWith(
      "access_token",
      "access.jwt",
      expect.any(Object),
    );
  });

  it("validates successful response bodies", async () => {
    cookieStore.getAll.mockReturnValue([
      { name: "csrf_token", value: "signed.csrf" },
    ]);
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          Response.json({ user: { id: "not-a-uuid" } }, { status: 200 }),
        ),
    );

    await expect(
      apiJson(
        "/auth/login",
        z.object({ user: z.object({ id: z.string().uuid() }) }),
        { method: "POST", body: {} },
      ),
    ).rejects.toBeInstanceOf(z.ZodError);
  });
});
