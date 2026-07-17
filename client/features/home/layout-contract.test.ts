import { describe, expect, it } from "vitest";
import {
  assertMainSectionContract,
  assertRightRailContract,
  DAILY_TODAY,
  DAILY_WEEK_DAYS,
  EN_LANDMARKS,
  GAME_MODE_KEYS,
  MAIN_SECTION_ORDER,
  PLAY_MODE_KEYS,
  RECOMMENDED_MAP_KEYS,
  RIGHT_RAIL_SECTION_ORDER,
} from "./layout-contract";

describe("authenticated home layout contract (home-loggedin.png)", () => {
  it("defines four play modes and five main sections in design order", () => {
    const main = assertMainSectionContract();
    expect(main.playModeCount).toBe(4);
    expect(PLAY_MODE_KEYS).toEqual([
      "singleplayer",
      "multiplayer",
      "party",
      "quiz",
    ]);
    expect(main.mainSections).toEqual([
      "welcome",
      "playModes",
      "premiumBanner",
      "recommended",
      "gameModes",
    ]);
    expect(MAIN_SECTION_ORDER[0]).toBe("welcome");
    expect(MAIN_SECTION_ORDER.at(-1)).toBe("gameModes");
  });

  it("defines recommended maps and three game mode rows", () => {
    const main = assertMainSectionContract();
    expect(main.mapCount).toBe(4);
    expect(RECOMMENDED_MAP_KEYS).toHaveLength(4);
    expect(main.gameModeCount).toBe(3);
    expect(GAME_MODE_KEYS).toEqual(["battle", "duels", "team"]);
  });

  it("documents game-mode visual layout: top pair + full-width team", () => {
    // Design: Battle | Duels on first row; Team Duels full-width second row.
    expect(GAME_MODE_KEYS[0]).toBe("battle");
    expect(GAME_MODE_KEYS[1]).toBe("duels");
    expect(GAME_MODE_KEYS[2]).toBe("team");
  });

  it("defines right-rail widgets and daily calendar week", () => {
    const rail = assertRightRailContract();
    expect(rail.railSections).toEqual([
      "dailyChallenge",
      "yourStats",
      "friendsOnline",
    ]);
    expect(RIGHT_RAIL_SECTION_ORDER).toHaveLength(3);
    expect(rail.weekDays).toEqual([13, 14, 15, 16, 17, 18, 19]);
    expect(rail.today).toBe(14);
    expect(DAILY_WEEK_DAYS).toContain(DAILY_TODAY);
  });

  it("exposes English landmarks used by smoke coverage", () => {
    expect(EN_LANDMARKS.welcome).toMatch(/Welcome back/);
    expect(EN_LANDMARKS.playModes).toHaveLength(4);
    expect(EN_LANDMARKS.premium).toMatch(/Subscribe/);
    expect(EN_LANDMARKS.recommended).toBe("Recommended for you");
    expect(EN_LANDMARKS.gameModes).toBe("Game Modes");
    expect(EN_LANDMARKS.daily).toBe("Daily Challenge");
    expect(EN_LANDMARKS.stats).toBe("Your Stats");
    expect(EN_LANDMARKS.friends).toBe("Friends Online");
  });
});
