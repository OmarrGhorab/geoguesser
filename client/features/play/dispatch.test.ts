import { describe, expect, it } from "vitest";
import {
  normalizeInboundMode,
  resolveGameDestination,
} from "./dispatch";

const gameId = "11111111-1111-4111-8111-111111111111";

describe("normalizeInboundMode", () => {
  it("maps legacy aliases one-way", () => {
    expect(normalizeInboundMode("private_room")).toBe("party_lobby");
    expect(normalizeInboundMode("ranked_standard")).toBe("ranked_solo");
    expect(normalizeInboundMode("ranked")).toBe("ranked_solo");
    expect(normalizeInboundMode("solo")).toBe("solo");
  });
});

describe("resolveGameDestination", () => {
  it("renders active solo-family modes", () => {
    expect(
      resolveGameDestination("en", {
        id: gameId,
        mode: "solo",
        status: "active",
      }),
    ).toEqual({ kind: "render_solo" });
    expect(
      resolveGameDestination("en", {
        id: gameId,
        mode: "quick_play",
        status: "active",
      }),
    ).toEqual({ kind: "render_quick_play" });
    expect(
      resolveGameDestination("en", {
        id: gameId,
        mode: "practice",
        status: "pending",
      }),
    ).toEqual({ kind: "render_practice" });
  });

  it("redirects completed solo and quick play to results", () => {
    expect(
      resolveGameDestination("en", {
        id: gameId,
        mode: "solo",
        status: "completed",
      }),
    ).toEqual({
      kind: "redirect",
      href: `/en/games/${gameId}/results`,
    });
    expect(
      resolveGameDestination("ar", {
        id: gameId,
        mode: "quick_play",
        status: "completed",
      }),
    ).toEqual({
      kind: "redirect",
      href: `/ar/games/${gameId}/results`,
    });
  });

  it("redirects daily to the semantic daily-mission play route", () => {
    expect(
      resolveGameDestination("en", {
        id: gameId,
        mode: "daily",
        status: "active",
      }),
    ).toEqual({
      kind: "redirect",
      href: `/en/daily-mission/play/${gameId}`,
    });
  });

  it("returns not_found for party lobby without a room code", () => {
    expect(
      resolveGameDestination("en", {
        id: gameId,
        mode: "private_room",
        status: "active",
      }),
    ).toEqual({ kind: "not_found" });
    expect(
      resolveGameDestination("en", {
        id: gameId,
        mode: "party_lobby",
        status: "active",
      }),
    ).toEqual({ kind: "not_found" });
  });

  it("returns not_found for active match modes without a match id", () => {
    expect(
      resolveGameDestination("en", {
        id: gameId,
        mode: "ranked_solo",
        status: "active",
      }),
    ).toEqual({ kind: "not_found" });
    expect(
      resolveGameDestination("en", {
        id: gameId,
        mode: "casual_duo",
        status: "active",
      }),
    ).toEqual({ kind: "not_found" });
  });

  it("redirects terminal ranked/casual game rows to generic results", () => {
    expect(
      resolveGameDestination("en", {
        id: gameId,
        mode: "ranked",
        status: "completed",
      }),
    ).toEqual({
      kind: "redirect",
      href: `/en/games/${gameId}/results`,
    });
  });

  it("returns not_found for unsupported modes", () => {
    expect(
      resolveGameDestination("en", {
        id: gameId,
        mode: "unknown_mode",
        status: "active",
      }),
    ).toEqual({ kind: "not_found" });
  });
});
