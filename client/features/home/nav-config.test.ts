import { describe, expect, it } from "vitest";
import {
  AUTHENTICATED_SIDE_NAV,
  AUTHENTICATED_TOP_NAV,
} from "./nav-config";

describe("authenticated nav config (backend-backed)", () => {
  it("lists top-nav titles only for implemented API domains", () => {
    const ids = AUTHENTICATED_TOP_NAV.map((i) => i.id);
    expect(ids).toEqual([
      "singleplayer",
      "multiplayer",
      "party",
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
    const backends = AUTHENTICATED_TOP_NAV.map((i) => i.backend);
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
    expect(AUTHENTICATED_SIDE_NAV.find((i) => i.id === "missions")?.backend).toBe(
      "challenges",
    );
  });
});
