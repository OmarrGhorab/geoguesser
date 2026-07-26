import { existsSync } from "node:fs";
import { describe, expect, it } from "vitest";
import ar from "@/messages/ar.json";
import en from "@/messages/en.json";
import {
  AUTHENTICATED_GAME_MODES,
  AUTHENTICATED_SIDE_NAV,
  AUTHENTICATED_TOP_NAV,
} from "./nav-config";

describe("authenticated nav config (backend-backed)", () => {
  it("lists top-nav titles only for implemented API domains", () => {
    const ids = AUTHENTICATED_TOP_NAV.map((i) => i.id);
    expect(ids).toEqual([
      "play",
      "challenges",
      "maps",
      "leaderboards",
      "friends",
    ]);
    // Removed non-backend design labels
    expect(ids).not.toContain("shop");
    expect(ids).not.toContain("merch");
    expect(ids).not.toContain("championship");
    expect(ids).not.toContain("quiz");
    expect(ids).not.toContain("community");
  });

  it("maps each top-nav item to a backend domain from routes.go", () => {
    const backends = AUTHENTICATED_TOP_NAV.flatMap((i) => i.backends);
    expect(backends).toEqual([
      "games",
      "matchmaking",
      "rooms",
      "challenges",
      "maps",
      "leaderboards",
      "friends",
    ]);
  });

  it("exposes every canonical player-facing backend game mode once", () => {
    const modes = AUTHENTICATED_GAME_MODES.map((mode) => mode.backendMode);

    expect(modes).toEqual([
      "solo",
      "practice",
      "daily",
      "quick_play",
      "casual_solo",
      "casual_duo",
      "casual_squad",
      "ranked_solo",
      "ranked_duo",
      "ranked_squad",
      "party_lobby",
    ]);
    expect(new Set(modes).size).toBe(modes.length);
    expect(modes).not.toContain("private_room");
    expect(modes).not.toContain("ranked");
    expect(modes).not.toContain("ranked_standard");
  });

  it("maps every mode to an extracted icon asset", () => {
    for (const mode of AUTHENTICATED_GAME_MODES) {
      expect(mode.asset).toMatch(
        /^\/authenticated-home\/game-modes\/[a-z-]+\.png$/,
      );
      expect(
        existsSync(new URL(`../../public${mode.asset}`, import.meta.url)),
      ).toBe(true);
    }
  });

  it("localizes every mode in English and Arabic", () => {
    for (const mode of AUTHENTICATED_GAME_MODES) {
      const english = en.AuthenticatedHome.gameModes.items[mode.id];
      const arabic = ar.AuthenticatedHome.gameModes.items[mode.id];

      expect(english.title).not.toBe("");
      expect(english.description).not.toBe("");
      expect(arabic.title).not.toBe("");
      expect(arabic.description).not.toBe("");
    }
  });

  it("maps sidebar items to profiles / friends / challenges", () => {
    const side = AUTHENTICATED_SIDE_NAV.map((i) => i.id);
    expect(side).toEqual([
      "home",
      "profile",
      "stats",
      "friends",
      "missions",
      "settings",
    ]);
    expect(
      AUTHENTICATED_SIDE_NAV.find((i) => i.id === "missions")?.backend,
    ).toBe("challenges");
  });
});
